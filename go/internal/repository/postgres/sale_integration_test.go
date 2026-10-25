//go:build integration

package postgres

// Test integrasi transaksi penjualan (POS).
//
// Membuktikan sifat yang hanya bisa dilihat terhadap PostgreSQL sungguhan:
//
//  1. Membuat penjualan mengurangi stok dan menerbitkan jurnal POSTED seimbang
//     dengan source_type 'SALE' dan source_id nota, semuanya atomik.
//  2. Pembatalan mengembalikan stok, menerbitkan jurnal pembalik, menandai
//     jurnal asli sebagai reversed, dan mengubah status nota menjadi VOID.
//  3. Penjualan wajib lunas: pembayaran sebagian ditolak, kelebihan bayar
//     dicatat sebagai kembalian dan debit kas tetap sebesar total nota
//     (tidak ada baris piutang).
//
// Test ini butuh PostgreSQL nyata. Kalau DATABASE_URL kosong/tidak merespon,
// test dilewati (skip). Data uji memakai penanda "UJI-SALE-" dan dibersihkan.

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"
	"go_backend/internal/usecase"
)

func TestSaleCreateDanVoidMenulisJurnalSeimbang(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	var categoryID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_product_categories ORDER BY id LIMIT 1`).Scan(&categoryID); err != nil {
		t.Skipf("tidak ada kategori produk, test dilewati: %v", err)
	}
	var supplierID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_suppliers ORDER BY id LIMIT 1`).Scan(&supplierID); err != nil {
		t.Skipf("tidak ada supplier, test dilewati: %v", err)
	}
	var unitID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_units ORDER BY id LIMIT 1`).Scan(&unitID); err != nil {
		t.Skipf("tidak ada satuan, test dilewati: %v", err)
	}
	var actorID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM s_users ORDER BY id LIMIT 1`).Scan(&actorID); err != nil {
		t.Skipf("tidak ada pengguna, test dilewati: %v", err)
	}
	var tunaiID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_payment_methods WHERE code = 'TUNAI' AND deleted_at IS NULL LIMIT 1`).Scan(&tunaiID); err != nil {
		t.Skipf("metode TUNAI belum ada, test dilewati: %v", err)
	}

	sku := fmt.Sprintf("UJI-SALE-%d", time.Now().UnixNano())
	var productID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_products
			(sku, name, category_id, supplier_id, selling_unit_id, purchase_unit_id,
			 purchase_price, selling_price, current_stock, avg_cost, is_active)
		VALUES ($1, 'Produk Uji Penjualan', $2, $3, $4, $4, 10000, 20000, 10, 10000, TRUE)
		RETURNING id`, sku, categoryID, supplierID, unitID).Scan(&productID); err != nil {
		t.Fatalf("gagal membuat produk uji: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM m_products WHERE id = $1`, productID)
	})

	coaRepo := NewSQLChartOfAccountRepository(pool)
	periodRepo := NewSQLAccountingPeriodRepository(pool)
	journalRepo := NewSQLJournalRepository(pool)
	settingsRepo := NewSQLAccountingSettingsRepository(pool)
	reportRepo := NewSQLAccountingReportRepository(pool)

	journalUC := usecase.NewJournalUsecase(journalRepo, coaRepo, periodRepo, settingsRepo, reportRepo)
	saleRepo := NewSQLSaleRepository(pool)
	memberRepo := NewSQLMemberRepository(pool)
	paymentRepo := NewSQLPaymentMethodRepository(pool)
	saleUC := usecase.NewSaleUsecase(saleRepo, memberRepo, paymentRepo, nil, journalUC, nil)

	sale, err := saleUC.CreateSale(ctx, domain.CreateSaleInput{
		MemberID: 0,
		TaxType:  domain.SaleTaxNonPPN,
		Items: []domain.SaleLineInput{
			{ProductID: productID, Qty: 2},
		},
		Payments: []domain.SalePaymentInput{
			{PaymentMethodID: tunaiID, Amount: 40000},
		},
		CreatedBy: actorID,
	})
	if err != nil {
		t.Fatalf("CreateSale gagal: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		if sale.JournalEntryID != nil {
			_, _ = pool.Exec(bg, `DELETE FROM t_journal_entry_lines WHERE journal_entry_id IN
				(SELECT id FROM t_journal_entries WHERE source_id = $1 OR reversal_of_id = $1 OR id = $2)`, sale.ID, *sale.JournalEntryID)
			_, _ = pool.Exec(bg, `DELETE FROM t_journal_entries WHERE source_id = $1 OR reversal_of_id = $1 OR id = $2`, sale.ID, *sale.JournalEntryID)
		}
		_, _ = pool.Exec(bg, `DELETE FROM t_sales WHERE id = $1`, sale.ID)
	})

	if sale.Status != domain.SaleStatusPosted {
		t.Fatalf("status nota harus POSTED, dapat %q", sale.Status)
	}
	if sale.JournalEntryID == nil {
		t.Fatal("nota harus memiliki jurnal")
	}
	if sale.TotalAmount != 40000 || sale.PaidAmount != 40000 {
		t.Fatalf("total/bayar harus 40000/40000, dapat %.2f/%.2f", sale.TotalAmount, sale.PaidAmount)
	}
	if sale.ChangeAmount != 0 {
		t.Fatalf("kembalian harus 0, dapat %.2f", sale.ChangeAmount)
	}
	if sale.CostTotal != 20000 {
		t.Fatalf("HPP harus 20000, dapat %.2f", sale.CostTotal)
	}

	// Stok berkurang 10 -> 8.
	var stok int
	if err := pool.QueryRow(ctx, `SELECT current_stock FROM m_products WHERE id = $1`, productID).Scan(&stok); err != nil {
		t.Fatalf("gagal membaca stok: %v", err)
	}
	if stok != 8 {
		t.Fatalf("stok harus 8, dapat %d", stok)
	}

	// Jurnal POSTED seimbang: Debit kas 40000 + Debit HPP 20000 = Kredit
	// pendapatan 40000 + Kredit persediaan 20000 = 60000.
	var status string
	var totalDebit, totalCredit float64
	var sourceID int64
	if err := pool.QueryRow(ctx, `
		SELECT status, total_debit, total_credit, source_id
		FROM t_journal_entries WHERE id = $1`, *sale.JournalEntryID).
		Scan(&status, &totalDebit, &totalCredit, &sourceID); err != nil {
		t.Fatalf("jurnal penjualan tidak ditemukan: %v", err)
	}
	if status != "POSTED" {
		t.Fatalf("status jurnal harus POSTED, dapat %q", status)
	}
	if totalDebit != 60000 || totalCredit != 60000 {
		t.Fatalf("jurnal harus 60000/60000, dapat %.2f/%.2f", totalDebit, totalCredit)
	}
	if sourceID != sale.ID {
		t.Fatalf("source_id harus %d, dapat %d", sale.ID, sourceID)
	}

	// Pembatalan: stok pulih, jurnal pembalik terbit, jurnal asli ditandai.
	if err := saleUC.VoidSale(ctx, sale.ID, actorID, "uji pembatalan"); err != nil {
		t.Fatalf("VoidSale gagal: %v", err)
	}

	if err := pool.QueryRow(ctx, `SELECT current_stock FROM m_products WHERE id = $1`, productID).Scan(&stok); err != nil {
		t.Fatalf("gagal membaca stok setelah void: %v", err)
	}
	if stok != 10 {
		t.Fatalf("stok harus pulih ke 10, dapat %d", stok)
	}

	var voidStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM t_sales WHERE id = $1`, sale.ID).Scan(&voidStatus); err != nil {
		t.Fatalf("gagal membaca status nota: %v", err)
	}
	if voidStatus != domain.SaleStatusVoid {
		t.Fatalf("status nota harus VOID, dapat %q", voidStatus)
	}

	var reversedBy *int64
	if err := pool.QueryRow(ctx, `SELECT reversed_by_id FROM t_journal_entries WHERE id = $1`, *sale.JournalEntryID).Scan(&reversedBy); err != nil {
		t.Fatalf("gagal membaca reversed_by_id: %v", err)
	}
	if reversedBy == nil {
		t.Fatal("jurnal asli harus ditandai reversed_by_id")
	}

	var reversalDebit, reversalCredit float64
	if err := pool.QueryRow(ctx, `
		SELECT total_debit, total_credit FROM t_journal_entries
		WHERE reversal_of_id = $1`, *sale.JournalEntryID).
		Scan(&reversalDebit, &reversalCredit); err != nil {
		t.Fatalf("jurnal pembalik tidak ditemukan: %v", err)
	}
	if reversalDebit != 60000 || reversalCredit != 60000 {
		t.Fatalf("jurnal pembalik harus 60000/60000, dapat %.2f/%.2f", reversalDebit, reversalCredit)
	}
}

func TestSaleTolakPembayaranSebagian(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	var categoryID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_product_categories ORDER BY id LIMIT 1`).Scan(&categoryID); err != nil {
		t.Skipf("tidak ada kategori produk, test dilewati: %v", err)
	}
	var supplierID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_suppliers ORDER BY id LIMIT 1`).Scan(&supplierID); err != nil {
		t.Skipf("tidak ada supplier, test dilewati: %v", err)
	}
	var unitID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_units ORDER BY id LIMIT 1`).Scan(&unitID); err != nil {
		t.Skipf("tidak ada satuan, test dilewati: %v", err)
	}
	var actorID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM s_users ORDER BY id LIMIT 1`).Scan(&actorID); err != nil {
		t.Skipf("tidak ada pengguna, test dilewati: %v", err)
	}
	var tunaiID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_payment_methods WHERE code = 'TUNAI' AND deleted_at IS NULL LIMIT 1`).Scan(&tunaiID); err != nil {
		t.Skipf("metode TUNAI belum ada, test dilewati: %v", err)
	}

	sku := fmt.Sprintf("UJI-SALE-PARTIAL-%d", time.Now().UnixNano())
	var productID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_products
			(sku, name, category_id, supplier_id, selling_unit_id, purchase_unit_id,
			 purchase_price, selling_price, current_stock, avg_cost, is_active)
		VALUES ($1, 'Produk Uji Bayar Kurang', $2, $3, $4, $4, 0, 10000, 5, 0, TRUE)
		RETURNING id`, sku, categoryID, supplierID, unitID).Scan(&productID); err != nil {
		t.Fatalf("gagal membuat produk uji: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM m_products WHERE id = $1`, productID)
	})

	coaRepo := NewSQLChartOfAccountRepository(pool)
	periodRepo := NewSQLAccountingPeriodRepository(pool)
	journalRepo := NewSQLJournalRepository(pool)
	settingsRepo := NewSQLAccountingSettingsRepository(pool)
	reportRepo := NewSQLAccountingReportRepository(pool)
	journalUC := usecase.NewJournalUsecase(journalRepo, coaRepo, periodRepo, settingsRepo, reportRepo)
	saleRepo := NewSQLSaleRepository(pool)
	memberRepo := NewSQLMemberRepository(pool)
	paymentRepo := NewSQLPaymentMethodRepository(pool)
	saleUC := usecase.NewSaleUsecase(saleRepo, memberRepo, paymentRepo, nil, journalUC, nil)

	_, err := saleUC.CreateSale(ctx, domain.CreateSaleInput{
		MemberID: 0,
		TaxType:  domain.SaleTaxNonPPN,
		Items: []domain.SaleLineInput{
			{ProductID: productID, Qty: 3},
		},
		Payments: []domain.SalePaymentInput{
			{PaymentMethodID: tunaiID, Amount: 10000},
		},
		CreatedBy: actorID,
	})
	if err != domain.ErrSalePaymentKurang {
		t.Fatalf("harus ErrSalePaymentKurang, dapat %v", err)
	}

	// Tidak ada nota/rincian yang tersimpan saat pembayaran ditolak.
	var rincianTercatat int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM t_sales_details WHERE product_id = $1`, productID).Scan(&rincianTercatat); err != nil {
		t.Fatalf("gagal menghitung rincian: %v", err)
	}
	if rincianTercatat != 0 {
		t.Fatalf("tidak boleh ada rincian nota, dapat %d", rincianTercatat)
	}
}

func TestSaleLebihBayarMencatatKembalianTanpaPiutang(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	var categoryID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_product_categories ORDER BY id LIMIT 1`).Scan(&categoryID); err != nil {
		t.Skipf("tidak ada kategori produk, test dilewati: %v", err)
	}
	var supplierID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_suppliers ORDER BY id LIMIT 1`).Scan(&supplierID); err != nil {
		t.Skipf("tidak ada supplier, test dilewati: %v", err)
	}
	var unitID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_units ORDER BY id LIMIT 1`).Scan(&unitID); err != nil {
		t.Skipf("tidak ada satuan, test dilewati: %v", err)
	}
	var actorID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM s_users ORDER BY id LIMIT 1`).Scan(&actorID); err != nil {
		t.Skipf("tidak ada pengguna, test dilewati: %v", err)
	}
	var tunaiID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_payment_methods WHERE code = 'TUNAI' AND deleted_at IS NULL LIMIT 1`).Scan(&tunaiID); err != nil {
		t.Skipf("metode TUNAI belum ada, test dilewati: %v", err)
	}

	sku := fmt.Sprintf("UJI-SALE-KEMBALI-%d", time.Now().UnixNano())
	var productID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_products
			(sku, name, category_id, supplier_id, selling_unit_id, purchase_unit_id,
			 purchase_price, selling_price, current_stock, avg_cost, is_active)
		VALUES ($1, 'Produk Uji Kembalian', $2, $3, $4, $4, 0, 10000, 5, 0, TRUE)
		RETURNING id`, sku, categoryID, supplierID, unitID).Scan(&productID); err != nil {
		t.Fatalf("gagal membuat produk uji: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM m_products WHERE id = $1`, productID)
	})

	coaRepo := NewSQLChartOfAccountRepository(pool)
	periodRepo := NewSQLAccountingPeriodRepository(pool)
	journalRepo := NewSQLJournalRepository(pool)
	settingsRepo := NewSQLAccountingSettingsRepository(pool)
	reportRepo := NewSQLAccountingReportRepository(pool)
	journalUC := usecase.NewJournalUsecase(journalRepo, coaRepo, periodRepo, settingsRepo, reportRepo)
	saleRepo := NewSQLSaleRepository(pool)
	memberRepo := NewSQLMemberRepository(pool)
	paymentRepo := NewSQLPaymentMethodRepository(pool)
	saleUC := usecase.NewSaleUsecase(saleRepo, memberRepo, paymentRepo, nil, journalUC, nil)

	sale, err := saleUC.CreateSale(ctx, domain.CreateSaleInput{
		MemberID: 0,
		TaxType:  domain.SaleTaxNonPPN,
		Items: []domain.SaleLineInput{
			{ProductID: productID, Qty: 3},
		},
		Payments: []domain.SalePaymentInput{
			{PaymentMethodID: tunaiID, Amount: 50000},
		},
		CreatedBy: actorID,
	})
	if err != nil {
		t.Fatalf("CreateSale gagal: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		if sale.JournalEntryID != nil {
			_, _ = pool.Exec(bg, `DELETE FROM t_journal_entry_lines WHERE journal_entry_id = $1`, *sale.JournalEntryID)
			_, _ = pool.Exec(bg, `DELETE FROM t_journal_entries WHERE id = $1`, *sale.JournalEntryID)
		}
		_, _ = pool.Exec(bg, `DELETE FROM t_sales WHERE id = $1`, sale.ID)
	})

	if sale.PaymentStatus != domain.SalePaymentPaid {
		t.Fatalf("status pembayaran harus PAID, dapat %q", sale.PaymentStatus)
	}
	if sale.ChangeAmount != 20000 {
		t.Fatalf("kembalian harus 20000, dapat %.2f", sale.ChangeAmount)
	}
	if sale.PaidAmount != 50000 {
		t.Fatalf("dibayar harus 50000, dapat %.2f", sale.PaidAmount)
	}

	var debitAR float64
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(debit),0) FROM t_journal_entry_lines
		WHERE journal_entry_id = $1 AND account_id = (SELECT account_id FROM s_accounting_settings WHERE mapping_key = 'AR')`,
		*sale.JournalEntryID).Scan(&debitAR); err != nil {
		t.Fatalf("gagal membaca debit piutang: %v", err)
	}
	if debitAR != 0 {
		t.Fatalf("debit piutang harus 0, dapat %.2f", debitAR)
	}

	// Debit kas hanya sebesar total nota (30000), bukan uang diterima (50000).
	var debitKas float64
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(l.debit),0) FROM t_journal_entry_lines l
		WHERE l.journal_entry_id = $1
		  AND l.account_id = (SELECT account_id FROM m_payment_methods WHERE id = $2)`,
		*sale.JournalEntryID, tunaiID).Scan(&debitKas); err != nil {
		t.Fatalf("gagal membaca debit kas: %v", err)
	}
	if debitKas != 30000 {
		t.Fatalf("debit kas harus 30000, dapat %.2f", debitKas)
	}

	var totalDebit, totalCredit float64
	if err := pool.QueryRow(ctx, `
		SELECT total_debit, total_credit FROM t_journal_entries WHERE id = $1`, *sale.JournalEntryID).
		Scan(&totalDebit, &totalCredit); err != nil {
		t.Fatalf("jurnal penjualan tidak ditemukan: %v", err)
	}
	if totalDebit != 30000 || totalCredit != 30000 {
		t.Fatalf("jurnal harus 30000/30000, dapat %.2f/%.2f", totalDebit, totalCredit)
	}
}

