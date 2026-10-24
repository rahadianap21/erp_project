//go:build integration

package postgres

// Test integrasi penyimpanan Idempotency-Key.
//
// Fokusnya pada sifat yang hanya bisa dibuktikan terhadap PostgreSQL nyata:
// klaim unik per (pengguna, endpoint, key), pemutaran ulang respons, penolakan
// payload berbeda, dan pengambilalihan klaim basi.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go_backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

func idempotencyTestUser(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	var userID int64
	if err := pool.QueryRow(context.Background(),
		`SELECT id FROM s_users ORDER BY id LIMIT 1`).Scan(&userID); err != nil {
		t.Skipf("tidak ada pengguna, test dilewati: %v", err)
	}
	return userID
}

func TestIdempotencyClaimCompleteDanReplay(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	userID := idempotencyTestUser(t, pool)
	key := fmt.Sprintf("UJI-IDEMPOTENSI-%d", time.Now().UnixNano())
	scope := "POST /api/sales"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM t_idempotency_keys WHERE user_id = $1 AND scope = $2 AND idempotency_key = $3`,
			userID, scope, key)
	})

	repo := NewSQLIdempotencyRepository(pool)
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	claimed, err := repo.Claim(ctx, userID, scope, key, hash)
	if err != nil {
		t.Fatalf("claim pertama gagal: %v", err)
	}
	if claimed.Completed {
		t.Fatal("klaim pertama tidak boleh dianggap selesai")
	}

	body := []byte(`{"data":{"id":99}}`)
	if err := repo.Complete(ctx, claimed.ID, 201, body, "application/json"); err != nil {
		t.Fatalf("Complete gagal: %v", err)
	}

	replay, err := repo.Claim(ctx, userID, scope, key, hash)
	if err != nil {
		t.Fatalf("claim ulang gagal: %v", err)
	}
	if !replay.Completed || replay.ResponseStatus != 201 {
		t.Fatalf("retry harus memutar ulang 201, dapat completed=%v status=%d", replay.Completed, replay.ResponseStatus)
	}
	if string(replay.ResponseBody) != string(body) {
		t.Fatalf("body replay berubah: %q", replay.ResponseBody)
	}
	if replay.ResponseContentType != "application/json" {
		t.Fatalf("content-type replay berubah: %q", replay.ResponseContentType)
	}

	otherHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := repo.Claim(ctx, userID, scope, key, otherHash); !errors.Is(err, domain.ErrIdempotencyKeyReused) {
		t.Fatalf("hash berbeda harus ErrIdempotencyKeyReused, dapat %v", err)
	}
}

func TestIdempotencyKlaimBerjalanTidakBisaDiambilAlih(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	userID := idempotencyTestUser(t, pool)
	key := fmt.Sprintf("UJI-IDEMPOTENSI-BASI-%d", time.Now().UnixNano())
	scope := "POST /api/goods-receipt-notes"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM t_idempotency_keys WHERE user_id = $1 AND scope = $2 AND idempotency_key = $3`,
			userID, scope, key)
	})

	repo := NewSQLIdempotencyRepository(pool)
	hash := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

	claimed, err := repo.Claim(ctx, userID, scope, key, hash)
	if err != nil {
		t.Fatalf("claim pertama gagal: %v", err)
	}
	if _, err := repo.Claim(ctx, userID, scope, key, hash); !errors.Is(err, domain.ErrIdempotencyInProgress) {
		t.Fatalf("klaim kedua harus ErrIdempotencyInProgress, dapat %v", err)
	}

	// Klaim tua tetap tidak boleh diambil alih: hasil transaksi bisnisnya belum
	// tentu tidak ada, sehingga eksekusi ulang bisa menggandakan dokumen.
	if _, err := pool.Exec(ctx, `
		UPDATE t_idempotency_keys SET updated_at = NOW() - INTERVAL '3 minutes'
		WHERE id = $1`, claimed.ID); err != nil {
		t.Fatalf("gagal membuat klaim basi: %v", err)
	}
	if _, err := repo.Claim(ctx, userID, scope, key, hash); !errors.Is(err, domain.ErrIdempotencyInProgress) {
		t.Fatalf("klaim basi harus tetap ErrIdempotencyInProgress, dapat %v", err)
	}
	if err := repo.Release(ctx, claimed.ID); err != nil {
		t.Fatalf("Release gagal: %v", err)
	}
}

func TestIdempotencyKlaimBersamaanHanyaSatuYangMenang(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	userID := idempotencyTestUser(t, pool)
	key := fmt.Sprintf("UJI-IDEMPOTENSI-BERSAMAAN-%d", time.Now().UnixNano())
	scope := "POST /api/sales"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM t_idempotency_keys WHERE user_id = $1 AND scope = $2 AND idempotency_key = $3`,
			userID, scope, key)
	})

	repo := NewSQLIdempotencyRepository(pool)
	hash := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"

	start := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			_, err := repo.Claim(ctx, userID, scope, key, hash)
			results <- err
		}()
	}
	close(start)

	var won, blocked int
	for i := 0; i < 2; i++ {
		err := <-results
		switch {
		case err == nil:
			won++
		case errors.Is(err, domain.ErrIdempotencyInProgress):
			blocked++
		default:
			t.Fatalf("hasil klaim tak terduga: %v", err)
		}
	}
	if won != 1 || blocked != 1 {
		t.Fatalf("harus tepat satu menang dan satu tertahan, dapat menang=%d tertahan=%d", won, blocked)
	}
}
