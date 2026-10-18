package domain

import (
	"context"
	"errors"
	"time"
)

// ErrKategoriTidakDitemukan dipakai ketika sebuah aksi menyasar kategori yang
// tidak ada atau sudah soft delete (deleted_at IS NULL gagal cocok).
//
// Dulu penghapusan mengabaikan RowsAffected, sehingga menghapus kategori yang
// tidak ada justru membalas 200 "berhasil dihapus" — menutupi salah ketik ID
// dan membuat pengguna mengira data sudah hilang padahal tidak tersentuh.
var ErrKategoriTidakDitemukan = errors.New("kategori produk tidak ditemukan")

// ProductCategory merepresentasikan entitas bisnis data master kategori produk.
// Mendukung struktur pohon (Self-Referencing) untuk tingkatan Sub-Kategori.
type ProductCategory struct {
	ID          int32      `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description,omitempty"`
	ParentID    *int32     `json:"parent_id,omitempty"`   // Pointer ke ID induk (NULL jika kategori utama)
	ParentName  string     `json:"parent_name,omitempty"` // Menyimpan nama kategori induk hasil LEFT JOIN
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CreatedBy   *int64     `json:"created_by,omitempty"`
	UpdatedBy   *int64     `json:"updated_by,omitempty"`
	DeletedBy   *int64     `json:"deleted_by,omitempty"`
}

// ProductCategoryRepository mendefinisikan kontrak komunikasi data kaku ke PostgreSQL database.
type ProductCategoryRepository interface {
	Create(ctx context.Context, cat *ProductCategory) error
	GetCategoriesCursor(ctx context.Context, search string, cursorID int32, limit int, direction string) ([]ProductCategory, int64, error)
	GetByID(ctx context.Context, id int32) (*ProductCategory, error)
	Update(ctx context.Context, cat *ProductCategory) error
	Delete(ctx context.Context, id int32, deletedBy int64) error
}

// ProductCategoryUsecase mendefinisikan aturan dan otorisasi kontrol bisnis logika sistem ERP.
type ProductCategoryUsecase interface {
	CreateCategory(ctx context.Context, cat *ProductCategory) error
	GetAllCategories(ctx context.Context, search string, cursorID int32, limit int, direction string) ([]ProductCategory, int64, error)
	GetCategoryByID(ctx context.Context, id int32) (*ProductCategory, error)
	UpdateCategory(ctx context.Context, cat *ProductCategory) error
	DeleteCategory(ctx context.Context, id int32, deletedBy int64) error
}