func TestSaleMemberPoinEarnRedeemDanVoid(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	var categoryID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_product_categories ORDER BY id LIMIT 1`).Scan(&categoryID); err != nil {
		t.Skipf("tidak ada kategori produk, test dilewati: %v", err)
	}
	var supplierID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_suppliers ORDER BY id LIMIT 1`).Scan(&supplierID); err != nil {
		t.Skipf("tidak ada supplier, test dilewati: %v", err)
	}
	var unitID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_units ORDER BY id LIMIT 1`).Scan(&unitID); err != nil {
		t.Skipf("tidak ada satuan, test dilewati: %v", err)
	}
	var actorID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM s_users ORDER BY id LIMIT 1`).Scan(&actorID); err != nil {
		t.Skipf("tidak ada pengguna, test dilewati: %v", err)
	}
	var tunaiID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_payment_methods WHERE code = 'TUNAI' AND deleted_at IS NULL LIMIT 1`).Scan(&tunaiID); err != nil {
		t.Skipf("metode TUNAI belum ada, test dilewati: %v", err)
	}

	memberCode := fmt.Sprintf("UJI-MEMBER-%d", time.Now().UnixNano())
	var memberID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_members (code, name, points, is_active)
		VALUES ($1, 'Member Uji Poin', 20, TRUE) RETURNING id`, memberCode).Scan(&memberID); err != nil {
		t.Fatalf("gagal membuat member uji: %v", err)
	}

	sku := fmt.Sprintf("UJI-SALE-POIN-%d", time.Now().UnixNano())
	var productID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_products
			(sku, name, category_id, supplier_id, selling_unit_id, purchase_unit_id,
			 purchase_price, selling_price, current_stock, avg_cost, is_active)
		VALUES ($1, 'Produk Uji Poin', $2, $3, $4, $4, 50000, 100000, 10, 50000, TRUE)
		RETURNING id`, sku, categoryID, supplierID, unitID).Scan(&productID); err != nil {
		t.Fatalf("gagal membuat produk uji: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM t_member_points WHERE member_id = $1`, memberID)
		_, _ = pool.Exec(bg, `DELETE FROM m_products WHERE id = $1`, productID)
		_, _ = pool.Exec(bg, `DELETE FROM m_members WHERE id = $1`, memberID)
	})

	coaRepo := NewSQLChartOfAccountRepository(pool)
	periodRepo := NewSQLAccountingPeriodRepository(pool)
	journalRepo := NewSQLJournalRepository(pool)
	settingsRepo := NewSQLAccountingSettingsRepository(pool)
	reportRepo := NewSQLAccountingReportRepository(pool)
	journalUC := usecase.NewJournalUsecase(journalRepo, coaRepo, periodRepo, settingsRepo, reportRepo)
	saleRepo := NewSQLSaleRepository(pool)
	memberRepo := NewSQLMemberRepository(pool)
	paymentRepo := NewSQLPaymentMethodRepository(pool)
	saleUC := usecase.NewSaleUsecase(saleRepo, memberRepo, paymentRepo, nil, journalUC, nil)

	sale, err := saleUC.CreateSale(ctx, domain.CreateSaleInput{
		MemberID:     memberID,
		TaxType:      domain.SaleTaxNonPPN,
		RedeemPoints: 20,
		Items: []domain.SaleLineInput{
			{ProductID: productID, Qty: 1},
		},
		Payments: []domain.SalePaymentInput{
			{PaymentMethodID: tunaiID, Amount: 98000},
		},
		CreatedBy: actorID,
	})
	if err != nil {
		t.Fatalf("CreateSale gagal: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		if sale.JournalEntryID != nil {
			_, _ = pool.Exec(bg, `DELETE FROM t_journal_entry_lines WHERE journal_entry_id IN
				(SELECT id FROM t_journal_entries WHERE source_id = $1 OR reversal_of_id = $1 OR id = $2)`, sale.ID, *sale.JournalEntryID)
			_, _ = pool.Exec(bg, `DELETE FROM t_journal_entries WHERE source_id = $1 OR reversal_of_id = $1 OR id = $2`, sale.ID, *sale.JournalEntryID)
		}
		_, _ = pool.Exec(bg, `DELETE FROM t_sales WHERE id = $1`, sale.ID)
	})

	if sale.RedeemPoints != 20 || sale.RedeemAmount != 2000 {
		t.Fatalf("tukar poin harus 20/2000, dapat %d/%.2f", sale.RedeemPoints, sale.RedeemAmount)
	}
	if sale.TotalAmount != 98000 {
		t.Fatalf("total harus 98000, dapat %.2f", sale.TotalAmount)
	}
	if sale.PointsEarned != 9 {
		t.Fatalf("poin perolehan harus 9, dapat %d", sale.PointsEarned)
	}

	assertMemberPoints := func(want int64) {
		var got int64
		if err := pool.QueryRow(ctx, `SELECT points FROM m_members WHERE id = $1`, memberID).Scan(&got); err != nil {
			t.Fatalf("gagal membaca saldo poin: %v", err)
		}
		if got != want {
			t.Fatalf("saldo poin harus %d, dapat %d", want, got)
		}
	}
	assertMemberPoints(9)

	// Buku besar: REDEEM -20 (saldo 0) lalu EARN +9 (saldo 9).
	var tipe string
	var poin, saldo int64
	rows, err := pool.Query(ctx, `SELECT entry_type, points, balance_after FROM t_member_points WHERE sale_id = $1 ORDER BY id`, sale.ID)
	if err != nil {
		t.Fatalf("gagal membaca buku besar: %v", err)
	}
	type ent struct {
		tipe  string
		poin  int64
		saldo int64
	}
	var entries []ent
	for rows.Next() {
		if err := rows.Scan(&tipe, &poin, &saldo); err != nil {
			t.Fatalf("gagal scan buku besar: %v", err)
		}
		entries = append(entries, ent{tipe, poin, saldo})
	}
	rows.Close()
	if len(entries) != 2 || entries[0].tipe != domain.MemberPointRedeem || entries[0].poin != -20 || entries[0].saldo != 0 ||
		entries[1].tipe != domain.MemberPointEarn || entries[1].poin != 9 || entries[1].saldo != 9 {
		t.Fatalf("buku besar tidak sesuai: %+v", entries)
	}

	// Void: poin perolehan ditarik dan poin tukar dikembalikan -> saldo pulih 20.
	if err := saleUC.VoidSale(ctx, sale.ID, actorID, "uji void poin"); err != nil {
		t.Fatalf("VoidSale gagal: %v", err)
	}
	assertMemberPoints(20)

	var jml int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM t_member_points WHERE sale_id = $1`, sale.ID).Scan(&jml); err != nil {
		t.Fatalf("gagal menghitung buku besar: %v", err)
	}
	if jml != 4 {
		t.Fatalf("buku besar harus 4 baris setelah void, dapat %d", jml)
	}
}

// TestSalePajakPerBarisTersimpanDanJurnalSeimbang memverifikasi pajak per-baris
// produk: baris kena PPN memisahkan DPP & PPN, baris tanpa PPN masuk DPP penuh,
// nilai tersimpan di t_sales_details, dan jurnal tetap seimbang (Pendapatan+PPN).
func TestSalePajakPerBarisTersimpanDanJurnalSeimbang(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	var categoryID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_product_categories ORDER BY id LIMIT 1`).Scan(&categoryID); err != nil {
		t.Skipf("tidak ada kategori produk, test dilewati: %v", err)
	}
	var supplierID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_suppliers ORDER BY id LIMIT 1`).Scan(&supplierID); err != nil {
		t.Skipf("tidak ada supplier, test dilewati: %v", err)
	}
	var unitID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_units ORDER BY id LIMIT 1`).Scan(&unitID); err != nil {
		t.Skipf("tidak ada satuan, test dilewati: %v", err)
	}
	var actorID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM s_users ORDER BY id LIMIT 1`).Scan(&actorID); err != nil {
		t.Skipf("tidak ada pengguna, test dilewati: %v", err)
	}
	var tunaiID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_payment_methods WHERE code = 'TUNAI' AND deleted_at IS NULL LIMIT 1`).Scan(&tunaiID); err != nil {
		t.Skipf("metode TUNAI belum ada, test dilewati: %v", err)
	}

	baseSKU := fmt.Sprintf("UJI-PAJAK-%d", time.Now().UnixNano())
	var taxableID, nonTaxableID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_products
			(sku, name, category_id, supplier_id, selling_unit_id, purchase_unit_id,
			 purchase_price, selling_price, current_stock, avg_cost, is_taxable, is_active)
		VALUES ($1, 'Produk Uji Kena PPN', $2, $3, $4, $4, 0, 112000, 5, 0, TRUE, TRUE)
		RETURNING id`, baseSKU+"-T", categoryID, supplierID, unitID).Scan(&taxableID); err != nil {
		t.Fatalf("gagal membuat produk kena PPN: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_products
			(sku, name, category_id, supplier_id, selling_unit_id, purchase_unit_id,
			 purchase_price, selling_price, current_stock, avg_cost, is_taxable, is_active)
		VALUES ($1, 'Produk Uji Tanpa PPN', $2, $3, $4, $4, 0, 100000, 5, 0, FALSE, TRUE)
		RETURNING id`, baseSKU+"-N", categoryID, supplierID, unitID).Scan(&nonTaxableID); err != nil {
		t.Fatalf("gagal membuat produk tanpa PPN: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM t_sales_details WHERE product_id IN ($1, $2)`, taxableID, nonTaxableID)
		_, _ = pool.Exec(bg, `DELETE FROM m_products WHERE id IN ($1, $2)`, taxableID, nonTaxableID)
	})

	coaRepo := NewSQLChartOfAccountRepository(pool)
	periodRepo := NewSQLAccountingPeriodRepository(pool)
	journalRepo := NewSQLJournalRepository(pool)
	settingsRepo := NewSQLAccountingSettingsRepository(pool)
	reportRepo := NewSQLAccountingReportRepository(pool)
	journalUC := usecase.NewJournalUsecase(journalRepo, coaRepo, periodRepo, settingsRepo, reportRepo)
	saleRepo := NewSQLSaleRepository(pool)
	memberRepo := NewSQLMemberRepository(pool)
	paymentRepo := NewSQLPaymentMethodRepository(pool)
	saleUC := usecase.NewSaleUsecase(saleRepo, memberRepo, paymentRepo, nil, journalUC, nil)

	// Harga kena PPN 112000 (inclusive -> DPP 100000 + PPN 12000) dan tanpa PPN
	// 100000 (DPP penuh). Total nota 212000.
	sale, err := saleUC.CreateSale(ctx, domain.CreateSaleInput{
		MemberID: 0,
		Items: []domain.SaleLineInput{
			{ProductID: taxableID, Qty: 1},
			{ProductID: nonTaxableID, Qty: 1},
		},
		Payments: []domain.SalePaymentInput{
			{PaymentMethodID: tunaiID, Amount: 212000},
		},
		CreatedBy: actorID,
	})
	if err != nil {
		t.Fatalf("CreateSale gagal: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		if sale.JournalEntryID != nil {
			_, _ = pool.Exec(bg, `DELETE FROM t_journal_entry_lines WHERE journal_entry_id IN
				(SELECT id FROM t_journal_entries WHERE source_id = $1 OR reversal_of_id = $1 OR id = $2)`, sale.ID, *sale.JournalEntryID)
			_, _ = pool.Exec(bg, `DELETE FROM t_journal_entries WHERE source_id = $1 OR reversal_of_id = $1 OR id = $2`, sale.ID, *sale.JournalEntryID)
		}
		_, _ = pool.Exec(bg, `DELETE FROM t_sales WHERE id = $1`, sale.ID)
	})

	if sale.TaxType != domain.SaleTaxInclusive {
		t.Fatalf("tax_type harus TAX_INCLUSIVE, dapat %q", sale.TaxType)
	}
	if sale.DppAmount != 200000 || sale.TaxAmount != 12000 || sale.TotalAmount != 212000 {
		t.Fatalf("DPP/PPN/total harus 200000/12000/212000, dapat %.2f/%.2f/%.2f",
			sale.DppAmount, sale.TaxAmount, sale.TotalAmount)
	}

	// Nilai pajak per baris tersimpan.
	var dppBaris, taxBaris float64
	if err := pool.QueryRow(ctx, `
		SELECT dpp_amount, tax_amount FROM t_sales_details
		WHERE sale_id = $1 AND product_id = $2`, sale.ID, taxableID).Scan(&dppBaris, &taxBaris); err != nil {
		t.Fatalf("gagal membaca rincian baris kena PPN: %v", err)
	}
	if dppBaris != 100000 || taxBaris != 12000 {
		t.Fatalf("baris kena PPN harus DPP 100000 / PPN 12000, dapat %.2f/%.2f", dppBaris, taxBaris)
	}
	if err := pool.QueryRow(ctx, `
		SELECT dpp_amount, tax_amount FROM t_sales_details
		WHERE sale_id = $1 AND product_id = $2`, sale.ID, nonTaxableID).Scan(&dppBaris, &taxBaris); err != nil {
		t.Fatalf("gagal membaca rincian baris tanpa PPN: %v", err)
	}
	if dppBaris != 100000 || taxBaris != 0 {
		t.Fatalf("baris tanpa PPN harus DPP 100000 / PPN 0, dapat %.2f/%.2f", dppBaris, taxBaris)
	}

	// Jurnal seimbang: Pendapatan = DPP total, PPN Keluaran = pajak.
	var debitKas, kreditPendapatan, kreditPPN float64
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(l.debit),0) FROM t_journal_entry_lines l
		WHERE l.journal_entry_id = $1
		  AND l.account_id = (SELECT account_id FROM m_payment_methods WHERE id = $2)`,
		*sale.JournalEntryID, tunaiID).Scan(&debitKas); err != nil {
		t.Fatalf("gagal membaca debit kas: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(l.credit),0) FROM t_journal_entry_lines l
		WHERE l.journal_entry_id = $1
		  AND l.account_id = (SELECT account_id FROM s_accounting_settings WHERE mapping_key = 'SALES_REVENUE')`,
		*sale.JournalEntryID).Scan(&kreditPendapatan); err != nil {
		t.Fatalf("gagal membaca kredit pendapatan: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(l.credit),0) FROM t_journal_entry_lines l
		WHERE l.journal_entry_id = $1
		  AND l.account_id = (SELECT account_id FROM s_accounting_settings WHERE mapping_key = 'VAT_OUTPUT')`,
		*sale.JournalEntryID).Scan(&kreditPPN); err != nil {
		t.Fatalf("gagal membaca kredit PPN: %v", err)
	}
	if debitKas != 212000 {
		t.Fatalf("debit kas harus 212000, dapat %.2f", debitKas)
	}
	if kreditPendapatan != 200000 {
		t.Fatalf("kredit pendapatan harus 200000, dapat %.2f", kreditPendapatan)
	}
	if kreditPPN != 12000 {
		t.Fatalf("kredit PPN keluaran harus 12000, dapat %.2f", kreditPPN)
	}
}

func TestSaleStaleSnapshotDitolakTanpaMengubahStok(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	var categoryID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_product_categories ORDER BY id LIMIT 1`).Scan(&categoryID); err != nil {
		t.Skipf("tidak ada kategori produk, test dilewati: %v", err)
	}
	var supplierID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_suppliers ORDER BY id LIMIT 1`).Scan(&supplierID); err != nil {
		t.Skipf("tidak ada supplier, test dilewati: %v", err)
	}
	var unitID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_units ORDER BY id LIMIT 1`).Scan(&unitID); err != nil {
		t.Skipf("tidak ada satuan, test dilewati: %v", err)
	}
	var actorID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM s_users ORDER BY id LIMIT 1`).Scan(&actorID); err != nil {
		t.Skipf("tidak ada pengguna, test dilewati: %v", err)
	}

	saleRepo := NewSQLSaleRepository(pool)
	cases := []struct {
		name   string
		mutate string
	}{
		{"harga berubah", `UPDATE m_products SET selling_price = 25000 WHERE id = $1`},
		{"hpp berubah", `UPDATE m_products SET avg_cost = 12000 WHERE id = $1`},
		{"diskon produk berubah", `UPDATE m_products SET discount_type = 'PERCENT', discount_value = 10 WHERE id = $1`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sku := fmt.Sprintf("UJI-SNAPSHOT-%d", time.Now().UnixNano())
			var productID int64
			if err := pool.QueryRow(ctx, `
				INSERT INTO m_products
					(sku, name, category_id, supplier_id, selling_unit_id, purchase_unit_id,
					 purchase_price, selling_price, current_stock, avg_cost, is_active)
				VALUES ($1, 'Produk Uji Snapshot', $2, $3, $4, $4, 10000, 20000, 5, 10000, TRUE)
				RETURNING id`, sku, categoryID, supplierID, unitID).Scan(&productID); err != nil {
				t.Fatalf("gagal membuat produk uji: %v", err)
			}
			t.Cleanup(func() {
				bg := context.Background()
				_, _ = pool.Exec(bg, `DELETE FROM t_sales WHERE id IN (SELECT sale_id FROM t_sales_details WHERE product_id = $1)`, productID)
				_, _ = pool.Exec(bg, `DELETE FROM m_products WHERE id = $1`, productID)
			})

			products, err := saleRepo.GetProductsForSale(ctx, []int64{productID})
			if err != nil {
				t.Fatalf("GetProductsForSale gagal: %v", err)
			}
			sale, err := domain.CalculateSale(domain.CreateSaleInput{
				Items: []domain.SaleLineInput{{ProductID: productID, Qty: 1}},
				Payments: []domain.SalePaymentInput{
					{PaymentMethodID: 1, Amount: 20000},
				},
			}, products, domain.TaxConfig{Rate: 0, Mode: domain.TaxModeExclusive})
			if err != nil {
				t.Fatalf("CalculateSale gagal: %v", err)
			}
			createdBy := actorID
			sale.CreatedBy = &createdBy

			if _, err := pool.Exec(ctx, tc.mutate, productID); err != nil {
				t.Fatalf("gagal mengubah produk: %v", err)
			}
			_, err = saleRepo.CreateSale(ctx, sale, nil)
			if !errors.Is(err, domain.ErrSaleSnapshotChanged) {
				t.Fatalf("harus ErrSaleSnapshotChanged, dapat %v", err)
			}

			var stock int
			if err := pool.QueryRow(ctx, `SELECT current_stock FROM m_products WHERE id = $1`, productID).Scan(&stock); err != nil {
				t.Fatalf("gagal membaca stok: %v", err)
			}
			if stock != 5 {
				t.Fatalf("stok tidak boleh berubah, dapat %d", stock)
			}
			var detailCount int
			if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM t_sales_details WHERE product_id = $1`, productID).Scan(&detailCount); err != nil {
				t.Fatalf("gagal menghitung rincian: %v", err)
			}
			if detailCount != 0 {
				t.Fatalf("tidak boleh ada rincian nota, dapat %d", detailCount)
			}
		})
	}
}

