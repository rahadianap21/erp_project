package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlProductRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewSQLProductRepository menginisialisasi implementasi ProductRepository berbasis pgxpool
func NewSQLProductRepository(pool *pgxpool.Pool) domain.ProductRepository {
	return &sqlProductRepository{
		pool: pool,
		q:    db.New(pool),
	}
}

func (r *sqlProductRepository) Create(ctx context.Context, prod *domain.Product) error {
	var barcodePg pgtype.Text
	if prod.Barcode != "" {
		barcodePg = pgtype.Text{String: prod.Barcode, Valid: true}
	} else {
		barcodePg = pgtype.Text{Valid: false}
	}

	var descPg pgtype.Text
	if prod.Description != "" {
		descPg = pgtype.Text{String: prod.Description, Valid: true}
	} else {
		descPg = pgtype.Text{Valid: false}
	}

	var createdByPg pgtype.Int8
	if prod.CreatedBy != nil && *prod.CreatedBy > 0 {
		createdByPg = pgtype.Int8{Int64: *prod.CreatedBy, Valid: true}
	} else {
		createdByPg = pgtype.Int8{Valid: false}
	}

	var updatedByPg pgtype.Int8
	if prod.UpdatedBy != nil && *prod.UpdatedBy > 0 {
		updatedByPg = pgtype.Int8{Int64: *prod.UpdatedBy, Valid: true}
	} else {
		updatedByPg = pgtype.Int8{Valid: false}
	}

	row, err := r.q.CreateProduct(ctx, db.CreateProductParams{
		Sku:            prod.Sku,
		Barcode:        barcodePg,
		Name:           prod.Name,
		Description:    descPg,
		CategoryID:     prod.CategoryID,
		SupplierID:     prod.SupplierID,
		SellingUnitID:  prod.SellingUnitID,
		PurchaseUnitID: prod.PurchaseUnitID,
		ConversionRate: prod.ConversionRate,
		PurchasePrice:  r.float64ToNumeric(prod.PurchasePrice),
		SellingPrice:   r.float64ToNumeric(prod.SellingPrice),
		MinStockLevel:  prod.MinStockLevel,
		DiscountType:   defaultDiscountType(prod.DiscountType),
		DiscountValue:  r.float64ToNumeric(prod.DiscountValue),
		IsTaxable:      prod.IsTaxable,
		CurrentStock:   prod.CurrentStock,
		CreatedBy:      createdByPg,
		UpdatedBy:      updatedByPg,
	})
	if err != nil {
		return err
	}

	prod.ID = row.ID
	prod.IsTaxable = row.IsTaxable
	prod.IsActive = row.IsActive
	prod.CreatedAt = r.interfaceToTime(row.CreatedAt)
	return nil
}

func (r *sqlProductRepository) GetProductsCursor(ctx context.Context, search string, categoryFilter int32, cursorID int64, limit int, direction string) ([]domain.Product, int64, error) {
	products := make([]domain.Product, 0)

	totalCount, err := r.q.CountProducts(ctx, db.CountProductsParams{
		SearchQuery:    search,
		CategoryFilter: categoryFilter,
	})
	if err != nil {
		return nil, 0, err
	}

	if direction == "prev" {
		rows, err := r.q.GetProductsCursorPrev(ctx, db.GetProductsCursorPrevParams{
			SearchQuery:    search,
			CategoryFilter: categoryFilter,
			CursorID:       cursorID,
			ReqLimit:       int32(limit),
		})
		if err != nil {
			return nil, 0, err
		}
		reverseRows(rows)
		for _, row := range rows {
			products = append(products, r.mapRowToDomain(row))
		}
	} else {
		rows, err := r.q.GetProductsCursorNext(ctx, db.GetProductsCursorNextParams{
			SearchQuery:    search,
			CategoryFilter: categoryFilter,
			CursorID:       cursorID,
			ReqLimit:       int32(limit),
		})
		if err != nil {
			return nil, 0, err
		}
		for _, row := range rows {
			products = append(products, r.mapRowToDomain(row))
		}
	}

	return products, totalCount, nil
}

