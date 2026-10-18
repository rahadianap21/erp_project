package postgres

import (
	"context"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlAccountingSettingsRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewSQLAccountingSettingsRepository menginisialisasi repository pengaturan akun.
func NewSQLAccountingSettingsRepository(pool *pgxpool.Pool) domain.AccountingSettingsRepository {
	return &sqlAccountingSettingsRepository{pool: pool, q: db.New(pool)}
}

// ListMapping mengembalikan pemetaan mapping_key → account_id.
func (r *sqlAccountingSettingsRepository) ListMapping(ctx context.Context) (map[string]int64, error) {
	rows, err := r.q.ListAccountingSettings(ctx)
	if err != nil {
		return nil, err
	}
	mapping := make(map[string]int64, len(rows))
	for _, row := range rows {
		mapping[row.MappingKey] = row.AccountID
	}
	return mapping, nil
}
