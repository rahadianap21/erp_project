package usecase

import (
	"context"
	"go_backend/internal/domain"
)

type regionUsecase struct {
	regionRepo domain.RegionRepository
}

func NewRegionUsecase(repo domain.RegionRepository) domain.RegionUsecase {
	return &regionUsecase{regionRepo: repo}
}

func (u *regionUsecase) GetProvinces(ctx context.Context) ([]domain.Region, error) {
	return u.regionRepo.GetProvinces(ctx)
}

func (u *regionUsecase) GetRegencies(ctx context.Context, provinceID string) ([]domain.Region, error) {
	if provinceID == "" {
		return []domain.Region{}, nil
	}
	return u.regionRepo.GetRegenciesByProvinceID(ctx, provinceID)
}

func (u *regionUsecase) GetDistricts(ctx context.Context, regencyID string) ([]domain.Region, error) {
	if regencyID == "" {
		return []domain.Region{}, nil
	}
	return u.regionRepo.GetDistrictsByRegencyID(ctx, regencyID)
}

func (u *regionUsecase) GetVillages(ctx context.Context, districtID string) ([]domain.VillageRegion, error) {
	if districtID == "" {
		return []domain.VillageRegion{}, nil
	}
	return u.regionRepo.GetVillagesByDistrictID(ctx, districtID)
}
