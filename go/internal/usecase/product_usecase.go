package usecase

import (
	"context"
	"errors"
	"fmt"
	"go_backend/internal/domain"
	"math/rand"
)

type productUsecase struct {
	productRepo domain.ProductRepository
}

// NewProductUsecase menginisialisasi bisnis logis Master Produk
func NewProductUsecase(repo domain.ProductRepository) domain.ProductUsecase {
	return &productUsecase{productRepo: repo}
}

// Generator SKU Otomatis Berbasis Kategori dan Acak Unik Penanda Barang
func (u *productUsecase) generateAutoSKU(categoryID int32) string {
	randomNum := rand.Intn(899999) + 100000 // Menghasilkan 6 digit angka acak (100000 - 999999)
	return fmt.Sprintf("SKU-CAT%02d-%d", categoryID, randomNum)
}

func (u *productUsecase) CreateProduct(ctx context.Context, prod *domain.Product) error {
	if prod.Name == "" {
		return domain.NewBusinessRuleError("nama barang/produk tidak boleh kosong")
	}
	if prod.CategoryID <= 0 {
		return domain.NewBusinessRuleError("kategori barang wajib ditentukan secara valid")
	}
	if prod.SupplierID <= 0 {
		return domain.NewBusinessRuleError("mitra supplier utama wajib ditentukan secara valid")
	}
	if prod.SellingPrice < prod.PurchasePrice {
		return domain.NewBusinessRuleError("peringatan akuntansi: harga jual tidak boleh lebih rendah dari harga beli modal (HPP)")
	}
	if err := normalizeProductDiscount(prod); err != nil {
		return err
	}

	// Faktor konversi wajib >= 1. Saat satuan beli sama dengan satuan jual,
	// nilainya dipaksa 1 untuk menjaga konsistensi data.
	if prod.ConversionRate < 1 {
		prod.ConversionRate = 1
	}
	if prod.SellingUnitID == prod.PurchaseUnitID && prod.ConversionRate != 1 {
		prod.ConversionRate = 1
	}

	// Jika SKU tidak dikirim dari frontend, otomatis generate kode kaku internal.
	// Dua pembuatan produk bersamaan bisa menghasilkan SKU acak yang sama; unique
	// index di database tetap penentu akhir, dan SKU otomatis diulang dengan
	// angka baru. SKU yang diketik manual tidak diulang agar pengguna melihat
	// konfliknya.
	autoSKU := prod.Sku == ""
	if autoSKU {
		prod.Sku = u.generateAutoSKU(prod.CategoryID)
	}

	const maksPercobaanSKU = 5
	for attempt := 1; ; attempt++ {
		err := u.productRepo.Create(ctx, prod)
		if err == nil {
			return nil
		}
		var duplicate *domain.DuplicateError
		if !autoSKU || !errors.As(err, &duplicate) || attempt >= maksPercobaanSKU {
			return err
		}
		prod.Sku = u.generateAutoSKU(prod.CategoryID)
	}
}

func (u *productUsecase) GetAllProducts(ctx context.Context, search string, categoryFilter int32, cursorID int64, limit int, direction string) ([]domain.Product, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	return u.productRepo.GetProductsCursor(ctx, search, categoryFilter, cursorID, limit, direction)
}

func (u *productUsecase) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	if id <= 0 {
		return nil, domain.NewBusinessRuleError("id produk tidak valid")
	}
	return u.productRepo.GetByID(ctx, id)
}

func (u *productUsecase) UpdateProduct(ctx context.Context, prod *domain.Product) error {
	if prod.ID <= 0 {
		return domain.NewBusinessRuleError("id produk yang akan diperbarui tidak valid")
	}
	if prod.Name == "" {
		return domain.NewBusinessRuleError("nama barang/produk tidak boleh kosong")
	}
	if prod.SellingPrice < prod.PurchasePrice {
		return domain.NewBusinessRuleError("peringatan akuntansi: harga jual tidak boleh lebih rendah dari harga beli modal (HPP)")
	}
	if err := normalizeProductDiscount(prod); err != nil {
		return err
	}

	// Samakan penjagaan dengan CreateProduct supaya nilai warisan/klien lama yang
	// mengirim 0 tidak pernah sampai ke CHECK constraint m_products_conversion_rate_check.
	if prod.ConversionRate < 1 {
		prod.ConversionRate = 1
	}
	if prod.SellingUnitID == prod.PurchaseUnitID && prod.ConversionRate != 1 {
		prod.ConversionRate = 1
	}

	return u.productRepo.Update(ctx, prod)
}

func (u *productUsecase) DeleteProduct(ctx context.Context, id int64, deletedBy int64) error {
	if id <= 0 {
		return domain.NewBusinessRuleError("id produk yang akan dihapus tidak valid")
	}
	return u.productRepo.Delete(ctx, id, deletedBy)
}

// normalizeProductDiscount memastikan diskon yang dikirim admin selalu sah
// sebelum tersimpan ke m_products: tipenya salah satu dari daftar yang
// diperbolehkan, nilainya tidak negatif, dan persen tidak melebihi 100.
func normalizeProductDiscount(prod *domain.Product) error {
	switch prod.DiscountType {
	case "":
		// Klien lama yang tidak mengirim tipe dianggap tanpa diskon.
		prod.DiscountType = domain.ProductDiscountNone
		prod.DiscountValue = 0
		return nil
	case domain.ProductDiscountNone:
		prod.DiscountValue = 0
		return nil
	case domain.ProductDiscountPercent, domain.ProductDiscountAmount:
	default:
		return domain.NewBusinessRuleError("tipe diskon produk tidak dikenal, gunakan NONE, PERCENT, atau AMOUNT")
	}

	if prod.DiscountValue < 0 {
		return domain.NewBusinessRuleError("nilai diskon produk tidak boleh negatif")
	}
	if prod.DiscountType == domain.ProductDiscountPercent && prod.DiscountValue > 100 {
		return domain.NewBusinessRuleError("diskon persen produk tidak boleh lebih dari 100")
	}
	return nil
}
