package usecase

import (
	"context"
	"errors"
	"go_backend/internal/domain"
)

type unitUsecase struct {
	unitRepo domain.UnitRepository
}

func NewUnitUsecase(repo domain.UnitRepository) domain.UnitUsecase {
	return &unitUsecase{unitRepo: repo}
}

func (u *unitUsecase) GetAllUnits(ctx context.Context) ([]domain.Unit, error) {
	return u.unitRepo.GetAllActive(ctx)
}

// Tambahkan di bagian bawah file unit_usecase.go Anda

func (u *unitUsecase) CreateUnit(ctx context.Context, unit *domain.Unit) error {
	if unit.Kode == "" {
		return errors.New("kode singkat satuan tidak boleh kosong")
	}
	if unit.Name == "" {
		return errors.New("nama panjang satuan tidak boleh kosong")
	}
	return u.unitRepo.Create(ctx, unit)
}
