//go:build integration

package postgres

// Test integrasi untuk penerimaan barang (GRN).
//
// Fokusnya pada jaminan yang hanya bisa dibuktikan terhadap PostgreSQL sungguhan:
//
//  1. Stok bertambah dalam satuan jual memakai conversion_rate.
//  2. qty_received tidak bisa melebihi qty_ordered.
//  3. Penolakan tidak meninggalkan efek samping: tidak ada GRN yatim,
//     qty_received tidak bergerak, dan stok tidak berubah.
//
// Test ini butuh PostgreSQL nyata. Kalau DATABASE_URL tidak tersedia, test
// dilewati (skip), bukan gagal. Data uji memakai penanda "uji-grn-" dan
// dihapus di t.Cleanup.

import (
	"context"
	"errors"
	"math"
	"testing"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// penandaUjiGRN dipakai pada seluruh kolom teks milik data uji supaya tidak
// pernah menabrak data produksi dan mudah dihapus.
const penandaUjiGRN = "uji-grn-"

type scenarioGRN struct {
	poID       int64
	detailID   int64
	productID  int64
	qtyOrdered int32
	rate       int32
}

// siapkanSkenarioGRN membuat satu PO APPROVED dengan satu baris produk uji.
//
// conversion_rate disetel eksplisit supaya delta stok bisa dihitung tanpa
// bergantung pada data master yang sewaktu-waktu bisa diubah.
func siapkanSkenarioGRN(t *testing.T, ctx context.Context, pool *pgxpool.Pool, qtyOrdered, rate int32) scenarioGRN {
	t.Helper()

	// Pakai village yang sudah ada; tabel uji tidak perlu\multirow master baru.
	// id village bertipe varchar di master wilayah, bukan bigint.
	var villageID string
	if err := pool.QueryRow(ctx, `SELECT id FROM m_villages LIMIT 1`).Scan(&villageID); err != nil {
		t.Fatalf("gagal mengambil village acuan: %v", err)
	}

	var categoryID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_product_categories (name, slug)
		VALUES ($1, $2) RETURNING id`,
		"Uji GRN "+penandaUjiGRN, penandaUjiGRN+"kategori").Scan(&categoryID); err != nil {
		t.Fatalf("gagal membuat kategori uji: %v", err)
	}

	var supplierID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_suppliers (name, pic_name, phone, email, address, village_id)
		VALUES ($1, 'Uji GRN', '0000', 'uji@grn.local', 'Alamat Uji', $2)
		RETURNING id`, "Supplier "+penandaUjiGRN, villageID).Scan(&supplierID); err != nil {
		t.Fatalf("gagal membuat supplier uji: %v", err)
	}

	var buyUnit, sellUnit int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_units (kode, name) VALUES ($1, 'Satuan Beli Uji') RETURNING id`,
		penandaUjiGRN+"beli").Scan(&buyUnit); err != nil {
		t.Fatalf("gagal membuat satuan beli uji: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_units (kode, name) VALUES ($1, 'Satuan Jual Uji') RETURNING id`,
		penandaUjiGRN+"jual").Scan(&sellUnit); err != nil {
		t.Fatalf("gagal membuat satuan jual uji: %v", err)
	}

	var productID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO m_products (sku, name, category_id, supplier_id, selling_unit_id,
			purchase_unit_id, conversion_rate, purchase_price, selling_price,
			min_stock_level, current_stock)
		VALUES ($1, 'Produk Uji GRN', $2, $3, $5, $4, $6, 1000, 1500, 1, 100)
		RETURNING id`,
		penandaUjiGRN+"sku", categoryID, supplierID, buyUnit, sellUnit, rate).Scan(&productID); err != nil {
		t.Fatalf("gagal membuat produk uji: %v", err)
	}

	var poID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO t_purchase_orders (po_number, supplier_id, po_date, total_amount, status)
		VALUES ('PO/' || $1 || '/0001', $2, CURRENT_DATE, 0, 'APPROVED')
		RETURNING id`, penandaUjiGRN, supplierID).Scan(&poID); err != nil {
		t.Fatalf("gagal membuat PO uji: %v", err)
	}

	var detailID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO t_purchase_order_details (purchase_order_id, product_id, qty_ordered,
			qty_received, purchase_unit_id, unit_price, sub_total)
		VALUES ($1, $2, $3, 0, $4, 1000, 0)
		RETURNING id`, poID, productID, qtyOrdered, buyUnit).Scan(&detailID); err != nil {
		t.Fatalf("gagal membuat detail PO uji: %v", err)
	}

	t.Cleanup(func() {
		// Urutan penghapusan mengikuti rantai foreign key.
		_, _ = pool.Exec(ctx, `DELETE FROM t_goods_receipt_note_details WHERE purchase_order_detail_id IN
			(SELECT id FROM t_purchase_order_details WHERE purchase_order_id = $1)`, poID)
		_, _ = pool.Exec(ctx, `DELETE FROM t_goods_receipt_notes WHERE purchase_order_id = $1`, poID)
		_, _ = pool.Exec(ctx, `DELETE FROM t_purchase_order_details WHERE purchase_order_id = $1`, poID)
		_, _ = pool.Exec(ctx, `DELETE FROM t_purchase_orders WHERE id = $1`, poID)
		_, _ = pool.Exec(ctx, `DELETE FROM m_products WHERE id = $1`, productID)
		_, _ = pool.Exec(ctx, `DELETE FROM m_units WHERE id = ANY($1)`, []int64{buyUnit, sellUnit})
		_, _ = pool.Exec(ctx, `DELETE FROM m_suppliers WHERE id = $1`, supplierID)
		_, _ = pool.Exec(ctx, `DELETE FROM m_product_categories WHERE id = $1`, categoryID)
	})

	return scenarioGRN{
		poID:       poID,
		detailID:   detailID,
		productID:  productID,
		qtyOrdered: qtyOrdered,
		rate:       rate,
	}
}

func (s scenarioGRN) input(qty int32) domain.CreateGRNInput {
	return domain.CreateGRNInput{
		PurchaseOrderID: s.poID,
		Items:           []domain.CreateGRNItem{{PurchaseOrderDetailID: s.detailID, QtyReceived: qty}},
	}
}

func (s scenarioGRN) stokDanQty(ctx context.Context, t *testing.T, pool *pgxpool.Pool) (stok, qtyReceived int32) {
	t.Helper()
	if err := pool.QueryRow(ctx, `
		SELECT p.current_stock, d.qty_received
		FROM m_products p, t_purchase_order_details d
		WHERE p.id = $1 AND d.id = $2`, s.productID, s.detailID).Scan(&stok, &qtyReceived); err != nil {
		t.Fatalf("gagal membaca stok/qty_received: %v", err)
	}
	return stok, qtyReceived
}

// TestGRNStokBaruBertambahSaatPenerimaanDitutup membuktikan stok tidak berubah
// saat GRN dibuat, lalu bertambah dalam satuan jual ketika penerimaan ditutup:
// 4 karton dengan rate 12 harus menambah 48 pcs ke current_stock, sementara
// qty_received naik 4 dalam satuan beli sejak GRN dibuat.
func TestGRNStokBaruBertambahSaatPenerimaanDitutup(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := NewSQLGoodsReceiptNoteRepository(pool)

	s := siapkanSkenarioGRN(t, ctx, pool, 4, 12)
	stokAwal, _ := s.stokDanQty(ctx, t, pool)

	grn, err := repo.CreateGRN(ctx, s.input(4))
	if err != nil {
		t.Fatalf("CreateGRN gagal: %v", err)
	}
	if grn.ReceivedDate.IsZero() {
		t.Fatal("tanggal terima harus terisi")
	}

	// Stok belum boleh bergerak hanya karena GRN dibuat.
	stokSetelahGRN, qtyReceived := s.stokDanQty(ctx, t, pool)
	if stokSetelahGRN != stokAwal {
		t.Fatalf("stok tidak boleh berubah saat GRN dibuat, awal %d jadi %d", stokAwal, stokSetelahGRN)
	}
	if qtyReceived != 4 {
		t.Fatalf("qty_received harus 4 dalam satuan beli, dapat %d", qtyReceived)
	}

	// Snapshot rate harus tersimpan di detail GRN supaya perubahan master
	// produk tidak mengubah riwayat.
	var rateSnapshot int32
	if err := pool.QueryRow(ctx,
		`SELECT conversion_rate FROM t_goods_receipt_note_details WHERE grn_id = $1`, grn.ID).Scan(&rateSnapshot); err != nil {
		t.Fatalf("gagal membaca snapshot konversi: %v", err)
	}
	if rateSnapshot != 12 {
		t.Fatalf("snapshot konversi harus 12, dapat %d", rateSnapshot)
	}

	// Menutup penerimaan baru menambahkan stok.
	if err := repo.FinishReceiving(ctx, s.poID, 0, 1.0, 1); err != nil {
		t.Fatalf("FinishReceiving gagal: %v", err)
	}
	stokAkhir, _ := s.stokDanQty(ctx, t, pool)
	if want := stokAwal + 4*12; stokAkhir != want {
		t.Fatalf("stok harus %d (%d + 4*12) setelah ditutup, dapat %d", want, stokAwal, stokAkhir)
	}
}

// TestGRNFinishMenolakDaftarBerubah membuktikan celah antara pembacaan daftar
// GRN di usecase dan penutupan stok di repository sudah tertutup. Bila sebuah
// GRN baru muncul setelah jurnal GRN lama terbit, penutupan harus batal: GRN
// baru itu belum berjurnal, sehingga stoknya tidak boleh ikut dimasukkan.
func TestGRNFinishMenolakDaftarBerubah(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := NewSQLGoodsReceiptNoteRepository(pool)

	s := siapkanSkenarioGRN(t, ctx, pool, 4, 12)
	stokAwal, _ := s.stokDanQty(ctx, t, pool)

	if _, err := repo.CreateGRN(ctx, s.input(4)); err != nil {
		t.Fatalf("CreateGRN gagal: %v", err)
	}

	// usecase membaca 0 GRN, tetapi satu GRN sudah ada saat repository mengunci.
	err := repo.FinishReceiving(ctx, s.poID, 0, 1.0, 0)
	if !errors.Is(err, domain.ErrGRNReceiptChanged) {
		t.Fatalf("harus menolak daftar GRN yang berubah, dapat %v", err)
	}

	stokSetelahTolak, qtyReceived := s.stokDanQty(ctx, t, pool)
	if stokSetelahTolak != stokAwal {
		t.Fatalf("stok tidak boleh berubah saat penutupan ditolak, awal %d jadi %d", stokAwal, stokSetelahTolak)
	}
	if qtyReceived != 4 {
		t.Fatalf("qty_received harus tetap 4, dapat %d", qtyReceived)
	}
	if got := statusPO(ctx, pool, s.poID); got != "APPROVED" {
		t.Fatalf("status PO harus tetap APPROVED, dapat %s", got)
	}

	// Setelah daftar dibaca ulang, penutupan dengan jumlah yang benar berhasil.
	if err := repo.FinishReceiving(ctx, s.poID, 0, 1.0, 1); err != nil {
		t.Fatalf("penutupan ulang dengan jumlah GRN benar gagal: %v", err)
	}
	stokAkhir, _ := s.stokDanQty(ctx, t, pool)
	if want := stokAwal + 4*12; stokAkhir != want {
		t.Fatalf("stok harus %d setelah penutupan ulang, dapat %d", want, stokAkhir)
	}
}

// TestGRNPenerimaanBertahapBertumpuk memastikan beberapa batch untuk PO yang
// sama boleh terjadi, qty_received terakumulasi, dan tiap batch jadi GRN sendiri.
func TestGRNPenerimaanBertahapBertumpuk(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := NewSQLGoodsReceiptNoteRepository(pool)

	s := siapkanSkenarioGRN(t, ctx, pool, 10, 1)

	if _, err := repo.CreateGRN(ctx, s.input(4)); err != nil {
		t.Fatalf("batch 1 gagal: %v", err)
	}
	if _, err := repo.CreateGRN(ctx, s.input(3)); err != nil {
		t.Fatalf("batch 2 gagal: %v", err)
	}

	_, qtyReceived := s.stokDanQty(ctx, t, pool)
	if qtyReceived != 7 {
		t.Fatalf("qty_received harus 7, dapat %d", qtyReceived)
	}

	var jumlahGRN int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM t_goods_receipt_notes WHERE purchase_order_id = $1`, s.poID).Scan(&jumlahGRN); err != nil {
		t.Fatalf("gagal menghitung GRN: %v", err)
	}
	if jumlahGRN != 2 {
		t.Fatalf("harus ada 2 GRN untuk satu PO, dapat %d", jumlahGRN)
	}

	// Dua batch harus punya nomor berbeda.
	var nomor []string
	rows, err := pool.Query(ctx,
		`SELECT grn_number FROM t_goods_receipt_notes WHERE purchase_order_id = $1 ORDER BY id`, s.poID)
	if err != nil {
		t.Fatalf("gagal membaca nomor GRN: %v", err)
	}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("gagal scan nomor GRN: %v", err)
		}
		nomor = append(nomor, n)
	}
	rows.Close()
	if len(nomor) != 2 || nomor[0] == nomor[1] {
		t.Fatalf("dua batch harus punya nomor berbeda, dapat %v", nomor)
	}
}

