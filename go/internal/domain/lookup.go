package domain

import "context"

// SupplierLookup adalah proyeksi ringan master supplier yang hanya memuat
// field yang dibutuhkan form Purchase Order untuk mengisi dropdown.
type SupplierLookup struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	DefaultTaxType string `json:"default_tax_type,omitempty"`
}

// ProductLookup adalah proyeksi ringan master barang untuk form Purchase Order.
type ProductLookup struct {
	ID               int64   `json:"id"`
	Sku              string  `json:"sku"`
	Name             string  `json:"name"`
	SupplierID       int64   `json:"supplier_id"`
	PurchasePrice    float64 `json:"purchase_price"`
	PurchaseUnitKode string  `json:"purchase_unit_kode,omitempty"`
}

// LookupRepository menyediakan data referensi read-only yang tidak terikat
// pada modul master, sehingga form PO tidak perlu Izin suppliers:read /
// products:read untuk bisa menampilkan nama pada dropdown.
type LookupRepository interface {
	GetSuppliersForPO(ctx context.Context) ([]SupplierLookup, error)
	GetProductsForPO(ctx context.Context, supplierID int64) ([]ProductLookup, error)
}

type LookupUsecase interface {
	GetSuppliersForPO(ctx context.Context) ([]SupplierLookup, error)
	GetProductsForPO(ctx context.Context, supplierID int64) ([]ProductLookup, error)
}
