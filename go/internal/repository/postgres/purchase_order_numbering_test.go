//go:build integration

package postgres

// Test integrasi untuk penerbitan nomor dokumen Purchase Order.
//
// Fokusnya pada dua jaminan yang hanya bisa dibuktikan terhadap PostgreSQL
// sungguhan:
//
//  1. Balanced — nomor yang terbit tidak akan pernah dipakai lagi, termasuk
//     setelah dokumennya dihapus.
//  2. Balapan — banyak pemanggil paralel tetap masing-masing mendapat nomor
//     yang berbeda.
//
// Test ini butuh PostgreSQL nyata. Kalau DATABASE_URL tidak tersedia, test
// dilewati (skip), bukan gagal.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"go_backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// prefiksUji memakai prefiks yang tidak mungkin dipakai dokumen sungguhan, jadi
// test tidak pernah menyentuh nomor PO yang sudah benar-benar terbit.
func prefiksUji(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("9%s", "99999999"[:7])
}

func bersihkanPrefiksUji(ctx context.Context, pool *pgxpool.Pool, prefiks string) {
	_, _ = pool.Exec(ctx, `DELETE FROM t_purchase_order_details WHERE purchase_order_id IN
		(SELECT id FROM t_purchase_orders WHERE po_number LIKE 'PO/'||$1||'/%')`, prefiks)
	_, _ = pool.Exec(ctx, `DELETE FROM t_purchase_orders WHERE po_number LIKE 'PO/'||$1||'/%'`, prefiks)
	_, _ = pool.Exec(ctx, `DELETE FROM t_po_number_counters WHERE prefix = $1`, prefiks)
}

// TestNextPurchaseOrderSeq_BerurutanDanUnik memeriksa nomor keluar berurutan dan
// tidak pernah kembar, termasuk saat dipanggil berulang pada prefiks yang sama.
func TestNextPurchaseOrderSeq_BerurutanDanUnik(t *testing.T) {
	pool := newTestPool(t)
	repo := NewSQLPurchaseOrderRepository(pool)
	ctx := context.Background()
	prefiks := prefiksUji(t)

	t.Cleanup(func() { bersihkanPrefiksUji(ctx, pool, prefiks) })

	seen := map[int64]bool{}
	for i := 1; i <= 25; i++ {
		n, err := repo.NextPurchaseOrderSeq(ctx, prefiks)
		if err != nil {
			t.Fatalf("percobaan %d gagal: %v", i, err)
		}
		if n != int64(i) {
			t.Fatalf("nomor ke-%d harus %d, dapat %d", i, i, n)
		}
		if seen[n] {
			t.Fatalf("nomor %d terbit dua kali", n)
		}
		seen[n] = true
	}
}

// TestNextPurchaseOrderSeq_ParalelSelaluUnik adalah inti dari perbaikan race.
// 24 goroutine meminta nomor sekaligus; semua harus mendapat angka berbeda.
func TestNextPurchaseOrderSeq_ParalelSelaluUnik(t *testing.T) {
	const worker = 24

	pool := newTestPool(t)
	repo := NewSQLPurchaseOrderRepository(pool)
	ctx := context.Background()
	prefiks := prefiksUji(t)

	t.Cleanup(func() { bersihkanPrefiksUji(ctx, pool, prefiks) })

	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		hasil = map[int64]bool{}
		dupl  []int64
		start = make(chan struct{})
	)

	for i := 0; i < worker; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			n, err := repo.NextPurchaseOrderSeq(ctx, prefiks)
			if err != nil {
				t.Errorf("penghitung gagal: %v", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if hasil[n] {
				dupl = append(dupl, n)
			}
			hasil[n] = true
		}()
	}
	close(start)
	wg.Wait()

	if len(dupl) > 0 {
		t.Fatalf("nomor kembar saat paralel: %v", dupl)
	}
	if len(hasil) != worker {
		t.Fatalf("harus ada %d nomor unik, dapat %d", worker, len(hasil))
	}

	var lastSeq int64
	if err := pool.QueryRow(ctx,
		`SELECT last_seq FROM t_po_number_counters WHERE prefix = $1`, prefiks).Scan(&lastSeq); err != nil {
		t.Fatal(err)
	}
	if lastSeq != worker {
		t.Fatalf("penghitung harus berhenti di %d, dapat %d", worker, lastSeq)
	}
}

