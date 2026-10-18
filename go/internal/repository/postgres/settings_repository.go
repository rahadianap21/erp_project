package postgres

import (
	"context"
	"database/sql"
	"errors"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlSettingsRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewSQLSettingsRepository(pool *pgxpool.Pool) domain.SettingsRepository {
	return &sqlSettingsRepository{
		pool: pool,
		q:    db.New(pool),
	}
}

func (r *sqlSettingsRepository) List(ctx context.Context) ([]domain.AppSetting, error) {
	rows, err := r.q.ListAppSettings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AppSetting, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAppSetting(row.Key, row.Value, row.Description, row.UpdatedAt, row.UpdatedBy))
	}
	return out, nil
}

func (r *sqlSettingsRepository) Get(ctx context.Context, key string) (*domain.AppSetting, error) {
	row, err := r.q.GetAppSetting(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	s := mapAppSetting(row.Key, row.Value, row.Description, row.UpdatedAt, row.UpdatedBy)
	return &s, nil
}

func (r *sqlSettingsRepository) Upsert(ctx context.Context, key, value, description string, actorID int64) error {
	var actor pgtype.Int8
	if actorID > 0 {
		actor = pgtype.Int8{Int64: actorID, Valid: true}
	}
	return r.q.UpsertAppSetting(ctx, db.UpsertAppSettingParams{
		Key:         key,
		Value:       value,
		Description: description,
		UpdatedBy:   actor,
	})
}

func mapAppSetting(key, value, description string, updatedAt pgtype.Timestamptz, updatedBy pgtype.Int8) domain.AppSetting {
	s := domain.AppSetting{
		Key:         key,
		Value:       value,
		Description: description,
	}
	if updatedAt.Valid {
		s.UpdatedAt = updatedAt.Time
	}
	if updatedBy.Valid {
		id := updatedBy.Int64
		s.UpdatedBy = &id
	}
	return s
}
