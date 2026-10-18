package postgres

import (
	"context"
	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlRegionRepository struct {
	q *db.Queries
}

func NewSQLRegionRepository(pool *pgxpool.Pool) domain.RegionRepository {
	return &sqlRegionRepository{q: db.New(pool)}
}

func (r *sqlRegionRepository) GetProvinces(ctx context.Context) ([]domain.Region, error) {
	rows, err := r.q.GetProvinces(ctx)
	if err != nil {
		return nil, err
	}
	var res []domain.Region
	for _, row := range rows {
		res = append(res, domain.Region{ID: row.ID, Name: row.Name})
	}
	return res, nil
}

func (r *sqlRegionRepository) GetRegenciesByProvinceID(ctx context.Context, provinceID string) ([]domain.Region, error) {
	rows, err := r.q.GetRegenciesByProvinceID(ctx, provinceID)
	if err != nil {
		return nil, err
	}
	var res []domain.Region
	for _, row := range rows {
		res = append(res, domain.Region{ID: row.ID, Name: row.Name})
	}
	return res, nil
}

func (r *sqlRegionRepository) GetDistrictsByRegencyID(ctx context.Context, regencyID string) ([]domain.Region, error) {
	rows, err := r.q.GetDistrictsByRegencyID(ctx, regencyID)
	if err != nil {
		return nil, err
	}
	var res []domain.Region
	for _, row := range rows {
		res = append(res, domain.Region{ID: row.ID, Name: row.Name})
	}
	return res, nil
}

func (r *sqlRegionRepository) GetVillagesByDistrictID(ctx context.Context, districtID string) ([]domain.VillageRegion, error) {
	rows, err := r.q.GetVillagesByDistrictID(ctx, districtID)
	if err != nil {
		return nil, err
	}
	var res []domain.VillageRegion
	for _, row := range rows {
		postalCode := ""
		if row.PostalCode.Valid {
			postalCode = row.PostalCode.String
		}
		res = append(res, domain.VillageRegion{ID: row.ID, Name: row.Name, PostalCode: postalCode})
	}
	return res, nil
}
