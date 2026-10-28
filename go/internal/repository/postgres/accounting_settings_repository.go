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
func NewSQLAccountingSettingsRepository(pool *pgxpool.Pool) domain.AccountingSettingsStore {
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

// ListDetails mengembalikan pemetaan beserta kode, nama, dan status akun tujuan.
func (r *sqlAccountingSettingsRepository) ListDetails(ctx context.Context) ([]domain.AccountingSettingDetail, error) {
	rows, err := r.q.ListAccountingSettingDetails(ctx)
	if err != nil {
		return nil, err
	}
	details := make([]domain.AccountingSettingDetail, 0, len(rows))
	for _, row := range rows {
		details = append(details, domain.AccountingSettingDetail{
			MappingKey:    row.SMappingKey,
			Description:   row.Description.String,
			AccountID:     row.AccountID,
			AccountCode:   row.AccountCode,
			AccountName:   row.AccountName,
			AccountActive: row.AccountActive,
		})
	}
	return details, nil
}

// UpsertMany menyimpan beberapa perubahan pemetaan dalam satu transaksi agar
// pengaturan tidak pernah tersimpan separuh saat salah satu baris gagal.
func (r *sqlAccountingSettingsRepository) UpsertMany(ctx context.Context, settings []domain.AccountingSetting, actorID int64) error {
	if len(settings) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)
	for _, setting := range settings {
		if err := qtx.UpsertAccountingSetting(ctx, db.UpsertAccountingSettingParams{
			MappingKey: setting.MappingKey,
			AccountID:  setting.AccountID,
			UpdatedBy:  actorID,
		}); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
