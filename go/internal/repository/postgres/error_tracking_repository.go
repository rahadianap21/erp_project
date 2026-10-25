package postgres

import (
	"context"
	"errors"
	"time"

	"go_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// sqlErrorStore menyimpan error ke PostgreSQL tanpa lewat sqlc karena tabel
// ini tidak dipakai di query bisnis lain. Semua teks sudah dibersihkan oleh
// usecase sebelum sampai ke sini.
type sqlErrorStore struct {
	pool *pgxpool.Pool
}

func NewSQLErrorStore(pool *pgxpool.Pool) domain.ErrorStore {
	return &sqlErrorStore{pool: pool}
}

func (r *sqlErrorStore) Record(ctx context.Context, event domain.ErrorEvent) (domain.ErrorRecordResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.ErrorRecordResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var result domain.ErrorRecordResult

	// Coba buat grup baru. ON CONFLICT DO NOTHING + RETURNING adalah cara
	// atomik untuk membedakan error baru dari error yang sudah pernah muncul,
	// tanpa balapan antar request yang datang bersamaan.
	err = tx.QueryRow(ctx, `
		INSERT INTO t_error_groups (fingerprint, source, message, route)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (fingerprint) DO NOTHING
		RETURNING first_seen, last_seen`,
		event.Fingerprint, event.Source, event.Message, event.Route,
	).Scan(&result.FirstSeen, &result.LastSeen)

	switch {
	case err == nil:
		result.IsNew = true
		result.Occurrences = 1
	case errors.Is(err, pgx.ErrNoRows):
		// Grup sudah ada: naikkan hitungan dan simpan pesan terbaru.
		err = tx.QueryRow(ctx, `
			UPDATE t_error_groups
			SET occurrences = occurrences + 1,
			    last_seen = NOW(),
			    message = $2,
			    route = $3
			WHERE fingerprint = $1
			RETURNING occurrences, first_seen, last_seen`,
			event.Fingerprint, event.Message, event.Route,
		).Scan(&result.Occurrences, &result.FirstSeen, &result.LastSeen)
		if err != nil {
			return domain.ErrorRecordResult{}, err
		}
	default:
		return domain.ErrorRecordResult{}, err
	}

	occurredAt := event.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO t_error_events (
			fingerprint, source, level, message, stack, component_stack,
			route, method, status_code, request_id, occurred_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		event.Fingerprint, event.Source, event.Level, event.Message,
		event.Stack, event.ComponentStack, event.Route, event.Method,
		event.StatusCode, event.RequestID, occurredAt,
	); err != nil {
		return domain.ErrorRecordResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ErrorRecordResult{}, err
	}
	return result, nil
}

// ClaimAlert memakai satu UPDATE bersyarat: dari banyak request yang melihat
// grup yang sama, hanya satu yang berhasil menetapkan last_alerted_at. Yang
// lain tidak mendapat baris dan tidak boleh mengirim alert.
func (r *sqlErrorStore) ClaimAlert(
	ctx context.Context,
	fingerprint string,
	now, notBefore time.Time,
) (bool, error) {
	var claimedAt time.Time
	err := r.pool.QueryRow(ctx, `
		UPDATE t_error_groups
		SET last_alerted_at = $2
		WHERE fingerprint = $1
		  AND (last_alerted_at IS NULL OR last_alerted_at <= $3)
		RETURNING last_alerted_at`,
		fingerprint, now.UTC(), notBefore.UTC(),
	).Scan(&claimedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ReleaseAlert membatalkan klaim hanya bila last_alerted_at masih sama dengan
// waktu klaim kita, sehingga tidak menghapus klaim milik request lain.
func (r *sqlErrorStore) ReleaseAlert(ctx context.Context, fingerprint string, claimedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE t_error_groups
		SET last_alerted_at = NULL
		WHERE fingerprint = $1 AND last_alerted_at = $2`,
		fingerprint, claimedAt.UTC())
	return err
}