func (r *sqlProductRepository) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
	row, err := r.q.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &domain.Product{
		ID:             row.ID,
		Sku:            row.Sku,
		Barcode:        r.interfaceToString(row.Barcode),
		Name:           row.Name,
		Description:    r.interfaceToString(row.Description),
		CategoryID:     row.CategoryID,
		SupplierID:     row.SupplierID,
		SellingUnitID:  row.SellingUnitID,
		PurchaseUnitID: row.PurchaseUnitID,
		ConversionRate: row.ConversionRate,
		CategoryName:   r.interfaceToString(row.CategoryName),
		SupplierName:   r.interfaceToString(row.SupplierName),
		PurchasePrice:  r.numericToFloat64(row.ProdPurchasePrice),
		SellingPrice:   r.numericToFloat64(row.ProdSellingPrice),
		MinStockLevel:  row.ProdMinStock,
		DiscountType:   row.ProdDiscountType,
		DiscountValue:  r.numericToFloat64(row.ProdDiscountValue),
		CurrentStock:   row.ProdCurrentStock,
		AvgCost:        r.numericToFloat64(row.ProdAvgCost),
		IsTaxable:      row.IsTaxable,
		IsActive:       row.IsActive,
		CreatedAt:      r.interfaceToTime(row.CreatedAt),
	}, nil
}

func (r *sqlProductRepository) Update(ctx context.Context, prod *domain.Product) error {
	var barcodePg pgtype.Text
	if prod.Barcode != "" {
		barcodePg = pgtype.Text{String: prod.Barcode, Valid: true}
	} else {
		barcodePg = pgtype.Text{Valid: false}
	}

	var descPg pgtype.Text
	if prod.Description != "" {
		descPg = pgtype.Text{String: prod.Description, Valid: true}
	} else {
		descPg = pgtype.Text{Valid: false}
	}

	var updatedByPg pgtype.Int8
	if prod.UpdatedBy != nil && *prod.UpdatedBy > 0 {
		updatedByPg = pgtype.Int8{Int64: *prod.UpdatedBy, Valid: true}
	} else {
		updatedByPg = pgtype.Int8{Valid: false}
	}

	return r.q.UpdateProduct(ctx, db.UpdateProductParams{
		ID:             prod.ID,
		Sku:            prod.Sku,
		Barcode:        barcodePg,
		Name:           prod.Name,
		Description:    descPg,
		CategoryID:     prod.CategoryID,
		SupplierID:     prod.SupplierID,
		SellingUnitID:  prod.SellingUnitID,  // 🌟 Ganti pemanggilan UnitOfMeasure lama menjadi UnitID
		PurchaseUnitID: prod.PurchaseUnitID, // 🌟 Ganti pemanggilan UnitOfMeasure lama menjadi UnitID
		ConversionRate: prod.ConversionRate,
		PurchasePrice:  r.float64ToNumeric(prod.PurchasePrice),
		SellingPrice:   r.float64ToNumeric(prod.SellingPrice),
		MinStockLevel:  prod.MinStockLevel,
		DiscountType:   defaultDiscountType(prod.DiscountType),
		DiscountValue:  r.float64ToNumeric(prod.DiscountValue),
		IsTaxable:      prod.IsTaxable,
		IsActive:       prod.IsActive,
		UpdatedBy:      updatedByPg,
	})
}

func (r *sqlProductRepository) Delete(ctx context.Context, id int64, deletedBy int64) error {
	deletedByPg := pgtype.Int8{Int64: deletedBy, Valid: true}
	return r.q.DeleteProduct(ctx, db.DeleteProductParams{
		ID:        id,
		DeletedBy: deletedByPg,
	})
}

// =========================================================================
// 🔒 UTILITY HELPERS: PENYELESAI AKURASI AKUNTANSI DAN NIL-SAFETY
// =========================================================================