func TestSaleVoidMengembalikanNilaiPersediaanRataRata(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	var categoryID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_product_categories ORDER BY id LIMIT 1`).Scan(&categoryID); err != nil {
		t.Skipf("tidak ada kategori produk, test dilewati: %v", err)
	}
	var supplierID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_suppliers ORDER BY id LIMIT 1`).Scan(&supplierID); err != nil {
		t.Skipf("tidak ada supplier, test dilewati: %v", err)
	}
	var unitID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_units ORDER BY id LIMIT 1`).Scan(&unitID); err != nil {
		t.Skipf("tidak ada satuan, test dilewati: %v", err)
	}
	var actorID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM s_users ORDER BY id LIMIT 1`).Scan(&actorID); err != nil {
		t.Skipf("tidak ada pengguna, test dilewati: %v", err)
	}
	var tunaiID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_payment_methods WHERE code = 'TUNAI' AND deleted_at IS NULL LIMIT 1`).Scan(&tunaiID); err != nil {
		t.Skipf("metode TUNAI belum ada, test dilewati: %v", err)
	}

	sku := fmt.Sprintf("UJI-VOID-VALUASI-%d", time.Now().UnixNano())
	var productID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_products
			(sku, name, category_id, supplier_id, selling_unit_id, purchase_unit_id,
			 purchase_price, selling_price, current_stock, avg_cost, is_active)
		VALUES ($1, 'Produk Uji Void Valuasi', $2, $3, $4, $4, 10000, 20000, 10, 10000, TRUE)
		RETURNING id`, sku, categoryID, supplierID, unitID).Scan(&productID); err != nil {
		t.Fatalf("gagal membuat produk uji: %v", err)
	}

	saleRepo := NewSQLSaleRepository(pool)
	memberRepo := NewSQLMemberRepository(pool)
	paymentRepo := NewSQLPaymentMethodRepository(pool)
	saleUC := usecase.NewSaleUsecase(saleRepo, memberRepo, paymentRepo, nil, nil, nil)

	sale, err := saleUC.CreateSale(ctx, domain.CreateSaleInput{
		Items: []domain.SaleLineInput{{ProductID: productID, Qty: 2}},
		Payments: []domain.SalePaymentInput{
			{PaymentMethodID: tunaiID, Amount: 40000},
		},
		CreatedBy: actorID,
	})
	if err != nil {
		t.Fatalf("CreateSale gagal: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM t_sales WHERE id = $1`, sale.ID)
		_, _ = pool.Exec(bg, `DELETE FROM m_products WHERE id = $1`, productID)
	})
	if sale.CostTotal != 20000 {
		t.Fatalf("HPP nota harus 20000, dapat %.2f", sale.CostTotal)
	}

	// Simulasikan perubahan nilai persediaan setelah penjualan (mis. penerimaan
	// barang baru): 10 unit @ 12000. Void harus mengembalikan 2 unit pada HPP
	// asli 10000, sehingga rata-rata menjadi (120000+20000)/12.
	if _, err := pool.Exec(ctx, `UPDATE m_products SET current_stock = 10, avg_cost = 12000 WHERE id = $1`, productID); err != nil {
		t.Fatalf("gagal menyiapkan nilai persediaan: %v", err)
	}
	if err := saleUC.VoidSale(ctx, sale.ID, actorID, "uji valuasi void"); err != nil {
		t.Fatalf("VoidSale gagal: %v", err)
	}

	var stock int
	var avgCost float64
	if err := pool.QueryRow(ctx, `SELECT current_stock, avg_cost FROM m_products WHERE id = $1`, productID).Scan(&stock, &avgCost); err != nil {
		t.Fatalf("gagal membaca stok/avg_cost: %v", err)
	}
	if stock != 12 {
		t.Fatalf("stok setelah void harus 12, dapat %d", stock)
	}
	wantAvg := 140000.0 / 12.0
	if math.Abs(avgCost-wantAvg) > 0.0001 {
		t.Fatalf("avg_cost setelah void harus %.4f, dapat %.4f", wantAvg, avgCost)
	}
}

