package domain

import "context"

type Bank struct {
	ID        int32  `json:"id"`   // 🌟 Berubah menjadi tipe angka (SERIAL di DB)
	Kode      string `json:"kode"` // 🌟 Properti baru untuk menyimpan sandi kliring bank
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
	IsActive  bool   `json:"is_active"`
}

type BankRepository interface {
	GetAllActive(ctx context.Context) ([]Bank, error)
}

type BankUsecase interface {
	GetAllBanks(ctx context.Context) ([]Bank, error)
}
