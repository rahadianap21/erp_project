//go:build integration

package postgres

// Test integrasi untuk compare-and-swap status Purchase Order.
//
// Yang diuji di sini adalah sifat yang hanya bisa dibuktikan terhadap PostgreSQL
// sungguhan: dua penulisan yang benar-benar bersamaan pada baris yang sama.
// Versi lama (SELECT status di aplikasi, lalu UPDATE tanpa syarat status)
// membiarkan semua penulisan berhasil; versi CAS hanya mengizinkan satu.
//
// Test ini butuh PostgreSQL nyata. Kalau DATABASE_URL tidak tersedia, test
// dilewati (skip), bukan gagal, supaya `go test ./...` tetap jalan di mesin
// tanpa database.

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// poUji membuat satu PO minimal berstatus PENDING dan mendaftarkan pembersihnya.
func poUji(t *testing.T, pool *pgxpool.Pool, suffix string) int64 {
	t.Helper()
	ctx := context.Background()

	var poID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO t_purchase_orders (po_number, supplier_id, status)
		VALUES ($1, 1, 'PENDING')
		RETURNING id
	`, "PO/UJI-CAS/"+suffix).Scan(&poID)
	if err != nil {
		t.Fatalf("gagal membuat PO uji: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM t_purchase_orders WHERE id = $1`, poID)
	})
	return poID
}

func statusPO(ctx context.Context, pool *pgxpool.Pool, id int64) string {
	var s string
	if err := pool.QueryRow(ctx,
		`SELECT status FROM t_purchase_orders WHERE id = $1`, id).Scan(&s); err != nil {
		return "(hilang)"
	}
	return s
}

// TestTransitionStatus_HanyaSatukaliYangCocok membuktikan sifat CAS-nya:
// percobaan kedua dengan syarat status yang sama harus menyentuh nol baris,
// bukan menimpa status yang sudah ditulis percobaan pertama.
func TestTransitionStatus_HanyaSatukaliYangCocok(t *testing.T) {
	pool := newTestPool(t)
	repo := NewSQLPurchaseOrderRepository(pool)
	ctx := context.Background()
	poID := poUji(t, pool, "SATU")

	rows, err := repo.TransitionStatus(ctx, poID, []string{"PENDING"}, nil, "APPROVED", 1, "")
	if err != nil {
		t.Fatalf("transition pertama gagal: %v", err)
	}
	if rows != 1 {
		t.Fatalf("transition pertama harus mengenai 1 baris, dapat %d", rows)
	}
	if got := statusPO(ctx, pool, poID); got != "APPROVED" {
		t.Fatalf("status harus APPROVED, dapat %s", got)
	}

	rows, err = repo.TransitionStatus(ctx, poID, []string{"PENDING"}, nil, "APPROVED", 1, "")
	if err != nil {
		t.Fatalf("transition kedua tidak boleh error: %v", err)
	}
	if rows != 0 {
		t.Fatalf("transition kedua harus 0 baris karena syarat status tidak lagi cocok, dapat %d", rows)
	}
}

// TestTransitionStatus_ForbidListMenutupPembatalan menggambarkan race approve/cancel:
// begitu approve menulis APPROVED, syarat pembatalan "bukan APPROVED dan bukan
// RECEIVED" tidak lagi cocok sehingga dokumen tidak bisa ditimpa.
func TestTransitionStatus_ForbidListMenutupPembatalan(t *testing.T) {
	pool := newTestPool(t)
	repo := NewSQLPurchaseOrderRepository(pool)
	ctx := context.Background()
	poID := poUji(t, pool, "FORBID")

	if rows, err := repo.TransitionStatus(ctx, poID, []string{"PENDING"}, nil, "APPROVED", 1, ""); err != nil || rows != 1 {
		t.Fatalf("approve harus berhasil, dapat rows=%d err=%v", rows, err)
	}

	rows, err := repo.TransitionStatus(ctx, poID, nil, []string{"APPROVED", "RECEIVED"}, "REJECTED", 1, "alasan")
	if err != nil {
		t.Fatalf("transition forbid tidak boleh error: %v", err)
	}
	if rows != 0 {
		t.Fatalf("pembatalan setelah approve harus 0 baris, dapat %d", rows)
	}
	if got := statusPO(ctx, pool, poID); got != "APPROVED" {
		t.Fatalf("status harus tetap APPROVED, dapat %s", got)
	}
}