func TestSaleVoucherSekaliPakaiDanDilepasSaatVoid(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	var categoryID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_product_categories ORDER BY id LIMIT 1`).Scan(&categoryID); err != nil {
		t.Skipf("tidak ada kategori produk, test dilewati: %v", err)
	}
	var supplierID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_suppliers ORDER BY id LIMIT 1`).Scan(&supplierID); err != nil {
		t.Skipf("tidak ada supplier, test dilewati: %v", err)
	}
	var unitID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_units ORDER BY id LIMIT 1`).Scan(&unitID); err != nil {
		t.Skipf("tidak ada satuan, test dilewati: %v", err)
	}
	var actorID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM s_users ORDER BY id LIMIT 1`).Scan(&actorID); err != nil {
		t.Skipf("tidak ada pengguna, test dilewati: %v", err)
	}
	var tunaiID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_payment_methods WHERE code = 'TUNAI' AND deleted_at IS NULL LIMIT 1`).Scan(&tunaiID); err != nil {
		t.Skipf("metode TUNAI belum ada, test dilewati: %v", err)
	}

	nano := time.Now().UnixNano()
	sku := fmt.Sprintf("UJI-SALE-VCR-%d", nano)
	var productID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_products
			(sku, name, category_id, supplier_id, selling_unit_id, purchase_unit_id,
			 purchase_price, selling_price, current_stock, avg_cost, is_active)
		VALUES ($1, 'Produk Uji Voucher', $2, $3, $4, $4, 10000, 20000, 10, 10000, TRUE)
		RETURNING id`, sku, categoryID, supplierID, unitID).Scan(&productID); err != nil {
		t.Fatalf("gagal membuat produk uji: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM m_products WHERE id = $1`, productID)
	})

	voucherRepo := NewSQLVoucherRepository(pool)
	voucher, err := voucherRepo.Create(ctx, domain.VoucherInput{
		Code:          fmt.Sprintf("UJI-VCR-%d", nano),
		Name:          "Voucher Uji",
		DiscountType:  domain.VoucherDiscountAmount,
		DiscountValue: 5000,
		IsActive:      true,
	}, actorID)
	if err != nil {
		t.Fatalf("gagal membuat voucher uji: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM m_vouchers WHERE id = $1`, voucher.ID)
	})

	coaRepo := NewSQLChartOfAccountRepository(pool)
	periodRepo := NewSQLAccountingPeriodRepository(pool)
	journalRepo := NewSQLJournalRepository(pool)
	settingsRepo := NewSQLAccountingSettingsRepository(pool)
	reportRepo := NewSQLAccountingReportRepository(pool)
	journalUC := usecase.NewJournalUsecase(journalRepo, coaRepo, periodRepo, settingsRepo, reportRepo)

	saleRepo := NewSQLSaleRepository(pool)
	saleUC := usecase.NewSaleUsecase(saleRepo, NewSQLMemberRepository(pool), NewSQLPaymentMethodRepository(pool), voucherRepo, journalUC, nil)

	sale, err := saleUC.CreateSale(ctx, domain.CreateSaleInput{
		TaxType:     domain.SaleTaxNonPPN,
		VoucherCode: "  " + voucher.Code + " ",
		Items: []domain.SaleLineInput{
			{ProductID: productID, Qty: 1},
		},
		Payments: []domain.SalePaymentInput{
			{PaymentMethodID: tunaiID, Amount: 15000},
		},
		CreatedBy: actorID,
	})
	if err != nil {
		t.Fatalf("CreateSale dengan voucher gagal: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		if sale.JournalEntryID != nil {
			_, _ = pool.Exec(bg, `DELETE FROM t_journal_entry_lines WHERE journal_entry_id IN
				(SELECT id FROM t_journal_entries WHERE source_id = $1 OR reversal_of_id = $1 OR id = $2)`, sale.ID, *sale.JournalEntryID)
			_, _ = pool.Exec(bg, `DELETE FROM t_journal_entries WHERE source_id = $1 OR reversal_of_id = $1 OR id = $2`, sale.ID, *sale.JournalEntryID)
		}
		_, _ = pool.Exec(bg, `DELETE FROM t_sales WHERE id = $1`, sale.ID)
	})

	if sale.VoucherID != voucher.ID || sale.VoucherCode != voucher.Code {
		t.Fatalf("voucher nota harus %d/%s, dapat %d/%s", voucher.ID, voucher.Code, sale.VoucherID, sale.VoucherCode)
	}
	if sale.VoucherDiscount != 5000 || sale.TotalAmount != 15000 {
		t.Fatalf("diskon/total harus 5000/15000, dapat %.2f/%.2f", sale.VoucherDiscount, sale.TotalAmount)
	}

	var usedAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT used_at FROM m_vouchers WHERE id = $1`, voucher.ID).Scan(&usedAt); err != nil {
		t.Fatalf("gagal membaca used_at: %v", err)
	}
	if usedAt == nil {
		t.Fatal("voucher harus ditandai terpakai setelah nota dibuat")
	}

	_, err = saleUC.CreateSale(ctx, domain.CreateSaleInput{
		TaxType:     domain.SaleTaxNonPPN,
		VoucherCode: voucher.Code,
		Items: []domain.SaleLineInput{
			{ProductID: productID, Qty: 1},
		},
		Payments: []domain.SalePaymentInput{
			{PaymentMethodID: tunaiID, Amount: 15000},
		},
		CreatedBy: actorID,
	})
	if !errors.Is(err, domain.ErrVoucherUsed) {
		t.Fatalf("pemakaian ulang harus ErrVoucherUsed, dapat %v", err)
	}

	if err := saleUC.VoidSale(ctx, sale.ID, actorID, "uji voucher"); err != nil {
		t.Fatalf("VoidSale gagal: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT used_at FROM m_vouchers WHERE id = $1`, voucher.ID).Scan(&usedAt); err != nil {
		t.Fatalf("gagal membaca used_at setelah void: %v", err)
	}
	if usedAt != nil {
		t.Fatal("voucher harus dilepas setelah nota dibatalkan")
	}
}

