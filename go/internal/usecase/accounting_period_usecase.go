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
	PostClosingEntry(ctx context.Context, period *domain.AccountingPeriod, actorID int64) error
	ReverseClosingEntry(ctx context.Context, period *domain.AccountingPeriod, actorID int64) error
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

	period, err := u.repo.Get(ctx, y, m)
	if errors.Is(err, pgx.ErrNoRows) {
		created, err := u.repo.EnsureOpen(ctx, y, m, userID)
		if err != nil {
			return nil, err
		}
		return created, nil
	}
	if err != nil {
		return nil, err
	}
	if period.Status == domain.PeriodStatusOpen {
		return period, nil
	}
	rows, err := u.repo.Reopen(ctx, y, m)
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, domain.ErrPeriodeBerubah
	}
	return u.repo.Get(ctx, y, m)
}

func (u *accountingPeriodUsecase) ClosePeriod(ctx context.Context, year, month, userID int64) error {
	y, m := int(year), int(month)
	if err := validatePeriode(y, m); err != nil {
		return err
	}
	period, err := u.repo.Get(ctx, y, m)
	if err != nil {
		return notFoundAs(err, domain.ErrPeriodNotFound)
	}
	if period.Status == domain.PeriodStatusClosed {
		return nil
	}
	// Terbitkan jurnal penutup SEBELUM periode dikunci. Bila ini gagal, periode
	// tetap terbuka sehingga tidak pernah ada periode tertutup tanpa penutupan
	// buku. Jurnal penutup idempoten, jadi pengulangan penutupan aman.
	if u.closing != nil {
		if err := u.closing.PostClosingEntry(ctx, period, userID); err != nil {
			return err
		}
	}
	rows, err := u.repo.Close(ctx, int64(y), int64(m), userID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrPeriodeBerubah
	}
	return nil
}

func (u *accountingPeriodUsecase) ReopenPeriod(ctx context.Context, year, month int, userID int64) error {
	if err := validatePeriode(year, month); err != nil {
		return err
	}
	period, err := u.repo.Get(ctx, year, month)
	if err != nil {
		return notFoundAs(err, domain.ErrPeriodNotFound)
	}
	if period.Status != domain.PeriodStatusOpen {
		rows, err := u.repo.Reopen(ctx, year, month)
		if err != nil {
			return err
		}
		if rows == 0 {
			return domain.ErrPeriodeBerubah
		}
		period.Status = domain.PeriodStatusOpen
	}
	// Batalkan jurnal penutup periode ini (bila ada) supaya pembukuan kembali
	// seperti sebelum tutup buku. Selalu dicoba, termasuk saat periode memang
	// sudah OPEN, agar percobaan ulang setelah kegagalan parsial tetap beres.
	if u.closing != nil {
		if err := u.closing.ReverseClosingEntry(ctx, period, userID); err != nil {
			return err
		}
	}
	return nil
}

func (u *accountingPeriodUsecase) ListPeriods(ctx context.Context, limit int) ([]domain.AccountingPeriod, error) {
	return u.repo.List(ctx, limit)
}