// TestGRNOverReceiptDitolak memastikan qty melebihi sisa ditolak tanpa efek
// samping apa pun.
func TestGRNOverReceiptDitolak(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := NewSQLGoodsReceiptNoteRepository(pool)

	s := siapkanSkenarioGRN(t, ctx, pool, 5, 1)
	stokAwal, _ := s.stokDanQty(ctx, t, pool)

	if _, err := repo.CreateGRN(ctx, s.input(6)); !errors.Is(err, domain.ErrGRNOverReceipt) {
		t.Fatalf("harus menolak over-receipt dengan %v, dapat %v", domain.ErrGRNOverReceipt, err)
	}

	stokAkhir, qtyReceived := s.stokDanQty(ctx, t, pool)
	if qtyReceived != 0 {
		t.Fatalf("qty_received harus tetap 0 setelah penolakan, dapat %d", qtyReceived)
	}
	if stokAkhir != stokAwal {
		t.Fatalf("stok harus tetap %d setelah penolakan, dapat %d", stokAwal, stokAkhir)
	}

	var jumlahGRN int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM t_goods_receipt_notes WHERE purchase_order_id = $1`, s.poID).Scan(&jumlahGRN); err != nil {
		t.Fatalf("gagal menghitung GRN: %v", err)
	}
	if jumlahGRN != 0 {
		t.Fatalf("tidak boleh ada GRN tersimpan setelah penolakan, dapat %d", jumlahGRN)
	}
}

// TestGRNSisaDihitungDariQtyReceivedTerakhir memastikan validasi memakai sisa
// terbaru, bukan qty_ordered awal.
func TestGRNSisaDihitungDariQtyReceivedTerakhir(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := NewSQLGoodsReceiptNoteRepository(pool)

	s := siapkanSkenarioGRN(t, ctx, pool, 10, 1)

	if _, err := repo.CreateGRN(ctx, s.input(6)); err != nil {
		t.Fatalf("batch 1 gagal: %v", err)
	}
	// Sisa sekarang 4, jadi 5 harus ditolak.
	if _, err := repo.CreateGRN(ctx, s.input(5)); !errors.Is(err, domain.ErrGRNOverReceipt) {
		t.Fatalf("sisa 4 sehingga 5 harus ditolak, dapat %v", err)
	}
	// Tepat sisa harus diterima.
	if _, err := repo.CreateGRN(ctx, s.input(4)); err != nil {
		t.Fatalf("batch 2 (tepat sisa) gagal: %v", err)
	}

	_, qtyReceived := s.stokDanQty(ctx, t, pool)
	if qtyReceived != 10 {
		t.Fatalf("qty_received harus tepat 10, dapat %d", qtyReceived)
	}
}

// TestGRNPOSalahStatusDitolak memastikan PO non-APPROVED tidak bisa menerima.
func TestGRNPOSalahStatusDitolak(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := NewSQLGoodsReceiptNoteRepository(pool)

	s := siapkanSkenarioGRN(t, ctx, pool, 10, 1)
	if _, err := pool.Exec(ctx, `UPDATE t_purchase_orders SET status = 'PENDING' WHERE id = $1`, s.poID); err != nil {
		t.Fatalf("gagal mengubah status PO: %v", err)
	}

	if _, err := repo.CreateGRN(ctx, s.input(1)); !errors.Is(err, domain.ErrGRNPOSalahStatus) {
		t.Fatalf("harus menolak PO PENDING, dapat %v", err)
	}
}

// TestGRNQtyNolDitolak memastikan baris tanpa qty tidak pernah ikut tersimpan.
func TestGRNQtyNolDitolak(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := NewSQLGoodsReceiptNoteRepository(pool)

	s := siapkanSkenarioGRN(t, ctx, pool, 10, 1)

	if _, err := repo.CreateGRN(ctx, s.input(0)); !errors.Is(err, domain.ErrGRNBarisTidakValid) {
		t.Fatalf("qty 0 harus ditolak, dapat %v", err)
	}

	var jumlahGRN int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM t_goods_receipt_notes WHERE purchase_order_id = $1`, s.poID).Scan(&jumlahGRN); err != nil {
		t.Fatalf("gagal menghitung GRN: %v", err)
	}
	if jumlahGRN != 0 {
		t.Fatalf("tidak boleh ada GRN tersimpan, dapat %d", jumlahGRN)
	}
}