// TestSaleConcurrentLastItemHanyaSatuBerhasil meniru dua kasir yang menjual
// unit terakhir pada saat bersamaan. Penguncian baris produk dan penjaga
// current_stock >= qty di SQL harus membuat tepat satu nota berhasil; nota
// yang kalah ditolak ErrSaleStokTidakCukup dan seluruh tulisannya digulung.
func TestSaleConcurrentLastItemHanyaSatuBerhasil(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	var categoryID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_product_categories ORDER BY id LIMIT 1`).Scan(&categoryID); err != nil {
		t.Skipf("tidak ada kategori produk, test dilewati: %v", err)
	}
	var supplierID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_suppliers ORDER BY id LIMIT 1`).Scan(&supplierID); err != nil {
		t.Skipf("tidak ada supplier, test dilewati: %v", err)
	}
	var unitID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_units ORDER BY id LIMIT 1`).Scan(&unitID); err != nil {
		t.Skipf("tidak ada satuan, test dilewati: %v", err)
	}
	var actorID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM s_users ORDER BY id LIMIT 1`).Scan(&actorID); err != nil {
		t.Skipf("tidak ada pengguna, test dilewati: %v", err)
	}
	var tunaiID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_payment_methods WHERE code = 'TUNAI' AND deleted_at IS NULL LIMIT 1`).Scan(&tunaiID); err != nil {
		t.Skipf("metode TUNAI belum ada, test dilewati: %v", err)
	}

	sku := fmt.Sprintf("UJI-SALE-RACE-%d", time.Now().UnixNano())
	var productID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_products
			(sku, name, category_id, supplier_id, selling_unit_id, purchase_unit_id,
			 purchase_price, selling_price, current_stock, avg_cost, is_active)
		VALUES ($1, 'Produk Uji Race POS', $2, $3, $4, $4, 10000, 20000, 1, 10000, TRUE)
		RETURNING id`, sku, categoryID, supplierID, unitID).Scan(&productID); err != nil {
		t.Fatalf("gagal membuat produk uji: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM t_sales WHERE id IN
			(SELECT sale_id FROM t_sales_details WHERE product_id = $1)`, productID)
		_, _ = pool.Exec(bg, `DELETE FROM m_products WHERE id = $1`, productID)
	})

	saleRepo := NewSQLSaleRepository(pool)
	memberRepo := NewSQLMemberRepository(pool)
	paymentRepo := NewSQLPaymentMethodRepository(pool)
	saleUC := usecase.NewSaleUsecase(saleRepo, memberRepo, paymentRepo, nil, nil, nil)

	const penjual = 2
	errs := make(chan error, penjual)
	var wg sync.WaitGroup
	for i := 0; i < penjual; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := saleUC.CreateSale(ctx, domain.CreateSaleInput{
				Items:     []domain.SaleLineInput{{ProductID: productID, Qty: 1}},
				Payments:  []domain.SalePaymentInput{{PaymentMethodID: tunaiID, Amount: 20000}},
				CreatedBy: actorID,
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	var berhasil, stokKurang int
	for err := range errs {
		switch {
		case err == nil:
			berhasil++
		case errors.Is(err, domain.ErrSaleStokTidakCukup):
			stokKurang++
		default:
			t.Fatalf("error tak terduga dari penjualan bersamaan: %v", err)
		}
	}
	if berhasil != 1 || stokKurang != 1 {
		t.Fatalf("harus tepat 1 berhasil dan 1 ditolak stok, dapat berhasil=%d ditolak=%d", berhasil, stokKurang)
	}

	var stok int
	if err := pool.QueryRow(ctx, `SELECT current_stock FROM m_products WHERE id = $1`, productID).Scan(&stok); err != nil {
		t.Fatalf("gagal membaca stok: %v", err)
	}
	if stok != 0 {
		t.Fatalf("stok akhir harus 0 (tidak oversell), dapat %d", stok)
	}

	var jumlahNota int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT sale_id) FROM t_sales_details WHERE product_id = $1`,
		productID).Scan(&jumlahNota); err != nil {
		t.Fatalf("gagal menghitung nota: %v", err)
	}
	if jumlahNota != 1 {
		t.Fatalf("nota yang tersimpan harus 1, dapat %d (nota gagal harusnya rollback)", jumlahNota)
	}
}

