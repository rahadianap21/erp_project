package postgres

import (
	"context"
	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlDashboardRepository struct {
	q *db.Queries
}

func NewSQLDashboardRepository(pool *pgxpool.Pool) domain.DashboardRepository {
	return &sqlDashboardRepository{q: db.New(pool)}
}

func (r *sqlDashboardRepository) GetStats(ctx context.Context) (domain.DashboardStats, error) {
	row, err := r.q.GetDashboardStats(ctx)
	if err != nil {
		return domain.DashboardStats{}, err
	}
	return domain.DashboardStats{
		TotalPOValue:   row.TotalPoValue,
		TotalPO:        row.TotalPo,
		POPending:      row.PoPending,
		POApproved:     row.PoApproved,
		POReceived:     row.PoReceived,
		TotalGRN:       row.TotalGrn,
		TotalProducts:  row.TotalProducts,
		TotalSuppliers: row.TotalSuppliers,
		LowStockCount:  row.LowStockCount,
	}, nil
}

func (r *sqlDashboardRepository) GetLowStock(ctx context.Context, limit int) ([]domain.DashboardLowStockItem, error) {
	rows, err := r.q.GetDashboardLowStock(ctx, int32(limit))
	if err != nil {
		return nil, err
	}

	res := make([]domain.DashboardLowStockItem, 0, len(rows))
	for _, row := range rows {
		res = append(res, domain.DashboardLowStockItem{
			ID:              row.ID,
			SKU:             row.Sku,
			Name:            row.Name,
			CurrentStock:    row.CurrentStock,
			MinStockLevel:   row.MinStockLevel,
			SellingUnitKode: row.SellingUnitKode,
		})
	}
	return res, nil
}

func (r *sqlDashboardRepository) GetRecentPOs(ctx context.Context, limit int) ([]domain.DashboardRecentPO, error) {
	rows, err := r.q.GetDashboardRecentPOs(ctx, int32(limit))
	if err != nil {
		return nil, err
	}

	res := make([]domain.DashboardRecentPO, 0, len(rows))
	for _, row := range rows {
		res = append(res, domain.DashboardRecentPO{
			ID:           row.ID,
			PONumber:     row.PoNumber,
			SupplierName: row.SupplierName,
			TotalAmount:  row.TotalAmount,
			Status:       row.Status,
			PODate:       formatPgDate(row.PoDate),
		})
	}
	return res, nil
}

func (r *sqlDashboardRepository) GetRecentGRNs(ctx context.Context, limit int) ([]domain.DashboardRecentGRN, error) {
	rows, err := r.q.GetDashboardRecentGRNs(ctx, int32(limit))
	if err != nil {
		return nil, err
	}

	res := make([]domain.DashboardRecentGRN, 0, len(rows))
	for _, row := range rows {
		res = append(res, domain.DashboardRecentGRN{
			ID:           row.ID,
			GRNNumber:    row.GrnNumber,
			PONumber:     row.PoNumber,
			TotalQty:     row.TotalQty,
			ReceivedDate: formatPgDate(row.ReceivedDate),
		})
	}
	return res, nil
}

// formatPgDate menyeragamkan tanggal DATE PostgreSQL ke string ISO (YYYY-MM-DD)
// supaya frontend tinggal mem-parse tanpa ambiguitas zona waktu.
func formatPgDate(d pgtype.Date) string {
	if !d.Valid || d.Time.IsZero() {
		return ""
	}
	return d.Time.Format("2006-01-02")
}