// TestTransitionStatus_DuaPuluhPermintaanParalel adalah inti pembuktiannya.
// 20 goroutine menjalankan approve pada PO yang sama persis bersamaan, dan
// yang diuji: tepat satu yang boleh mengubah status.
//
// Pada implementasi lama (UPDATE tanpa syarat status) seluruh 20 permintaan akan
// melaporkan berhasil. Dengan CAS, PostgreSQL mengunci baris sehingga hanya
// transition pertama yang syaratnya masih cocok.
func TestTransitionStatus_DuaPuluhPermintaanParalel(t *testing.T) {
	const worker = 20

	pool := newTestPool(t)
	repo := NewSQLPurchaseOrderRepository(pool)
	poID := poUji(t, pool, "PARALEL")

	var (
		mu       sync.Mutex
		success  int
		rowsAll  []int64
		failErrs []error
	)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < worker; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start // dilepas semua sekaligus supaya balapan benar-benar terjadi

			rows, err := repo.TransitionStatus(context.Background(), poID,
				[]string{"PENDING"}, nil, "APPROVED", int64(n+1), "")

			mu.Lock()
			defer mu.Unlock()
			rowsAll = append(rowsAll, rows)
			if err != nil {
				failErrs = append(failErrs, err)
				return
			}
			if rows == 1 {
				success++
			}
		}(i)
	}

	close(start)
	wg.Wait()

	for _, err := range failErrs {
		t.Errorf("transition paralel tidak boleh error: %v", err)
	}
	if success != 1 {
		t.Fatalf("tepat 1 permintaan yang harusnya berhasil, dapat %d (rows=%v)", success, rowsAll)
	}
	if got := statusPO(context.Background(), pool, poID); got != "APPROVED" {
		t.Fatalf("status akhir harus APPROVED, dapat %s", got)
	}
}

// TestTransitionStatus_POTerhapusTidakBisaDitransisi memastikan penjaga
// deleted_at tetap berlaku pada jalur CAS.
func TestTransitionStatus_POTerhapusTidakBisaDitransisi(t *testing.T) {
	pool := newTestPool(t)
	repo := NewSQLPurchaseOrderRepository(pool)
	ctx := context.Background()
	poID := poUji(t, pool, "HAPUS")

	if _, err := pool.Exec(ctx,
		`UPDATE t_purchase_orders SET deleted_at = NOW() WHERE id = $1`, poID); err != nil {
		t.Fatalf("gagal menandai PO terhapus: %v", err)
	}

	rows, err := repo.TransitionStatus(ctx, poID, []string{"PENDING"}, nil, "APPROVED", 1, "")
	if err != nil {
		t.Fatalf("transition tidak boleh error: %v", err)
	}
	if rows != 0 {
		t.Fatalf("PO terhapus tidak boleh bertransisi, dapat %d baris", rows)
	}
}

// TestConstraintStatus_MenolakNilaiAsing memverifikasi CHECK constraint dari
// migrasi 000003. Tanpa ini, daftar status yang diandalkan compare-and-swap
// bisa aus tanpa ada yang menyadari.
func TestConstraintStatus_MenolakNilaiAsing(t *testing.T) {
	// Constraint tidak boleh di-skip kalau tidak ada di database. Dulu test ini
	// dilewati karena migrasi 000003 tercatat sudah dijalankan di
	// schema_migrations padahal constraint-nya hilang. Skip seperti itu membuat
	// regresi terlihat hijau.
	pool := newTestPool(t)
	ctx := context.Background()
	poID := poUji(t, pool, "KONSTRAINT")

	for _, tidakSah := range []string{"SELESAI", "approved", "APPROVED ", ""} {
		t.Run("status="+tidakSah, func(t *testing.T) {
			_, err := pool.Exec(ctx,
				`UPDATE t_purchase_orders SET status = $2 WHERE id = $1`, poID, tidakSah)
			if err == nil {
				_, reset := pool.Exec(ctx,
					`UPDATE t_purchase_orders SET status = 'PENDING' WHERE id = $1`, poID)
				if reset != nil {
					t.Fatalf("gagal mengembalikan status: %v", reset)
				}
				t.Fatalf("status %q seharusnya ditolak CHECK constraint", tidakSah)
			}
		})
	}
}
