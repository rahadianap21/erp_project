package postgres

// Test integrasi auto-posting akuntansi dari penerimaan barang (GRN).
//
// Membuktikan jaminan yang hanya bisa dilihat terhadap PostgreSQL sungguhan:
//
//  1. Penutupan penerimaan menerbitkan jurnal POSTED per GRN dengan akun yang
//     diambil dari pemetaan s_accounting_settings.
//  2. Jurnal seimbang (debit = kredit) dan memakai source_id GRN.
//  3. Penerbitan ulang untuk GRN yang sama tidak menambah jurnal (idempoten),
//     sehingga penutupan penerimaan yang diulang tidak mencatat persediaan dua
//     kali.
//
// Test ini butuh PostgreSQL nyata. Kalau DATABASE_URL kosong/tidak merespon,
// test dilewati (skip). Data uji memakai penanda "uji-grn-" dan dihapus di
// t.Cleanup.

import (
	"context"
	"testing"

	"go_backend/internal/usecase"
)

func TestAutoPostingGRNMenerbitkanJurnalIdempoten(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	// PO NON_PPN, unit_price 1000, qty 4 => persediaan/hutang 4000 tanpa PPN.
	s := siapkanSkenarioGRN(t, ctx, pool, 4, 1)

	grnRepo := NewSQLGoodsReceiptNoteRepository(pool)
	poRepo := NewSQLPurchaseOrderRepository(pool)
	coaRepo := NewSQLChartOfAccountRepository(pool)
	periodRepo := NewSQLAccountingPeriodRepository(pool)
	journalRepo := NewSQLJournalRepository(pool)
	settingsRepo := NewSQLAccountingSettingsRepository(pool)
	reportRepo := NewSQLAccountingReportRepository(pool)

	mapping, err := settingsRepo.ListMapping(ctx)
	if err != nil {
		t.Fatalf("ListMapping gagal: %v", err)
	}
	inventoryID, ok := mapping["INVENTORY"]
	if !ok {
		t.Fatal("pemetaan INVENTORY belum ada di s_accounting_settings")
	}
	apID, ok := mapping["AP"]
	if !ok {
		t.Fatal("pemetaan AP belum ada di s_accounting_settings")
	}

	journalUC := usecase.NewJournalUsecase(journalRepo, coaRepo, periodRepo, settingsRepo, reportRepo)
	grnUC := usecase.NewGoodsReceiptNoteUsecase(grnRepo, poRepo, journalUC)

	grn, err := grnRepo.CreateGRN(ctx, s.input(4))
	if err != nil {
		t.Fatalf("CreateGRN gagal: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM t_journal_entry_lines WHERE journal_entry_id IN
			(SELECT id FROM t_journal_entries WHERE source_type = 'GRN' AND source_id = $1)`, grn.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM t_journal_entries WHERE source_type = 'GRN' AND source_id = $1`, grn.ID)
	})

	if err := grnUC.FinishReceiving(ctx, s.poID, false, 0); err != nil {
		t.Fatalf("FinishReceiving gagal: %v", err)
	}

	var journalID int64
	var status string
	var totalDebit, totalCredit float64
	var sourceID int64
	if err := pool.QueryRow(ctx, `
		SELECT id, status, total_debit, total_credit, source_id
		FROM t_journal_entries
		WHERE source_type = 'GRN' AND source_id = $1`, grn.ID).
		Scan(&journalID, &status, &totalDebit, &totalCredit, &sourceID); err != nil {
		t.Fatalf("jurnal GRN tidak ditemukan: %v", err)
	}
	if status != "POSTED" {
		t.Fatalf("status jurnal harus POSTED, dapat %q", status)
	}
	if totalDebit != 4000 || totalCredit != 4000 {
		t.Fatalf("jurnal harus 4000/4000, dapat %.2f/%.2f", totalDebit, totalCredit)
	}
	if sourceID != grn.ID {
		t.Fatalf("source_id harus %d, dapat %d", grn.ID, sourceID)
	}

	// Dua baris: Debit Persediaan, Kredit Hutang Usaha.
	var debitInventory, creditAP float64
	if err := pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(debit) FILTER (WHERE account_id = $2), 0),
			COALESCE(SUM(credit) FILTER (WHERE account_id = $3), 0)
		FROM t_journal_entry_lines
		WHERE journal_entry_id = $1`, journalID, inventoryID, apID).
		Scan(&debitInventory, &creditAP); err != nil {
		t.Fatalf("gagal membaca baris jurnal: %v", err)
	}
	if debitInventory != 4000 {
		t.Fatalf("debit persediaan harus 4000, dapat %.2f", debitInventory)
	}
	if creditAP != 4000 {
		t.Fatalf("kredit hutang usaha harus 4000, dapat %.2f", creditAP)
	}

	// Menerbitkan ulang untuk GRN yang sama harus dilewati (idempoten).
	loadedGRN, err := grnRepo.GetByID(ctx, grn.ID)
	if err != nil {
		t.Fatalf("GetByID GRN gagal: %v", err)
	}
	loadedPO, err := poRepo.GetByID(ctx, s.poID)
	if err != nil {
		t.Fatalf("GetByID PO gagal: %v", err)
	}
	if err := journalUC.PostGRNJournal(ctx, loadedGRN, loadedPO, 0); err != nil {
		t.Fatalf("penerbitan ulang harus dilewati tanpa error, dapat %v", err)
	}

	var jumlahJurnal int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM t_journal_entries WHERE source_type = 'GRN' AND source_id = $1`, grn.ID).
		Scan(&jumlahJurnal); err != nil {
		t.Fatalf("gagal menghitung jurnal: %v", err)
	}
	if jumlahJurnal != 1 {
		t.Fatalf("harus tetap 1 jurnal untuk GRN ini, dapat %d", jumlahJurnal)
	}
}
