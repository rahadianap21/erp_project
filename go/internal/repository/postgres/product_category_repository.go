package postgres

import (
	"context"
	"database/sql"
	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlProductCategoryRepository struct {
	q *db.Queries
}

// NewSQLProductCategoryRepository menginisialisasi repository kategori produk
func NewSQLProductCategoryRepository(pool *pgxpool.Pool) domain.ProductCategoryRepository {
	return &sqlProductCategoryRepository{q: db.New(pool)}
}

func (r *sqlProductCategoryRepository) Create(ctx context.Context, cat *domain.ProductCategory) error {
	var parentIDPg pgtype.Int4
	if cat.ParentID != nil && *cat.ParentID > 0 {
		parentIDPg = pgtype.Int4{Int32: *cat.ParentID, Valid: true}
	} else {
		parentIDPg = pgtype.Int4{Valid: false}
	}

	var descPg pgtype.Text
	if cat.Description != "" {
		descPg = pgtype.Text{String: cat.Description, Valid: true}
	} else {
		descPg = pgtype.Text{Valid: false}
	}

	var createdByPg pgtype.Int8
	if cat.CreatedBy != nil && *cat.CreatedBy > 0 {
		createdByPg = pgtype.Int8{Int64: *cat.CreatedBy, Valid: true}
	} else {
		createdByPg = pgtype.Int8{Valid: false}
	}

	var updatedByPg pgtype.Int8
	if cat.UpdatedBy != nil && *cat.UpdatedBy > 0 {
		updatedByPg = pgtype.Int8{Int64: *cat.UpdatedBy, Valid: true}
	} else {
		updatedByPg = pgtype.Int8{Valid: false}
	}

	row, err := r.q.CreateProductCategory(ctx, db.CreateProductCategoryParams{
		Name:        cat.Name,
		Slug:        cat.Slug,
		Description: descPg,
		ParentID:    parentIDPg,
		CreatedBy:   createdByPg,
		UpdatedBy:   updatedByPg,
	})
	if err != nil {
		return err
	}

	cat.ID = row.ID
	cat.IsActive = row.IsActive
	cat.CreatedAt = r.interfaceToTime(row.CreatedAt)
	return nil
}

func (r *sqlProductCategoryRepository) GetCategoriesCursor(ctx context.Context, search string, cursorID int32, limit int, direction string) ([]domain.ProductCategory, int64, error) {
	categories := make([]domain.ProductCategory, 0)

	totalCount, err := r.q.CountProductCategories(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	if direction == "prev" {
		rows, err := r.q.GetProductCategoriesCursorPrev(ctx, db.GetProductCategoriesCursorPrevParams{
			SearchQuery: search,
			CursorID:    cursorID,
			ReqLimit:    int32(limit),
		})
		if err != nil {
			return nil, 0, err
		}
		reverseRows(rows)
		for _, row := range rows {
			categories = append(categories, r.mapRowToDomain(row))
		}
	} else {
		rows, err := r.q.GetProductCategoriesCursorNext(ctx, db.GetProductCategoriesCursorNextParams{
			SearchQuery: search,
			CursorID:    cursorID,
			ReqLimit:    int32(limit),
		})
		if err != nil {
			return nil, 0, err
		}
		for _, row := range rows {
			categories = append(categories, r.mapRowToDomain(row))
		}
	}

	return categories, totalCount, nil
}

func (r *sqlProductCategoryRepository) GetByID(ctx context.Context, id int32) (*domain.ProductCategory, error) {
	row, err := r.q.GetProductCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &domain.ProductCategory{
		ID:          row.ID,
		Name:        row.Name,
		Slug:        row.Slug,
		Description: r.interfaceToString(row.Description),
		ParentID:    r.interfaceToInt32Ptr(row.ParentID),
		ParentName:  r.interfaceToString(row.ParentName),
		IsActive:    row.IsActive,
		CreatedAt:   r.interfaceToTime(row.CreatedAt),
	}, nil
}

func (r *sqlProductCategoryRepository) Update(ctx context.Context, cat *domain.ProductCategory) error {
	var parentIDPg pgtype.Int4
	if cat.ParentID != nil && *cat.ParentID > 0 {
		parentIDPg = pgtype.Int4{Int32: *cat.ParentID, Valid: true}
	} else {
		parentIDPg = pgtype.Int4{Valid: false}
	}

	var descPg pgtype.Text
	if cat.Description != "" {
		descPg = pgtype.Text{String: cat.Description, Valid: true}
	} else {
		descPg = pgtype.Text{Valid: false}
	}

	var updatedByPg pgtype.Int8
	if cat.UpdatedBy != nil && *cat.UpdatedBy > 0 {
		updatedByPg = pgtype.Int8{Int64: *cat.UpdatedBy, Valid: true}
	} else {
		updatedByPg = pgtype.Int8{Valid: false}
	}

	return r.q.UpdateProductCategory(ctx, db.UpdateProductCategoryParams{
		ID:          cat.ID,
		Name:        cat.Name,
		Slug:        cat.Slug,
		Description: descPg,
		ParentID:    parentIDPg,
		IsActive:    cat.IsActive,
		UpdatedBy:   updatedByPg,
	})
}

func (r *sqlProductCategoryRepository) Delete(ctx context.Context, id int32, deletedBy int64) error {
	deletedByPg := pgtype.Int8{Int64: deletedBy, Valid: true}
	rows, err := r.q.DeleteProductCategory(ctx, db.DeleteProductCategoryParams{
		ID:        id,
		DeletedBy: deletedByPg,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		// 0 baris berarti ID tidak ada atau kategori sudah soft delete sebelumnya.
		// Tanpa pemeriksaan ini penghapusan yang gagal tetap terbalas sukses.
		return domain.ErrKategoriTidakDitemukan
	}
	return nil
}

// Helper utilitas internal polimorfisme pemetaan baris data
func (r *sqlProductCategoryRepository) mapRowToDomain(row any) domain.ProductCategory {
	c := domain.ProductCategory{}
	switch v := row.(type) {
	case db.GetProductCategoriesCursorNextRow:
		c.ID = v.ID
		c.Name = v.Name
		c.Slug = v.Slug
		c.Description = r.interfaceToString(v.Description)
		c.ParentID = r.interfaceToInt32Ptr(v.ParentID)
		c.ParentName = r.interfaceToString(v.ParentName)
		c.IsActive = v.IsActive
		c.CreatedAt = r.interfaceToTime(v.CreatedAt)
	case db.GetProductCategoriesCursorPrevRow:
		c.ID = v.ID
		c.Name = v.Name
		c.Slug = v.Slug
		c.Description = r.interfaceToString(v.Description)
		c.ParentID = r.interfaceToInt32Ptr(v.ParentID)
		c.ParentName = r.interfaceToString(v.ParentName)
		c.IsActive = v.IsActive
		c.CreatedAt = r.interfaceToTime(v.CreatedAt)
	}
	return c
}

func (r *sqlProductCategoryRepository) interfaceToString(val any) string {
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

func (r *sqlProductCategoryRepository) interfaceToTime(val any) time.Time {
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

func (r *sqlProductCategoryRepository) interfaceToInt32Ptr(val any) *int32 {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case int32:
		return &v
	case pgtype.Int4:
		if v.Valid {
			return &v.Int32
		}
	}
	return nil
}