// TestGRNGabungDeltaProdukSama memastikan satu produk yang muncul di dua baris
// PO digabung jadi satu operasi stok, bukan saling menimpa.
func TestGRNGabungDeltaProdukSama(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := NewSQLGoodsReceiptNoteRepository(pool)

	s := siapkanSkenarioGRN(t, ctx, pool, 100, 2)

	var detailID2 int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO t_purchase_order_details (purchase_order_id, product_id, qty_ordered,
			qty_received, purchase_unit_id, unit_price, sub_total)
		SELECT purchase_order_id, product_id, 50, 0, purchase_unit_id, 1000, 0
		FROM t_purchase_order_details WHERE id = $1
		RETURNING id`, s.detailID).Scan(&detailID2); err != nil {
		t.Fatalf("gagal membuat baris kedua: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM t_purchase_order_details WHERE id = $1`, detailID2)
	})

	stokAwal, _ := s.stokDanQty(ctx, t, pool)

	_, err := repo.CreateGRN(ctx, domain.CreateGRNInput{
		PurchaseOrderID: s.poID,
		Items: []domain.CreateGRNItem{
			{PurchaseOrderDetailID: s.detailID, QtyReceived: 3},
			{PurchaseOrderDetailID: detailID2, QtyReceived: 2},
		},
	})
	if err != nil {
		t.Fatalf("CreateGRN gagal: %v", err)
	}

	// Produk yang sama dari dua baris belum menambah stok sampai ditutup.
	stokSetelahGRN, _ := s.stokDanQty(ctx, t, pool)
	if stokSetelahGRN != stokAwal {
		t.Fatalf("stok tidak boleh berubah saat GRN dibuat, awal %d jadi %d", stokAwal, stokSetelahGRN)
	}

	if err := repo.FinishReceiving(ctx, s.poID, 0, 1.0, 1); err != nil {
		t.Fatalf("FinishReceiving gagal: %v", err)
	}

	stokAkhir, _ := s.stokDanQty(ctx, t, pool)
	// (3 + 2) * rate 2 = 10
	if want := stokAwal + 10; stokAkhir != want {
		t.Fatalf("stok harus %d (%d + (3+2)*2) setelah ditutup, dapat %d", want, stokAwal, stokAkhir)
	}
}

