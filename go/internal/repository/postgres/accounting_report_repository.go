package postgres

import (
	"context"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// sqlAccountingReportRepository menyediakan agregat jurnal POSTED untuk
// laporan keuangan. Seluruh kueri membatasi diri pada status POSTED sehingga
// jurnal DRAFT atau VOID tidak pernah mengubah angka laporan.
type sqlAccountingReportRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewSQLAccountingReportRepository menginisialisasi repository laporan keuangan.
func NewSQLAccountingReportRepository(pool *pgxpool.Pool) domain.AccountingReportRepository {
	return &sqlAccountingReportRepository{pool: pool, q: db.New(pool)}
}

func (r *sqlAccountingReportRepository) AccountBalances(ctx context.Context, from, to time.Time) ([]domain.AccountBalance, error) {
	rows, err := r.q.GetPostedAccountBalances(ctx, db.GetPostedAccountBalancesParams{
		FromDate: pgtype.Date{Time: from, Valid: true},
		ToDate:   pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	balances := make([]domain.AccountBalance, 0, len(rows))
	for _, row := range rows {
		balances = append(balances, domain.AccountBalance{
			AccountID:     row.AccountID,
			Code:          row.Code,
			Name:          row.Name,
			AccountType:   row.AccountType,
			NormalBalance: row.NormalBalance,
			TotalDebit:    moneyFromNumeric(row.TotalDebit),
			TotalCredit:   moneyFromNumeric(row.TotalCredit),
		})
	}
	return balances, nil
}

func (r *sqlAccountingReportRepository) AccountLedgerLines(ctx context.Context, accountID int64, from, to time.Time) ([]domain.AccountLedgerLine, error) {
	rows, err := r.q.GetAccountLedgerLines(ctx, db.GetAccountLedgerLinesParams{
		AccountID: accountID,
		FromDate:  pgtype.Date{Time: from, Valid: true},
		ToDate:    pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	lines := make([]domain.AccountLedgerLine, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, domain.AccountLedgerLine{
			EntryNumber: row.EntryNumber,
			EntryDate:   row.EntryDate.Time,
			Description: row.Description,
			Debit:       moneyFromNumeric(row.Debit),
			Credit:      moneyFromNumeric(row.Credit),
		})
	}
	return lines, nil
}

func (r *sqlAccountingReportRepository) AccountOpeningBalance(ctx context.Context, accountID int64, before time.Time) (domain.Money, domain.Money, error) {
	row, err := r.q.GetAccountOpeningBalance(ctx, db.GetAccountOpeningBalanceParams{
		AccountID:  accountID,
		BeforeDate: pgtype.Date{Time: before, Valid: true},
	})
	if err != nil {
		return domain.Money{}, domain.Money{}, err
	}
	return moneyFromNumeric(row.TotalDebit), moneyFromNumeric(row.TotalCredit), nil
}
