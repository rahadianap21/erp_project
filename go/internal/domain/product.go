package domain

import (
	"context"
	"time"
)

// Tipe diskon default produk. Nilainya sama dengan tipe diskon penjualan
// supaya diskon dari master produk langsung bisa dipakai baris POS.
const (
	ProductDiscountNone    = "NONE"
	ProductDiscountPercent = "PERCENT"
	ProductDiscountAmount  = "AMOUNT"
)

// Product merepresentasikan data master barang/produk jadi maupun bahan baku sistem ERP.
type Product struct {
	ID           int64  `json:"id"`
	Sku          string `json:"sku"`
	Barcode      string `json:"barcode,omitempty"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	CategoryID   int32  `json:"category_id"`
	CategoryName string `json:"category_name,omitempty"`
	CategorySlug string `json:"category_slug,omitempty"`
	SupplierID   int64  `json:"supplier_id"`
	SupplierName string `json:"supplier_name,omitempty"`
	SupplierPic  string `json:"supplier_pic,omitempty"`

	// 🌟 PERBAIKAN: Pisah properti Selling UoM & Purchase UoM
	SellingUnitID   int32  `json:"selling_unit_id"`
	SellingUnitKode string `json:"selling_unit_kode,omitempty"`
	SellingUnitName string `json:"selling_unit_name,omitempty"`

	PurchaseUnitID   int32  `json:"purchase_unit_id"`
	PurchaseUnitKode string `json:"purchase_unit_kode,omitempty"`
	PurchaseUnitName string `json:"purchase_unit_name,omitempty"`

	// ConversionRate menyatakan berapa banyak satuan jual yang didapat dari satu
	// satuan beli. Nilainya 1 kalau satuan beli sama dengan satuan jual, dan
	// tidak boleh kurang dari 1 karena selling_unit_id bermakna "satuan terkecil".
	//
	// Kolom ini ada karena m_units tidak menyimpan kelipatan satuan apa pun.
	// m_units tidak menyimpan kelipatan satuan apa pun. current_stock dinyatakan
	// dalam satuan jual, jadi penerimaan barang wajib mengalikan jumlah yang
	// diterima dengan faktor ini sebelum menambah stok.
	ConversionRate int32 `json:"conversion_rate"`

	PurchasePrice float64 `json:"purchase_price"`
	SellingPrice  float64 `json:"selling_price"`
	MinStockLevel int32   `json:"min_stock_level"`
	CurrentStock  int32   `json:"current_stock"`

	// AvgCost adalah harga pokok rata-rata per SATUAN JUAL, hasil penimbangan
	// seluruh penerimaan barang yang pernah ditutup untuk produk ini. Dipakai
	// sebagai dasar penilaian persediaan dan harga pokok penjualan, dan selalu
	// selaras dengan nilai yang dijurnal ke akun Persediaan saat penerimaan.
	AvgCost float64 `json:"avg_cost"`

	// DiscountType dan DiscountValue adalah potongan harga default yang sudah
	// ditetapkan admin per produk. Kasir cukup memilih produk di POS dan
	// diskon ini terbawa otomatis sebagai nilai awal baris penjualan.
	DiscountType  string  `json:"discount_type"`
	DiscountValue float64 `json:"discount_value"`

	// IsTaxable menandai produk kena PPN saat dijual. Produk lama default TRUE
	// karena usaha sudah PKP.
	IsTaxable bool `json:"is_taxable"`

	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy *int64    `json:"created_by,omitempty"`
	UpdatedBy *int64    `json:"updated_by,omitempty"`
}

// ProductRepository mendefinisikan kontrak komunikasi data kaku ke PostgreSQL database.
type ProductRepository interface {
	Create(ctx context.Context, prod *Product) error
	GetProductsCursor(ctx context.Context, search string, categoryFilter int32, cursorID int64, limit int, direction string) ([]Product, int64, error)
	GetByID(ctx context.Context, id int64) (*Product, error)
	Update(ctx context.Context, prod *Product) error
	Delete(ctx context.Context, id int64, deletedBy int64) error
}

// ProductUsecase mendefinisikan aturan dan otorisasi kontrol bisnis logika sistem ERP Axion.
type ProductUsecase interface {
	CreateProduct(ctx context.Context, prod *Product) error
	GetAllProducts(ctx context.Context, search string, categoryFilter int32, cursorID int64, limit int, direction string) ([]Product, int64, error)
	GetProductByID(ctx context.Context, id int64) (*Product, error)
	UpdateProduct(ctx context.Context, prod *Product) error
	DeleteProduct(ctx context.Context, id int64, deletedBy int64) error
}
