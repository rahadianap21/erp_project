package usecase

import (
	"context"
	"go_backend/internal/domain"
)

type bankUsecase struct {
	bankRepo domain.BankRepository
}

// NewBankUsecase menginisialisasi BankUsecase terpusat yang terpusat
func NewBankUsecase(repo domain.BankRepository) domain.BankUsecase {
	return &bankUsecase{bankRepo: repo}
}

// GetAllBanks mengambil seluruh data master bank aktif dari level data repository
func (u *bankUsecase) GetAllBanks(ctx context.Context) ([]domain.Bank, error) {
	// Panggil data langsung dari layer repositori database postgres
	banks, err := u.bankRepo.GetAllActive(ctx)
	if err != nil {
		return nil, err
	}

	// Antisipasi Nil-Safety: Jika database kosong murni, kembalikan slice kosong [] bukan null
	if banks == nil {
		return []domain.Bank{}, nil
	}

	return banks, nil
}
