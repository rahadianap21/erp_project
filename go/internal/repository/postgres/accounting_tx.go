package postgres

import (
	"context"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// sqlAccountingTx membungkus kueri yang terikat pada satu transaksi database.
// Hanya operasi yang dibutuhkan alur tutup/buka buku yang diekspos.
type sqlAccountingTx struct {
	q *db.Queries
}

func (r *sqlAccountingPeriodRepository) WithTx(ctx context.Context, fn func(tx domain.AccountingTx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(&sqlAccountingTx{q: r.q.WithTx(tx)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (tx *sqlAccountingTx) LockPeriod(ctx context.Context, year, month int) (*domain.AccountingPeriod, error) {
	row, err := tx.q.GetAccountPeriodForUpdate(ctx, db.GetAccountPeriodForUpdateParams{
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
	})
	if err != nil {
		return nil, err
	}
	return &domain.AccountingPeriod{
		ID:          row.ID,
		PeriodYear:  int(row.PeriodYear),
		PeriodMonth: int(row.PeriodMonth),
		Status:      row.Status,
		ClosedAt:    convertPgTimeZ(row.ClosedAt),
		ClosedBy:    convertPgInt64(row.ClosedBy),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func (tx *sqlAccountingTx) CreatePeriod(ctx context.Context, year, month int, actorID int64) (*domain.AccountingPeriod, error) {
	created, err := tx.q.CreateAccountPeriod(ctx, db.CreateAccountPeriodParams{
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
		CreatedBy:   pgtype.Int8{Int64: actorID, Valid: actorID > 0},
		UpdatedBy:   pgtype.Int8{Int64: actorID, Valid: actorID > 0},
	})
	if err != nil {
		return nil, err
	}
	period := mapPeriod(db.GetAccountPeriodRow(created))
	return &period, nil
}

func (tx *sqlAccountingTx) AccountBalances(ctx context.Context, from, to time.Time) ([]domain.AccountBalance, error) {
	rows, err := tx.q.GetPostedAccountBalances(ctx, db.GetPostedAccountBalancesParams{
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
			TotalDebit:    numericToFloat(row.TotalDebit),
			TotalCredit:   numericToFloat(row.TotalCredit),
		})
	}
	return balances, nil
}

func (tx *sqlAccountingTx) GetAccountsByIDs(ctx context.Context, ids []int64) ([]domain.ChartOfAccount, error) {
	rows, err := tx.q.GetChartOfAccountsForPosting(ctx, ids)
	if err != nil {
		return nil, err
	}
	accounts := make([]domain.ChartOfAccount, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, domain.ChartOfAccount{
			ID:            row.ID,
			Code:          row.Code,
			ParentID:      convertPgInt64(row.ParentID),
			IsHeader:      row.IsHeader,
			IsActive:      row.IsActive,
			AccountType:   row.AccountType,
			NormalBalance: row.NormalBalance,
		})
	}
	return accounts, nil
}

func (tx *sqlAccountingTx) ListAccountingSettings(ctx context.Context) (map[string]int64, error) {
	rows, err := tx.q.ListAccountingSettings(ctx)
	if err != nil {
		return nil, err
	}
	mapping := make(map[string]int64, len(rows))
	for _, row := range rows {
		mapping[row.MappingKey] = row.AccountID
	}
	return mapping, nil
}

func (tx *sqlAccountingTx) NextJournalSeq(ctx context.Context, prefix string) (int64, error) {
	return tx.q.NextJournalSeq(ctx, prefix)
}

func (tx *sqlAccountingTx) GetPostedJournalIDBySource(ctx context.Context, sourceType string, sourceID int64) (int64, error) {
	return tx.q.GetPostedJournalIDBySource(ctx, db.GetPostedJournalIDBySourceParams{
		SourceType: sourceType,
		SourceID:   sourceID,
	})
}

func (tx *sqlAccountingTx) CreatePostedJournal(ctx context.Context, entry *domain.JournalEntry) error {
	return createPosted(ctx, tx.q, entry)
}

func (tx *sqlAccountingTx) ReverseJournal(ctx context.Context, originalID int64, entryDate time.Time, reason string, actorID int64) (*domain.JournalEntry, error) {
	return reverseJournal(ctx, tx.q, originalID, entryDate, reason, actorID)
}

func (tx *sqlAccountingTx) ClosePeriod(ctx context.Context, year, month, actorID int64) (int64, error) {
	return tx.q.CloseAccountPeriod(ctx, db.CloseAccountPeriodParams{
		ActorUserID: actorID,
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
	})
}

func (tx *sqlAccountingTx) ReopenPeriod(ctx context.Context, year, month int) (int64, error) {
	return tx.q.ReopenAccountPeriod(ctx, db.ReopenAccountPeriodParams{
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
	})
}
