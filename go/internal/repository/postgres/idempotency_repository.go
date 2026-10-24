package postgres

import (
	"context"
	"errors"

	"go_backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlIdempotencyRepository struct {
	pool *pgxpool.Pool
}

func NewSQLIdempotencyRepository(pool *pgxpool.Pool) domain.IdempotencyStore {
	return &sqlIdempotencyRepository{pool: pool}
}

func (r *sqlIdempotencyRepository) Claim(ctx context.Context, userID int64, scope, key, requestHash string) (*domain.IdempotencyRecord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Hanya respons yang sudah selesai yang boleh kedaluwarsa. Klaim
	// IN_PROGRESS sengaja TIDAK pernah dihapus otomatis: proses bisa saja mati
	// setelah transaksi bisnis commit tetapi sebelum respons tersimpan. Kalau
	// klaim seperti itu diambil alih, retry akan membuat dokumen kedua.
	if _, err := tx.Exec(ctx, `
		DELETE FROM t_idempotency_keys
		WHERE user_id = $1 AND scope = $2 AND idempotency_key = $3
		  AND status = 'COMPLETED' AND expires_at <= NOW()`, userID, scope, key); err != nil {
		return nil, err
	}

	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO t_idempotency_keys (user_id, scope, idempotency_key, request_hash)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, scope, idempotency_key) DO NOTHING
		RETURNING id`, userID, scope, key, requestHash).Scan(&id)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return &domain.IdempotencyRecord{ID: id, RequestHash: requestHash}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	var (
		requestHashDB string
		status        string
		responseCode  pgtype.Int4
		responseBody  []byte
		contentType   pgtype.Text
	)
	err = tx.QueryRow(ctx, `
		SELECT id, request_hash, status, response_status, response_body,
		       response_content_type
		FROM t_idempotency_keys
		WHERE user_id = $1 AND scope = $2 AND idempotency_key = $3
		FOR UPDATE`, userID, scope, key).
		Scan(&id, &requestHashDB, &status, &responseCode, &responseBody, &contentType)
	if err != nil {
		return nil, err
	}

	if requestHashDB != requestHash {
		return nil, domain.ErrIdempotencyKeyReused
	}

	if status == "COMPLETED" {
		rec := &domain.IdempotencyRecord{
			ID:           id,
			RequestHash:  requestHashDB,
			Completed:    true,
			ResponseBody: responseBody,
		}
		if responseCode.Valid {
			rec.ResponseStatus = int(responseCode.Int32)
		}
		if contentType.Valid {
			rec.ResponseContentType = contentType.String
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return rec, nil
	}

	// Klaim yang belum selesai tidak boleh diambil alih otomatis. Status hasil
	// transaksi bisnisnya tidak diketahui, jadi mengeksekusi ulang berisiko
	// menggandakan dokumen. Baris ini perlu diperiksa/dibersihkan manual.
	return nil, domain.ErrIdempotencyInProgress
}

func (r *sqlIdempotencyRepository) Complete(ctx context.Context, id int64, status int, body []byte, contentType string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE t_idempotency_keys
		SET status = 'COMPLETED', response_status = $2, response_body = $3,
		    response_content_type = $4, updated_at = NOW()
		WHERE id = $1 AND status = 'IN_PROGRESS'`,
		id, status, body, contentType)
	return err
}

func (r *sqlIdempotencyRepository) Release(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM t_idempotency_keys
		WHERE id = $1 AND status = 'IN_PROGRESS'`, id)
	return err
}