// TestNextPurchaseOrderSeq_TidakMundurSetelahPenghapusan membuktikan numbering
// tidak bergantung pada baris PO. Setelah semua PO dihapus (termasuk soft
// delete), nomor berikutnya tetap berlanjut dan tidak pernah mengulang.
func TestNextPurchaseOrderSeq_TidakMundurSetelahPenghapusan(t *testing.T) {
	pool := newTestPool(t)
	repo := NewSQLPurchaseOrderRepository(pool)
	ctx := context.Background()
	prefiks := prefiksUji(t)

	t.Cleanup(func() { bersihkanPrefiksUji(ctx, pool, prefiks) })

	// Terbitkan tiga nomor dan buat dokumennya.
	var ids []int64
	for i := 1; i <= 3; i++ {
		n, err := repo.NextPurchaseOrderSeq(ctx, prefiks)
		if err != nil {
			t.Fatal(err)
		}
		var id int64
		err = pool.QueryRow(ctx, `
			INSERT INTO t_purchase_orders (po_number, supplier_id, po_date, status)
			VALUES ($1, 1, NOW(), 'PENDING') RETURNING id`,
			fmt.Sprintf("PO/%s/%04d", prefiks, n)).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}

	// Soft delete semua, persis seperti pembatalan dari UI.
	if _, err := pool.Exec(ctx,
		`UPDATE t_purchase_orders SET deleted_at = NOW() WHERE id = ANY($1)`, ids); err != nil {
		t.Fatal(err)
	}
	// Dan hapus satu secara permanen.
	if _, err := pool.Exec(ctx,
		`DELETE FROM t_purchase_orders WHERE id = $1`, ids[0]); err != nil {
		t.Fatal(err)
	}

	// Nomor berikutnya harus 4, bukan 1.
	n, err := repo.NextPurchaseOrderSeq(ctx, prefiks)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatalf("setelah penghapusan nomor berikutnya harus 4, dapat %d", n)
	}
}

// TestCreate_NomorBentrokDipetakanJadiErrNomorPOBentrok memastikan pelanggaran
// unique constraint diterjemahkan menjadi error domain, bukan pesan PostgreSQL
// mentah.
//
// Ini yang membuat pesan "duplicate key value violates unique constraint
// \"t_purchase_orders_po_number_key\"" tidak pernah sampai ke pengguna sebagai
// HTTP 500. Nomornya dibuat unik untuk test ini supaya tidak pernah bentrok
// dengan dokumen lain.
func TestCreate_NomorBentrokDipetakanJadiErrNomorPOBentrok(t *testing.T) {
	pool := newTestPool(t)
	repo := NewSQLPurchaseOrderRepository(pool)
	ctx := context.Background()

	nomor := fmt.Sprintf("PO/77777777/%04d", time.Now().UnixNano()%10000)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM t_purchase_orders WHERE po_number = $1`, nomor)
	})

	// Satu dokumen lebih dulu sebagai jebakan.
	if _, err := pool.Exec(ctx, `
		INSERT INTO t_purchase_orders (po_number, supplier_id, po_date, status)
		VALUES ($1, 1, NOW(), 'PENDING')`, nomor); err != nil {
		t.Fatal(err)
	}

	//.Domain Create dengan nomor yang sama harus gagal dengan error yang bisa
	// dikenali pemanggil.
	po := &domain.PurchaseOrder{
		PoNumber:    nomor,
		SupplierID:  1,
		PoDate:      time.Now(),
		Status:      "PENDING",
		TotalAmount: 0,
	}
	err := repo.Create(ctx, po)
	if err == nil {
		t.Fatal("nomor kembar harus ditolak")
	}
	if !errors.Is(err, domain.ErrNomorPOBentrok) {
		t.Fatalf("error harus ErrNomorPOBentrok, dapat: %v", err)
	}
	// Nama constraint boleh ikut karena membantu pelaporan ke log internal.
	if !strings.Contains(err.Error(), "t_purchase_orders_po_number_key") {
		t.Errorf("pesan internal sebaiknya menyebut constraint, dapat: %s", err.Error())
	}
}

// TestCreate_NomorKembar karya_SoftDeletedTetapDitolak membuktikan soft delete
// tidak melonggarkan unique constraint: nomor yang "dipakai ulang" setelah
// penghapusan tetap bentrok di database, dan aplikasi harus mengambil
// nomor lain, bukan memaksa.
func TestCreate_NomorKembarSoftDeletedTetapDitolak(t *testing.T) {
	pool := newTestPool(t)
	repo := NewSQLPurchaseOrderRepository(pool)
	ctx := context.Background()

	nomor := fmt.Sprintf("PO/77777777/%04d", (time.Now().UnixNano()+7)%10000)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM t_purchase_orders WHERE po_number = $1`, nomor)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO t_purchase_orders (po_number, supplier_id, po_date, status, deleted_at)
		VALUES ($1, 1, NOW(), 'PENDING', NOW())`, nomor); err != nil {
		t.Fatal(err)
	}

	err := repo.Create(ctx, &domain.PurchaseOrder{
		PoNumber: nomor, SupplierID: 1, PoDate: time.Now(), Status: "PENDING",
	})
	if !errors.Is(err, domain.ErrNomorPOBentrok) {
		t.Fatalf("nomor soft-deleted tetap terpakai, harus ErrNomorPOBentrok, dapat: %v", err)
	}
}