// TestGRNPenjagaSQLMenolakOverReceipt memanggil statement increment langsung,
// melewati validasi aplikasi, untuk membuktikan penjaga di SQL benar-benar
// bekerja. Ini defense-in-depth terakhir: kalau lapisan ini longgar, stok dan
// qty_received bisa tidak sinkron meski validasi aplikasi terlihat rapi.
func TestGRNPenjagaSQLMenolakOverReceipt(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	q := db.New(pool)

	s := siapkanSkenarioGRN(t, ctx, pool, 5, 1)

	if _, err := q.IncrementDetailQtyReceived(ctx, db.IncrementDetailQtyReceivedParams{
		ID:       s.detailID,
		QtyToAdd: 9,
	}); err == nil {
		t.Fatal("penjaga SQL harus menolak qty melebihi qty_ordered")
	}

	_, qtyReceived := s.stokDanQty(ctx, t, pool)
	if qtyReceived != 0 {
		t.Fatalf("qty_received harus tetap 0, dapat %d", qtyReceived)
	}
}

// buatPOProdukSama membuat PO APPROVED tambahan dengan satu baris untuk produk
// yang sudah ada. Dipakai menguji penimbangan harga pokok dari dua penerimaan
// dengan harga per satuan yang berbeda.
func buatPOProdukSama(t *testing.T, ctx context.Context, pool *pgxpool.Pool, productID, purchaseUnitID int64, qty int32, unitPrice int64) scenarioGRN {
	t.Helper()

	var supplierID int64
	if err := pool.QueryRow(ctx, `SELECT supplier_id FROM m_products WHERE id = $1`, productID).Scan(&supplierID); err != nil {
		t.Fatalf("gagal membaca supplier produk: %v", err)
	}

	var poID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO t_purchase_orders (po_number, supplier_id, po_date, total_amount, status)
		VALUES ('PO/' || $1 || '/0002', $2, CURRENT_DATE, 0, 'APPROVED')
		RETURNING id`, penandaUjiGRN, supplierID).Scan(&poID); err != nil {
		t.Fatalf("gagal membuat PO kedua: %v", err)
	}

	var detailID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO t_purchase_order_details (purchase_order_id, product_id, qty_ordered,
			qty_received, purchase_unit_id, unit_price, sub_total)
		VALUES ($1, $2, $3, 0, $4, $5, 0)
		RETURNING id`, poID, productID, qty, purchaseUnitID, unitPrice).Scan(&detailID); err != nil {
		t.Fatalf("gagal membuat detail PO kedua: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM t_goods_receipt_note_details WHERE purchase_order_detail_id = $1`, detailID)
		_, _ = pool.Exec(ctx, `DELETE FROM t_goods_receipt_notes WHERE purchase_order_id = $1`, poID)
		_, _ = pool.Exec(ctx, `DELETE FROM t_purchase_order_details WHERE purchase_order_id = $1`, poID)
		_, _ = pool.Exec(ctx, `DELETE FROM t_purchase_orders WHERE id = $1`, poID)
	})

	return scenarioGRN{poID: poID, detailID: detailID, productID: productID, qtyOrdered: qty, rate: 1}
}