// TestDecrementStockMenolakQtyTidakPositif membuktikan penjaga SQL tidak
// mengubah stok untuk qty 0/negatif meskipun pemanggil melewati validasi
// repository. Pemanggil mendeteksinya lewat expected_count != updated_count.
func TestDecrementStockMenolakQtyTidakPositif(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	var categoryID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_product_categories ORDER BY id LIMIT 1`).Scan(&categoryID); err != nil {
		t.Skipf("tidak ada kategori produk, test dilewati: %v", err)
	}
	var supplierID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM m_suppliers ORDER BY id LIMIT 1`).Scan(&supplierID); err != nil {
		t.Skipf("tidak ada supplier, test dilewati: %v", err)
	}
	var unitID int32
	if err := pool.QueryRow(ctx, `SELECT id FROM m_units ORDER BY id LIMIT 1`).Scan(&unitID); err != nil {
		t.Skipf("tidak ada satuan, test dilewati: %v", err)
	}

	sku := fmt.Sprintf("UJI-SALE-QTY-%d", time.Now().UnixNano())
	var productID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_products
			(sku, name, category_id, supplier_id, selling_unit_id, purchase_unit_id,
			 purchase_price, selling_price, current_stock, avg_cost, is_active)
		VALUES ($1, 'Produk Uji Qty', $2, $3, $4, $4, 10000, 20000, 10, 10000, TRUE)
		RETURNING id`, sku, categoryID, supplierID, unitID).Scan(&productID); err != nil {
		t.Fatalf("gagal membuat produk uji: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM m_products WHERE id = $1`, productID)
	})

	q := db.New(pool)
	for _, qty := range []int32{0, -1, -5} {
		applied, err := q.DecrementProductStocksForSale(ctx, db.DecrementProductStocksForSaleParams{
			ProductIds: []int64{productID},
			Qtys:       []int32{qty},
		})
		if err != nil {
			t.Fatalf("qty %d tidak boleh error SQL: %v", qty, err)
		}
		if applied.ExpectedCount != 1 || applied.UpdatedCount != 0 {
			t.Fatalf("qty %d harus expected=1 updated=0, dapat %d/%d", qty, applied.ExpectedCount, applied.UpdatedCount)
		}
		var stock int
		if err := pool.QueryRow(ctx, `SELECT current_stock FROM m_products WHERE id = $1`, productID).Scan(&stock); err != nil {
			t.Fatalf("gagal membaca stok: %v", err)
		}
		if stock != 10 {
			t.Fatalf("stok harus tetap 10 untuk qty %d, dapat %d", qty, stock)
		}
	}

	applied, err := q.DecrementProductStocksForSale(ctx, db.DecrementProductStocksForSaleParams{
		ProductIds: []int64{productID},
		Qtys:       []int32{1},
	})
	if err != nil {
		t.Fatalf("qty valid tidak boleh error: %v", err)
	}
	if applied.ExpectedCount != 1 || applied.UpdatedCount != 1 {
		t.Fatalf("qty valid harus expected=1 updated=1, dapat %d/%d", applied.ExpectedCount, applied.UpdatedCount)
	}
	var stock int
	if err := pool.QueryRow(ctx, `SELECT current_stock FROM m_products WHERE id = $1`, productID).Scan(&stock); err != nil {
		t.Fatalf("gagal membaca stok: %v", err)
	}
	if stock != 9 {
		t.Fatalf("stok harus 9 setelah decrement valid, dapat %d", stock)
	}
}
