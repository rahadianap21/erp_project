package postgres

import (
	"context"
	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlUnitRepository struct {
	q *db.Queries
}

func NewSQLUnitRepository(pool *pgxpool.Pool) domain.UnitRepository {
	return &sqlUnitRepository{q: db.New(pool)}
}

func (r *sqlUnitRepository) GetAllActive(ctx context.Context) ([]domain.Unit, error) {
	rows, err := r.q.GetAllActiveUnits(ctx)
	if err != nil {
		return nil, err
	}

	// Gunakan make array kosong biar saat DB kosong JSON melahirkan [] bukan null
	res := make([]domain.Unit, 0)
	for _, row := range rows {
		res = append(res, domain.Unit{
			ID:          row.ID,
			Kode:        row.Kode,
			Name:        row.Name,
			Description: r.interfaceToString(row.Description),
		})
	}
	return res, nil
}

func (r *sqlUnitRepository) Create(ctx context.Context, unit *domain.Unit) error {
	var createdByPg pgtype.Int8
	if unit.CreatedBy != nil && *unit.CreatedBy > 0 {
		createdByPg = pgtype.Int8{Int64: *unit.CreatedBy, Valid: true}
	} else {
		createdByPg = pgtype.Int8{Valid: false}
	}

	var updatedByPg pgtype.Int8
	if unit.UpdatedBy != nil && *unit.UpdatedBy > 0 {
		updatedByPg = pgtype.Int8{Int64: *unit.UpdatedBy, Valid: true}
	} else {
		updatedByPg = pgtype.Int8{Valid: false}
	}

	row, err := r.q.CreateUnit(ctx, db.CreateUnitParams{
		Kode:        unit.Kode,
		Name:        unit.Name,
		Description: unit.Description,
		CreatedBy:   createdByPg,
		UpdatedBy:   updatedByPg,
	})
	if err != nil {
		return err
	}

	unit.ID = row.ID
	return nil
}

func (r *sqlUnitRepository) interfaceToString(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case pgtype.Text:
		if v.Valid {
			return v.String
		}
	}
	return ""
}
