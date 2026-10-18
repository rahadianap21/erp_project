package domain

import "context"

type Region struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type VillageRegion struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PostalCode string `json:"postal_code"`
}

type RegionRepository interface {
	GetProvinces(ctx context.Context) ([]Region, error)
	GetRegenciesByProvinceID(ctx context.Context, provinceID string) ([]Region, error)
	GetDistrictsByRegencyID(ctx context.Context, regencyID string) ([]Region, error)
	GetVillagesByDistrictID(ctx context.Context, districtID string) ([]VillageRegion, error)
}

type RegionUsecase interface {
	GetProvinces(ctx context.Context) ([]Region, error)
	GetRegencies(ctx context.Context, provinceID string) ([]Region, error)
	GetDistricts(ctx context.Context, regencyID string) ([]Region, error)
	GetVillages(ctx context.Context, districtID string) ([]VillageRegion, error)
}
