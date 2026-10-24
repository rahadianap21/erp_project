package usecase

import (
	"context"
	"errors"

	"go_backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

type accountingPeriodUsecase struct {
	repo domain.AccountingPeriodRepository
	// closing mengurus jurnal penutup saat periode ditutup/dibuka kembali. Boleh
	// nil; bila nil, penutupan periode hanya mengunci status (tanpa jurnal).
	closing periodClosingManager
}

// periodClosingManager adalah irisan perilaku domain.JournalUsecase yang dipakai
// periode fiskal. Didefinisikan sempit supaya periode tidak bergantung pada
// seluruh permukaan jurnal umum.
type periodClosingManager interface {
	PostClosingEntryTx(ctx context.Context, tx domain.AccountingTx, period *domain.AccountingPeriod, actorID int64) error
	ReverseClosingEntryTx(ctx context.Context, tx domain.AccountingTx, period *domain.AccountingPeriod, actorID int64) error
}

func NewAccountingPeriodUsecase(repo domain.AccountingPeriodRepository, closing periodClosingManager) domain.AccountingPeriodUsecase {
	return &accountingPeriodUsecase{repo: repo, closing: closing}
}

func validatePeriode(year, month int) error {
	if year < 2000 || year > 2100 || month < 1 || month > 12 {
		return domain.ErrPeriodeInvalid
	}
	return nil
}

func (u *accountingPeriodUsecase) GetPeriod(ctx context.Context, year, month int64) (*domain.AccountingPeriod, error) {
	if err := validatePeriode(int(year), int(month)); err != nil {
		return nil, err
	}
	period, err := u.repo.Get(ctx, int(year), int(month))
	return period, notFoundAs(err, domain.ErrPeriodNotFound)
}

func (u *accountingPeriodUsecase) OpenPeriod(ctx context.Context, year, month, userID int64) (*domain.AccountingPeriod, error) {
	y, m := int(year), int(month)
	if err := validatePeriode(y, m); err != nil {
		return nil, err
	}

	var result *domain.AccountingPeriod
	err := u.repo.WithTx(ctx, func(tx domain.AccountingTx) error {
		period, err := tx.LockPeriod(ctx, y, m)
		if errors.Is(err, pgx.ErrNoRows) {
			created, err := tx.CreatePeriod(ctx, y, m, userID)
			if err != nil {
				return err
			}
			if created.Status != domain.PeriodStatusOpen {
				if err := u.reopenPeriodTx(ctx, tx, created, userID); err != nil {
					return err
				}
				created.Status = domain.PeriodStatusOpen
				created.ClosedAt = nil
				created.ClosedBy = nil
			}
			result = created
			return nil
		}
		if err != nil {
			return err
		}
		if err := u.reopenPeriodTx(ctx, tx, period, userID); err != nil {
			return err
		}
		period.Status = domain.PeriodStatusOpen
		period.ClosedAt = nil
		period.ClosedBy = nil
		result = period
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (u *accountingPeriodUsecase) ClosePeriod(ctx context.Context, year, month, userID int64) error {
	y, m := int(year), int(month)
	if err := validatePeriode(y, m); err != nil {
		return err
	}
	return u.repo.WithTx(ctx, func(tx domain.AccountingTx) error {
		period, err := tx.LockPeriod(ctx, y, m)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrPeriodNotFound
		}
		if err != nil {
			return err
		}
		if period.Status == domain.PeriodStatusClosed {
			return nil
		}
		if u.closing != nil {
			if err := u.closing.PostClosingEntryTx(ctx, tx, period, userID); err != nil {
				return err
			}
		}
		rows, err := tx.ClosePeriod(ctx, int64(y), int64(m), userID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return domain.ErrPeriodeBerubah
		}
		return nil
	})
}

func (u *accountingPeriodUsecase) ReopenPeriod(ctx context.Context, year, month int, userID int64) error {
	if err := validatePeriode(year, month); err != nil {
		return err
	}
	return u.repo.WithTx(ctx, func(tx domain.AccountingTx) error {
		period, err := tx.LockPeriod(ctx, year, month)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrPeriodNotFound
		}
		if err != nil {
			return err
		}
		return u.reopenPeriodTx(ctx, tx, period, userID)
	})
}

// reopenPeriodTx membuka status periode lalu membalik jurnal penutupnya dalam
// transaksi yang sama. Urutan ini penting: jurnal pembalik hanya boleh tercatat
// setelah periode kembali OPEN.
func (u *accountingPeriodUsecase) reopenPeriodTx(ctx context.Context, tx domain.AccountingTx, period *domain.AccountingPeriod, userID int64) error {
	if period.Status != domain.PeriodStatusOpen {
		rows, err := tx.ReopenPeriod(ctx, period.PeriodYear, period.PeriodMonth)
		if err != nil {
			return err
		}
		if rows == 0 {
			return domain.ErrPeriodeBerubah
		}
	}
	if u.closing != nil {
		if err := u.closing.ReverseClosingEntryTx(ctx, tx, period, userID); err != nil {
			return err
		}
	}
	return nil
}

func (u *accountingPeriodUsecase) ListPeriods(ctx context.Context, limit int) ([]domain.AccountingPeriod, error) {
	return u.repo.List(ctx, limit)
}
