package usecase

import (
	"context"
	"errors"
	"testing"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres"
)

// TestDeleteCategory_0BarisMemunculkanErrKategoriTidakDitemukan menjaga perbaikan
// penghapusan kategori produk.
//
// Dulu query DeleteProductCategory memakai :exec dan RowsAffected dibuang, sehingga
// menghapus ID yang tidak ada — atau yang sudah pernah soft delete — tetap membalas
// HTTP 200 "Kategori produk berhasil dihapus". Salah ketik ID jadi tertutupi dan
// pengguna mengira datanya sudah hilang padahal baris tidak tersentuh.
//
// Perbaikan: query jadi :execrows, 0 baris dipetakan ke ErrKategoriTidakDitemukan
// lalu handler membalas 404.
//
// Test butuh PostgreSQL nyata; dilewati bila DATABASE_URL tidak tersedia.
func TestDeleteCategory_0BarisMemunculkanErrKategoriTidakDitemukan(t *testing.T) {
	pool := poolUji(t)
	ctx := context.Background()
	uc := NewProductCategoryUsecase(postgres.NewSQLProductCategoryRepository(pool))

	cat := &domain.ProductCategory{Name: "Kategori Uji Hapus", Slug: "kategori-uji-hapus"}
	if err := uc.CreateCategory(ctx, cat); err != nil {
		t.Fatalf("gagal membuat kategori uji: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM m_product_categories WHERE id = $1`, cat.ID)
	})

	// 1. ID yang tidak ada: harus not-found, bukan sukses palsu.
	if err := uc.DeleteCategory(ctx, 999999999, 1); !errors.Is(err, domain.ErrKategoriTidakDitemukan) {
		t.Errorf("hapus ID tidak ada: err = %v, want ErrKategoriTidakDitemukan", err)
	}

	// 2. Kategori yang baru dibuat bisa dihapus.
	if err := uc.DeleteCategory(ctx, cat.ID, 1); err != nil {
		t.Errorf("hapus kategori valid: err = %v, want nil", err)
	}

	// 3. Soft delete kedua kali dianggap tidak ditemukan, bukan sukses.
	if err := uc.DeleteCategory(ctx, cat.ID, 1); !errors.Is(err, domain.ErrKategoriTidakDitemukan) {
		t.Errorf("hapus ulang (sudah soft delete): err = %v, want ErrKategoriTidakDitemukan", err)
	}
}
