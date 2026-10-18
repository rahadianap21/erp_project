package usecase

import (
	"context"
	"go_backend/internal/domain"
	"regexp"
	"strings"
)

type productCategoryUsecase struct {
	catRepo domain.ProductCategoryRepository
}

// NewProductCategoryUsecase menginisialisasi usecase master kategori produk
func NewProductCategoryUsecase(repo domain.ProductCategoryRepository) domain.ProductCategoryUsecase {
	return &productCategoryUsecase{catRepo: repo}
}

// Generator Slug Teks Otomatis bawaan Logika Bisnis ERP
func (u *productCategoryUsecase) generateSlug(title string) string {
	slug := strings.ToLower(title)
	// Ganti semua spasi, ampersand, atau tanda baca menjadi tanda strip tunggal
	reg := regexp.MustCompile(`[^a-z0-9\s-_]`)
	slug = reg.ReplaceAllString(slug, "")
	slug = strings.ReplaceAll(slug, " ", "-")

	// Bersihkan jika ada strip ganda berurutan (--)
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return strings.Trim(slug, "-")
}

func (u *productCategoryUsecase) CreateCategory(ctx context.Context, cat *domain.ProductCategory) error {
	if cat.Name == "" {
		return domain.NewBusinessRuleError("nama kategori produk wajib diisi")
	}
	// Otomatis buat slug ramah URL dari nama jika tidak diisi manual
	if cat.Slug == "" {
		cat.Slug = u.generateSlug(cat.Name)
	}
	return u.catRepo.Create(ctx, cat)
}

func (u *productCategoryUsecase) GetAllCategories(ctx context.Context, search string, cursorID int32, limit int, direction string) ([]domain.ProductCategory, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	return u.catRepo.GetCategoriesCursor(ctx, search, cursorID, limit, direction)
}

func (u *productCategoryUsecase) GetCategoryByID(ctx context.Context, id int32) (*domain.ProductCategory, error) {
	if id <= 0 {
		return nil, domain.NewBusinessRuleError("id kategori tidak valid")
	}
	return u.catRepo.GetByID(ctx, id)
}

func (u *productCategoryUsecase) UpdateCategory(ctx context.Context, cat *domain.ProductCategory) error {
	if cat.ID <= 0 {
		return domain.NewBusinessRuleError("id kategori yang akan diubah tidak valid")
	}
	if cat.Name == "" {
		return domain.NewBusinessRuleError("nama kategori tidak boleh kosong")
	}
	if cat.Slug == "" {
		cat.Slug = u.generateSlug(cat.Name)
	}
	return u.catRepo.Update(ctx, cat)
}

func (u *productCategoryUsecase) DeleteCategory(ctx context.Context, id int32, deletedBy int64) error {
	if id <= 0 {
		return domain.NewBusinessRuleError("id kategori yang akan dihapus tidak valid")
	}
	return u.catRepo.Delete(ctx, id, deletedBy)
}
