package postgres

import (
	"context"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlChartOfAccountRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewSQLChartOfAccountRepository menginisialisasi repository bagan akun.
func NewSQLChartOfAccountRepository(pool *pgxpool.Pool) domain.ChartOfAccountRepository {
	return &sqlChartOfAccountRepository{pool: pool, q: db.New(pool)}
}

func mapCOA(v db.GetChartOfAccountsCursorNextRow) domain.ChartOfAccount {
	return domain.ChartOfAccount{
		ID:            v.ID,
		Code:          v.Code,
		Name:          v.Name,
		AccountType:   v.AccountType,
		NormalBalance: v.NormalBalance,
		ParentID:      convertPgInt64(v.ParentID),
		ParentCode:    v.ParentCode.String,
		ParentName:    v.ParentName.String,
		IsHeader:      v.IsHeader,
		IsActive:      v.IsActive,
		CreatedAt:     v.CreatedAt.Time,
		UpdatedAt:     v.UpdatedAt.Time,
		DeletedAt:     convertPgTimeZ(v.DeletedAt),
	}
}

func (r *sqlChartOfAccountRepository) Create(ctx context.Context, acc *domain.ChartOfAccount) error {
	row, err := r.q.CreateChartOfAccount(ctx, db.CreateChartOfAccountParams{
		Code:          acc.Code,
		Name:          acc.Name,
		AccountType:   acc.AccountType,
		NormalBalance: acc.NormalBalance,
		ParentID:      nullableInt8(acc.ParentID),
		IsHeader:      acc.IsHeader,
		CreatedBy:     nullableInt8(acc.CreatedBy),
		UpdatedBy:     nullableInt8(acc.UpdatedBy),
	})
	if err != nil {
		if _, ok := uniqueViolationConstraint(err); ok {
			return domain.ErrAccountDuplicate
		}
		return err
	}
	acc.ID = row.ID
	acc.IsActive = row.IsActive
	acc.CreatedAt = row.CreatedAt.Time
	acc.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *sqlChartOfAccountRepository) GetListCursor(ctx context.Context, param domain.ChartOfAccountParam) ([]domain.ChartOfAccount, int64, error) {
	var rows []db.GetChartOfAccountsCursorNextRow
	var err error

	if param.Direction == "prev" {
		prevRows, err := r.q.GetChartOfAccountsCursorPrev(ctx, db.GetChartOfAccountsCursorPrevParams{
			SearchQuery: param.Search,
			CursorID:    param.CursorID,
			ReqLimit:    param.Limit,
		})
		if err != nil {
			return nil, 0, err
		}
		for _, v := range prevRows {
			rows = append(rows, db.GetChartOfAccountsCursorNextRow(v))
		}
		reverseRows(rows)
	} else {
		rows, err = r.q.GetChartOfAccountsCursorNext(ctx, db.GetChartOfAccountsCursorNextParams{
			SearchQuery: param.Search,
			CursorID:    param.CursorID,
			ReqLimit:    param.Limit,
		})
		if err != nil {
			return nil, 0, err
		}
	}

	count, err := r.q.CountChartOfAccounts(ctx, param.Search)
	if err != nil {
		return nil, 0, err
	}

	results := make([]domain.ChartOfAccount, 0, len(rows))
	for _, v := range rows {
		results = append(results, mapCOA(v))
	}
	return results, count, nil
}

func (r *sqlChartOfAccountRepository) GetByID(ctx context.Context, id int64) (*domain.ChartOfAccount, error) {
	row, err := r.q.GetChartOfAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	acc := mapCOA(db.GetChartOfAccountsCursorNextRow(row))
	return &acc, nil
}

func (r *sqlChartOfAccountRepository) GetByIDs(ctx context.Context, ids []int64) ([]domain.ChartOfAccount, error) {
	if len(ids) == 0 {
		return []domain.ChartOfAccount{}, nil
	}
	rows, err := r.q.GetChartOfAccountsForPosting(ctx, ids)
	if err != nil {
		return nil, err
	}
	results := make([]domain.ChartOfAccount, 0, len(rows))
	for _, v := range rows {
		results = append(results, domain.ChartOfAccount{
			ID:            v.ID,
			Code:          v.Code,
			ParentID:      convertPgInt64(v.ParentID),
			IsHeader:      v.IsHeader,
			IsActive:      v.IsActive,
			AccountType:   v.AccountType,
			NormalBalance: v.NormalBalance,
		})
	}
	return results, nil
}

func (r *sqlChartOfAccountRepository) Update(ctx context.Context, acc *domain.ChartOfAccount) error {
	err := r.q.UpdateChartOfAccount(ctx, db.UpdateChartOfAccountParams{
		Code:          acc.Code,
		Name:          acc.Name,
		AccountType:   acc.AccountType,
		NormalBalance: acc.NormalBalance,
		ParentID:      nullableInt8(acc.ParentID),
		IsHeader:      acc.IsHeader,
		IsActive:      acc.IsActive,
		UpdatedBy:     nullableInt8(acc.UpdatedBy),
		ID:            acc.ID,
	})
	if err != nil {
		if _, ok := uniqueViolationConstraint(err); ok {
			return domain.ErrAccountDuplicate
		}
		return err
	}
	return nil
}

func (r *sqlChartOfAccountRepository) Delete(ctx context.Context, id int64, deletedBy int64) error {
	return r.q.DeleteChartOfAccount(ctx, db.DeleteChartOfAccountParams{
		DeletedBy: pgtype.Int8{Int64: deletedBy, Valid: deletedBy > 0},
		ID:        id,
	})
}

// --- Accounting Period ---

// sqlAccountingPeriodRepository mengelola periode fiskal.
