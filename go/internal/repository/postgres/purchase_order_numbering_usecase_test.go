//go:build integration

package postgres

// Test full-stack untuk penerbitan nomor dokumen Purchase Order.
//
// Test di purchase_order_usecase_test.go memakai fake, jadi hanya menguji
// orkestrasi use case. Test di sini memakai repository PostgreSQL sungguhan
// supaya dua jaminan yang menyangkut database ikut terbukti:
//
//  1. Nomor tidak pernah dipakai ulang, termasuk setelah dokumen di-soft delete.
//     Implementasi lama memakai COUNT+1 terhadap baris yang masih aktif, sehingga
//     percobaan kedua meminta nomor yang sama dan gagal dengan pelanggaran unique
//     constraint "t_purchase_orders_po_number_key" yang bocor ke user sebagai 500.
//  2. Nomor tetap unik saat banyak permintaan datang bersamaan.
//
// CATATAN KEAMANAN DATA: karena CreatePO selalu memakai tanggal hari ini, test ini
// memangHEVmemakai prefiks counter yang sama dengan dokumen sungguhan. Demi itu
// test TIDAK BOLEH menghapus berdasarkan prefiks: setiap PO uji dikumpulkan nomornya
// lalu dihapus tepat pada nomor tersebut, jadi dokumen asli tidak pernah tersentuh.
// Counter hari ini sengaja dibiarkan maju — itu hanya menyisip celah angka, bukan
// memakai ulang nomor; mundur demi kerapian justru berisiko membuat nomor bentrok
// dengan dokumen yang sedang jalan.
//
// Test butuh PostgreSQL nyata. Kalau DATABASE_URL tidak tersedia, test dilewati.

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/usecase"

	"github.com/jackc/pgx/v5/pgxpool"
)

// supplierUji memakai supplier yang sudah ada, bukan membuat baru.
//
// Test ini tidak butuh supplier khusus: dokumen uji dibersihkan berdasarkan nomor
// PO yang benar-benar terbit, bukan berdasarkan supplier atau prefiks
// tanggal, sehingga supplier asli tidak pernah tersentuh.
func supplierUjiPO(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(),
		`SELECT id FROM m_suppliers WHERE deleted_at IS NULL ORDER BY id LIMIT 1`).Scan(&id); err != nil {
		t.Skipf("tidak ada supplier, test dilewati: %v", err)
	}
	return id
}

// produkSiapBeliPO memilih produk milik supplier dokumen. CreatePO sekarang
// menolak barang vendor lain (sama seperti UpdatePO), jadi produk dan supplier
// uji tidak boleh diambil dari dua query yang saling lepas.
func produkSiapBeliPO(t *testing.T, pool *pgxpool.Pool, supplierID int64) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(), `
		SELECT id FROM m_products
		WHERE deleted_at IS NULL AND purchase_price > 0 AND purchase_unit_id IS NOT NULL
		  AND supplier_id = $1
		ORDER BY id LIMIT 1`, supplierID).Scan(&id); err != nil {
		t.Skipf("tidak ada produk siap compra untuk supplier %d, test dilewati: %v", supplierID, err)
	}
	return id
}

