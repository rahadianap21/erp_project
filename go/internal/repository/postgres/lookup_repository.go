package postgres

import (
	"context"
	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlLookupRepository struct {
	q *db.Queries
}

func NewSQLLookupRepository(pool *pgxpool.Pool) domain.LookupRepository {
	return &sqlLookupRepository{q: db.New(pool)}
}

func (r *sqlLookupRepository) GetSuppliersForPO(ctx context.Context) ([]domain.SupplierLookup, error) {
	rows, err := r.q.LookupSuppliersForPO(ctx)
	if err != nil {
		return nil, err
	}

	// Slice kosong bukan nil agar JSON merespons [] dan bukan null
	res := make([]domain.SupplierLookup, 0, len(rows))
	for _, row := range rows {
		res = append(res, domain.SupplierLookup{
			ID:             row.ID,
			Name:           row.Name,
			DefaultTaxType: row.DefaultTaxType,
		})
	}
	return res, nil
}

func (r *sqlLookupRepository) GetProductsForPO(ctx context.Context, supplierID int64) ([]domain.ProductLookup, error) {
	rows, err := r.q.LookupProductsForPO(ctx, supplierID)
	if err != nil {
		return nil, err
	}

	res := make([]domain.ProductLookup, 0, len(rows))
	for _, row := range rows {
		res = append(res, domain.ProductLookup{
			ID:               row.ID,
			Sku:              row.Sku,
			Name:             row.Name,
			SupplierID:       row.SupplierID,
			PurchasePrice:    moneyFromNumeric(row.PurchasePrice),
			PurchaseUnitKode: r.interfaceToString(row.PurchaseUnitKode),
		})
	}
	return res, nil
}

func (r *sqlLookupRepository) interfaceToString(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case pgtype.Text:
		if v.Valid {
			return v.String
		}
	}
	return ""
}