func (r *sqlProductRepository) mapRowToDomain(row any) domain.Product {
	p := domain.Product{}
	switch v := row.(type) {
	case db.GetProductsCursorNextRow:
		p.ID = v.ID
		p.Sku = v.Sku
		p.Barcode = r.interfaceToString(v.Barcode)
		p.Name = v.Name
		p.Description = r.interfaceToString(v.Description)
		p.CategoryID = v.CategoryID
		p.SupplierID = v.SupplierID
		p.CategoryName = r.interfaceToString(v.CategoryName)
		p.CategorySlug = r.interfaceToString(v.CategorySlug)
		p.SupplierName = r.interfaceToString(v.SupplierName)
		p.SupplierPic = r.interfaceToString(v.SupplierPic)
		p.SellingUnitID = v.SellingUnitID
		p.SellingUnitKode = r.interfaceToString(v.SellingUnitKode)
		p.SellingUnitName = r.interfaceToString(v.SellingUnitName)

		p.PurchaseUnitID = v.PurchaseUnitID
		p.PurchaseUnitKode = r.interfaceToString(v.PurchaseUnitKode)
		p.PurchaseUnitName = r.interfaceToString(v.PurchaseUnitName)
		p.PurchasePrice = r.numericToFloat64(v.ProdPurchasePrice)
		p.SellingPrice = r.numericToFloat64(v.ProdSellingPrice)
		p.MinStockLevel = v.ProdMinStock
		p.DiscountType = v.ProdDiscountType
		p.DiscountValue = r.numericToFloat64(v.ProdDiscountValue)
		p.CurrentStock = v.ProdCurrentStock
		p.AvgCost = r.numericToFloat64(v.ProdAvgCost)
		p.IsTaxable = v.IsTaxable
		p.IsActive = v.IsActive
		p.CreatedAt = r.interfaceToTime(v.CreatedAt)
	case db.GetProductsCursorPrevRow:
		p.ID = v.ID
		p.Sku = v.Sku
		p.Barcode = r.interfaceToString(v.Barcode)
		p.Name = v.Name
		p.Description = r.interfaceToString(v.Description)
		p.CategoryID = v.CategoryID
		p.SupplierID = v.SupplierID
		p.CategoryName = r.interfaceToString(v.CategoryName)
		p.CategorySlug = r.interfaceToString(v.CategorySlug)
		p.SupplierName = r.interfaceToString(v.SupplierName)
		p.SupplierPic = r.interfaceToString(v.SupplierPic)
		p.SellingUnitID = v.SellingUnitID
		p.SellingUnitKode = r.interfaceToString(v.SellingUnitKode)
		p.SellingUnitName = r.interfaceToString(v.SellingUnitName)

		p.PurchaseUnitID = v.PurchaseUnitID
		p.PurchaseUnitKode = r.interfaceToString(v.PurchaseUnitKode)
		p.PurchaseUnitName = r.interfaceToString(v.PurchaseUnitName)
		p.PurchasePrice = r.numericToFloat64(v.ProdPurchasePrice)
		p.SellingPrice = r.numericToFloat64(v.ProdSellingPrice)
		p.MinStockLevel = v.ProdMinStock
		p.DiscountType = v.ProdDiscountType
		p.DiscountValue = r.numericToFloat64(v.ProdDiscountValue)
		p.CurrentStock = v.ProdCurrentStock
		p.AvgCost = r.numericToFloat64(v.ProdAvgCost)
		p.IsTaxable = v.IsTaxable
		p.IsActive = v.IsActive
		p.CreatedAt = r.interfaceToTime(v.CreatedAt)
	}
	return p
}

// defaultDiscountType menjaga nilai diskon produk tetap dalam daftar yang
// diperbolehkan database, sehingga produk lama tanpa diskon tetap aman.
func defaultDiscountType(t string) string {
	switch t {
	case "PERCENT", "AMOUNT":
		return t
	default:
		return "NONE"
	}
}

func (r *sqlProductRepository) float64ToNumeric(f float64) pgtype.Numeric {
	var num pgtype.Numeric
	err := num.Scan(fmt.Sprintf("%.2f", f))
	if err != nil {
		num.Valid = false
	}
	return num
}

func (r *sqlProductRepository) numericToFloat64(num pgtype.Numeric) float64 {
	if !num.Valid {
		return 0.0
	}
	if num.Exp == 0 {
		return float64(num.Int.Int64())
	}
	bigF := new(big.Float).SetInt(num.Int)
	if num.Exp < 0 {
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-num.Exp)), nil)
		bigF.Quo(bigF, new(big.Float).SetInt(divisor))
	} else {
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(num.Exp)), nil)
		bigF.Mul(bigF, new(big.Float).SetInt(multiplier))
	}
	f, _ := bigF.Float64()
	return f
}

func (r *sqlProductRepository) interfaceToString(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case sql.NullString:
		if v.Valid {
			return v.String
		}
	case pgtype.Text:
		if v.Valid {
			return v.String
		}
	}
	return ""
}

func (r *sqlProductRepository) interfaceToTime(val any) time.Time {
	if val == nil {
		return time.Time{}
	}
	switch v := val.(type) {
	case time.Time:
		return v
	case pgtype.Timestamp:
		if v.Valid {
			return v.Time
		}
	case pgtype.Timestamptz:
		if v.Valid {
			return v.Time
		}
	}
	return time.Time{}
}