// bacaAvgCostStok membaca harga pokok rata-rata dan stok dalam satuan jual.
func bacaAvgCostStok(ctx context.Context, t *testing.T, pool *pgxpool.Pool, productID int64) (float64, int32) {
	t.Helper()
	var avgCost float64
	var stok int32
	if err := pool.QueryRow(ctx,
		`SELECT avg_cost::float8, current_stock FROM m_products WHERE id = $1`, productID).Scan(&avgCost, &stok); err != nil {
		t.Fatalf("gagal membaca avg_cost/stok: %v", err)
	}
	return avgCost, stok
}

// TestGRNHargaPokokRataRataMenimbangDuaPenerimaan membuktikan current_stock dan
// harga pokok rata-rata berubah bersama, dan rata-ratanya benar-benar menimbang
// nilai yang masuk. Stok awal dinolkan supaya angkanya murni berasal dari
// penerimaan yang diuji: 10 @1000 lalu 10 @1400 harus menghasilkan 20 satuan
// berharga pokok 1200.
func TestGRNHargaPokokRataRataMenimbangDuaPenerimaan(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := NewSQLGoodsReceiptNoteRepository(pool)

	s := siapkanSkenarioGRN(t, ctx, pool, 10, 1)
	if _, err := pool.Exec(ctx,
		`UPDATE m_products SET current_stock = 0, avg_cost = 0 WHERE id = $1`, s.productID); err != nil {
		t.Fatalf("gagal menolkan stok awal: %v", err)
	}

	var purchaseUnitID int64
	if err := pool.QueryRow(ctx,
		`SELECT purchase_unit_id FROM m_products WHERE id = $1`, s.productID).Scan(&purchaseUnitID); err != nil {
		t.Fatalf("gagal membaca satuan beli: %v", err)
	}
	po2 := buatPOProdukSama(t, ctx, pool, s.productID, purchaseUnitID, 10, 1400)

	// Penerimaan pertama: 10 satuan @1000, tanpa stok awal → harga pokok 1000.
	if _, err := repo.CreateGRN(ctx, s.input(10)); err != nil {
		t.Fatalf("GRN-1 gagal: %v", err)
	}
	if err := repo.FinishReceiving(ctx, s.poID, 0, 1.0, 1); err != nil {
		t.Fatalf("FinishReceiving-1 gagal: %v", err)
	}
	avg1, stok1 := bacaAvgCostStok(ctx, t, pool, s.productID)
	if stok1 != 10 {
		t.Fatalf("stok setelah penerimaan pertama harus 10, dapat %d", stok1)
	}
	if math.Abs(avg1-1000) > 0.0001 {
		t.Fatalf("harga pokok setelah penerimaan pertama harus 1000, dapat %v", avg1)
	}

	// Penerimaan kedua: 10 satuan @1400 → (10*1000 + 10*1400)/20 = 1200.
	if _, err := repo.CreateGRN(ctx, po2.input(10)); err != nil {
		t.Fatalf("GRN-2 gagal: %v", err)
	}
	if err := repo.FinishReceiving(ctx, po2.poID, 0, 1.0, 1); err != nil {
		t.Fatalf("FinishReceiving-2 gagal: %v", err)
	}
	avg2, stok2 := bacaAvgCostStok(ctx, t, pool, s.productID)
	if stok2 != 20 {
		t.Fatalf("stok setelah dua penerimaan harus 20, dapat %d", stok2)
	}
	if math.Abs(avg2-1200) > 0.0001 {
		t.Fatalf("harga pokok rata-rata harus 1200, dapat %v", avg2)
	}
}