func poolUjiPO(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL kosong, test integrasi dilewati")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("gagal membuat pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("database tidak merespon, test integrasi dilewati: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// hapusPOUji menghapus hanya PO dengan nomor yang dikumpulkan test.
func hapusPOUjiNomer(t *testing.T, pool *pgxpool.Pool, nomor []string) {
	t.Helper()
	if len(nomor) == 0 {
		return
	}
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `
		DELETE FROM t_purchase_order_details
		WHERE purchase_order_id IN (SELECT id FROM t_purchase_orders WHERE po_number = ANY($1))`, nomor)
	_, _ = pool.Exec(ctx, `DELETE FROM t_purchase_orders WHERE po_number = ANY($1)`, nomor)
}

func poPayloadUji(supplierID, productID int64) *domain.PurchaseOrder {
	return &domain.PurchaseOrder{
		SupplierID: supplierID,
		TaxType:    "NON_PPN",
		Items:      []domain.PurchaseOrderDetail{{ProductID: productID, QtyOrdered: 1}},
	}
}

// TestCreatePO_NomorTidakDipakaiUlangSetelahSoftDelete menguji alur yang dulu
// selalu gagal: buat PO, hapus dari UI (soft delete), lalu buat lagi.
//
// Versi lama menghitung nomor dari baris yang belum dihapus, jadi percobaan kedua
// meminta nomor pertama dan gagal. Versi baru harus menerbitkan nomor baru.
func TestCreatePO_NomorTidakDipakaiUlangSetelahSoftDelete(t *testing.T) {
	pool := poolUjiPO(t)
	ctx := context.Background()
	supplierID := supplierUjiPO(t, pool)
	productID := produkSiapBeliPO(t, pool, supplierID)
	u := usecase.NewPurchaseOrderUsecase(
		NewSQLPurchaseOrderRepository(pool),
		NewSQLProductRepository(pool),
	)

	// PO pertama.
	pertama := poPayloadUji(supplierID, productID)
	if err := u.CreatePO(ctx, pertama); err != nil {
		t.Fatalf("create pertama gagal: %v", err)
	}
	t.Cleanup(func() { hapusPOUjiNomer(t, pool, []string{pertama.PoNumber}) })

	if pertama.PoNumber == "" {
		t.Fatal("nomor PO pertama harus terbit")
	}

	// Pembatalan dari UI: soft delete, baris tetap ada di tabel.
	if _, err := pool.Exec(ctx,
		`UPDATE t_purchase_orders SET deleted_at = NOW() WHERE po_number = $1`, pertama.PoNumber); err != nil {
		t.Fatalf("gagal soft delete: %v", err)
	}

	// PO kedua, setelah dokumen pertama dihapus.
	kedua := poPayloadUji(supplierID, productID)
	if err := u.CreatePO(ctx, kedua); err != nil {
		t.Fatalf("create setelah soft delete harus berhasil, dapat error: %v", err)
	}
	t.Cleanup(func() { hapusPOUjiNomer(t, pool, []string{kedua.PoNumber}) })

	if kedua.PoNumber == pertama.PoNumber {
		t.Fatalf("nomor dipakai ulang setelah soft delete: %s", kedua.PoNumber)
	}
	t.Logf("pertama %s (soft deleted), kedua %s", pertama.PoNumber, kedua.PoNumber)

	// Penghitung harus terus maju, tidak pernah mundur.
	var lastSeq int64
	prefiks := time.Now().Format("20060102")
	if err := pool.QueryRow(ctx,
		`SELECT last_seq FROM t_po_number_counters WHERE prefix = $1`, prefiks).Scan(&lastSeq); err != nil {
		t.Fatalf("counter hari ini harus ada: %v", err)
	}
	if lastSeq < 2 {
		t.Errorf("counter harus minimal 2 setelah dua PO, dapat %d", lastSeq)
	}
}

// TestCreatePO_ParalelSemuaBerhasilAdalahIntiRace menguji banyak create
// bersamaan. Implementasi lama hanya berhasil satu permintaan; sisanya gagal
// dengan SQLSTATE 23505 yang sampai ke user sebagai HTTP 500.
func TestCreatePO_ParalelSemuaBerhasilAdalahIntiRace(t *testing.T) {
	const worker = 8

	pool := poolUjiPO(t)
	ctx := context.Background()
	supplierID := supplierUjiPO(t, pool)
	productID := produkSiapBeliPO(t, pool, supplierID)
	u := usecase.NewPurchaseOrderUsecase(
		NewSQLPurchaseOrderRepository(pool),
		NewSQLProductRepository(pool),
	)

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		nomor = map[string]bool{}
		errs  []error
		start = make(chan struct{})
	)

	for i := 0; i < worker; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			po := poPayloadUji(supplierID, productID)
			err := u.CreatePO(ctx, po)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			nomor[po.PoNumber] = true
		}()
	}
	close(start)
	wg.Wait()

	// Bersihkan berdasarkan nomor yang benar-benar terbit, bukan berdasarkan
	// prefiks tanggal, supaya dokumen asli tidak ikut terhapus.
	var terbit []string
	for n := range nomor {
		terbit = append(terbit, n)
	}
	t.Cleanup(func() { hapusPOUjiNomer(t, pool, terbit) })

	for _, e := range errs {
		if strings.Contains(e.Error(), "duplicate key") || strings.Contains(e.Error(), "23505") {
			t.Fatalf("create paralel gagal karena nomor bentrok — race belum diperbaiki: %v", e)
		}
	}
	if len(errs) > 0 {
		t.Fatalf("%d dari %d create paralel gagal: %v", len(errs), worker, errs[0])
	}
	if len(nomor) != worker {
		t.Fatalf("harus ada %d nomor unik dari %d create, hanya didapat %d: %v",
			worker, worker, len(nomor), terbit)
	}
}
