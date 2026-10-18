package postgres

import (
	"context"
	"errors"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlAccountingPeriodRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewSQLAccountingPeriodRepository menginisialisasi repository periode fiskal.
func NewSQLAccountingPeriodRepository(pool *pgxpool.Pool) domain.AccountingPeriodRepository {
	return &sqlAccountingPeriodRepository{pool: pool, q: db.New(pool)}
}

func mapPeriod(v db.GetAccountPeriodRow) domain.AccountingPeriod {
	return domain.AccountingPeriod{
		ID:          v.ID,
		PeriodYear:  int(v.PeriodYear),
		PeriodMonth: int(v.PeriodMonth),
		Status:      v.Status,
		ClosedAt:    convertPgTimeZ(v.ClosedAt),
		ClosedBy:    convertPgInt64(v.ClosedBy),
		CreatedAt:   v.CreatedAt.Time,
		UpdatedAt:   v.UpdatedAt.Time,
	}
}

func (r *sqlAccountingPeriodRepository) Get(ctx context.Context, year, month int) (*domain.AccountingPeriod, error) {
	row, err := r.q.GetAccountPeriod(ctx, db.GetAccountPeriodParams{
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
	})
	if err != nil {
		return nil, err
	}
	period := mapPeriod(row)
	return &period, nil
}

// EnsureOpen memastikan periode sasaran jurnal boleh dipakai:
//   - belum ada → periode baru dibuat (OPEN);
//   - sudah CLOSED → ditolak agar catatan periode tertutup tidak berubah.
//
// Membuat periode baru secara implisit tetap diizinkan karena UX lebih mulus
// daripada mewajibkan admin membuka periode manual sebelum jurnal pertama.
func (r *sqlAccountingPeriodRepository) EnsureOpen(ctx context.Context, year, month int, actorID int64) (*domain.AccountingPeriod, error) {
	row, err := r.q.GetAccountPeriod(ctx, db.GetAccountPeriodParams{
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		if row.Status != domain.PeriodStatusOpen {
			return nil, domain.ErrJournalPeriodClosed
		}
		period := mapPeriod(row)
		return &period, nil
	}

	created, err := r.q.CreateAccountPeriod(ctx, db.CreateAccountPeriodParams{
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
		CreatedBy:   pgtype.Int8{Int64: actorID, Valid: actorID > 0},
		UpdatedBy:   pgtype.Int8{Int64: actorID, Valid: actorID > 0},
	})
	if err != nil {
		return nil, err
	}
	if created.Status != domain.PeriodStatusOpen {
		return nil, domain.ErrJournalPeriodClosed
	}
	period := mapPeriod(db.GetAccountPeriodRow(created))
	return &period, nil
}

// Close menutup periode dengan compare-and-swap: hanya status OPEN yang berubah.
// Mengembalikan jumlah baris terpengaruh (0 = sudah CLOSED / tidak ada).
func (r *sqlAccountingPeriodRepository) Close(ctx context.Context, year, month, actorID int64) (int64, error) {
	return r.q.CloseAccountPeriod(ctx, db.CloseAccountPeriodParams{
		ActorUserID: actorID,
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
	})
}

// Reopen membuka kembali periode CLOSED. Mengembalikan jumlah baris terpengaruh.
func (r *sqlAccountingPeriodRepository) Reopen(ctx context.Context, year, month int) (int64, error) {
	return r.q.ReopenAccountPeriod(ctx, db.ReopenAccountPeriodParams{
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
	})
}

func (r *sqlAccountingPeriodRepository) List(ctx context.Context, limit int) ([]domain.AccountingPeriod, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := r.q.ListAccountPeriods(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	results := make([]domain.AccountingPeriod, 0, len(rows))
	for _, v := range rows {
		results = append(results, mapPeriod(db.GetAccountPeriodRow(v)))
	}
	return results, nil
}

// --- Journal ---

// sqlJournalRepository mengelola jurnal umum.
