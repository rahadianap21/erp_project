package domain

import (
	"context"
	"time"
)

// Supplier merepresentasikan entitas bisnis data supplier lengkap dengan informasi wilayahnya
type Supplier struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	PicName        string    `json:"pic_name"`
	Phone          string    `json:"phone"`
	Email          string    `json:"email"`
	Address        string    `json:"address"`
	VillageID      string    `json:"village_id"`
	VillageName    string    `json:"village_name,omitempty"`
	PostalCode     string    `json:"postal_code,omitempty"`
	DistrictID     string    `json:"district_id,omitempty"`
	DistrictName   string    `json:"district_name,omitempty"`
	RegencyID      string    `json:"regency_id,omitempty"`
	RegencyName    string    `json:"regency_name,omitempty"`
	ProvinceID     string    `json:"province_id,omitempty"`
	ProvinceName   string    `json:"province_name,omitempty"`
	BankID         *int32    `json:"bank_id,omitempty"`
	BankKode       string    `json:"bank_kode,omitempty"`
	BankName       string    `json:"bank_name,omitempty"`
	AccountNumber  string    `json:"account_number,omitempty"`
	AccountName    string    `json:"account_name,omitempty"`
	DefaultTaxType string    `json:"default_tax_type"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedBy      *int64    `json:"created_by,omitempty"`
	UpdatedBy      *int64    `json:"updated_by,omitempty"`
}

// SupplierRepository mendefinisikan kontrak komunikasi ke database
type SupplierRepository interface {
	Create(ctx context.Context, supplier *Supplier) error
	GetSuppliersCursor(ctx context.Context, search, province string, cursorID int64, limit int, direction string) ([]Supplier, int64, error)
	GetByID(ctx context.Context, id int64) (*Supplier, error)
	Update(ctx context.Context, supplier *Supplier) error
	Delete(ctx context.Context, id, deletedBy int64) error
}

// SupplierUsecase mendefinisikan kontrak logika bisnis sistem
type SupplierUsecase interface {
	CreateSupplier(ctx context.Context, supplier *Supplier) error
	GetAllSuppliers(ctx context.Context, search, province string, cursorID int64, limit int, direction string) ([]Supplier, int64, error)
	GetSupplierByID(ctx context.Context, id int64) (*Supplier, error)
	UpdateSupplier(ctx context.Context, supplier *Supplier) error
	DeleteSupplier(ctx context.Context, id, deletedBy int64) error
}
