package domain

import "context"

type Unit struct {
	ID          int32  `json:"id"`
	Kode        string `json:"kode"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedBy   *int64 `json:"created_by,omitempty"`
	UpdatedBy   *int64 `json:"updated_by,omitempty"`
}

type UnitRepository interface {
	GetAllActive(ctx context.Context) ([]Unit, error)
	Create(ctx context.Context, unit *Unit) error // 🌟 TAMBAHKAN KONTRAK REPO
}

type UnitUsecase interface {
	GetAllUnits(ctx context.Context) ([]Unit, error)
	CreateUnit(ctx context.Context, unit *Unit) error // 🌟 TAMBAHKAN KONTRAK USECASE
}
