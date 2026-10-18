package usecase

import (
	"context"
	"errors"
	"strings"

	"go_backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

// notFoundAs menerjemahkan "data tidak ditemukan" PostgreSQL menjadi error
// domain yang spesifik agar handler bisa memilih status HTTP yang tepat.
func notFoundAs(err error, fallback error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fallback
	}
	return err
}

type chartOfAccountUsecase struct {
	repo domain.ChartOfAccountRepository
}

func NewChartOfAccountUsecase(repo domain.ChartOfAccountRepository) domain.ChartOfAccountUsecase {
	return &chartOfAccountUsecase{repo: repo}
}

// balanceForType mengembalikan saldo normal yang wajib untuk suatu tipe akun.
func balanceForType(accountType string) (string, bool) {
	switch accountType {
	case domain.AccountTypeAsset, domain.AccountTypeExpense:
		return domain.NormalBalanceDebit, true
	case domain.AccountTypeLiability, domain.AccountTypeEquity, domain.AccountTypeRevenue:
		return domain.NormalBalanceCredit, true
	}
	return "", false
}

// validateAccount memvalidasi payload akun dan menormalkan nilainya. Dipakai
// bersama oleh pembuatan dan koreksi akun.
func validateAccount(acc *domain.ChartOfAccount) error {
	acc.Code = strings.TrimSpace(strings.ToUpper(acc.Code))
	acc.Name = strings.TrimSpace(acc.Name)
	acc.AccountType = strings.ToUpper(strings.TrimSpace(acc.AccountType))
	acc.NormalBalance = strings.ToUpper(strings.TrimSpace(acc.NormalBalance))

	if acc.Code == "" {
		return domain.ErrKodeAkunWajib
	}
	if len(acc.Code) > 20 {
		return domain.NewBusinessRuleError("kode akun maksimal 20 karakter")
	}
	if acc.Name == "" {
		return domain.ErrNamaAkunWajib
	}
	if len(acc.Name) > 150 {
		return domain.NewBusinessRuleError("nama akun maksimal 150 karakter")
	}
	expected, ok := balanceForType(acc.AccountType)
	if !ok {
		return domain.ErrTipeAkunInvalid
	}
	if acc.NormalBalance == "" {
		acc.NormalBalance = expected
	}
	if acc.NormalBalance != expected {
		return domain.ErrSaldoNormalMismatch
	}
	return nil
}

func (u *chartOfAccountUsecase) CreateAccount(ctx context.Context, acc *domain.ChartOfAccount, userID int64) error {
	if err := validateAccount(acc); err != nil {
		return err
	}
	// Hierarki: induk wajib ada. Pengecekan eksplisit memberikan pesan yang
	// lebih jelas daripada error foreign key mentah dari database.
	if acc.ParentID != nil && *acc.ParentID > 0 {
		parent, err := u.repo.GetByID(ctx, *acc.ParentID)
		if err != nil {
			return notFoundAs(err, domain.ErrIndukAkunNotFound)
		}
		if parent.IsHeader == false {
			return domain.NewBusinessRuleError("akun induk harus akun bertipe header")
		}
	} else {
		acc.ParentID = nil
	}

	acc.CreatedBy = &userID
	acc.UpdatedBy = &userID
	return u.repo.Create(ctx, acc)
}

func (u *chartOfAccountUsecase) FindAllAccounts(ctx context.Context, param domain.ChartOfAccountParam) ([]domain.ChartOfAccount, int64, error) {
	if param.Limit <= 0 {
		param.Limit = 20
	}
	return u.repo.GetListCursor(ctx, param)
}

func (u *chartOfAccountUsecase) FindAccountByID(ctx context.Context, id int64) (*domain.ChartOfAccount, error) {
	acc, err := u.repo.GetByID(ctx, id)
	return acc, notFoundAs(err, domain.ErrAccountNotFound)
}

func (u *chartOfAccountUsecase) UpdateAccount(ctx context.Context, acc *domain.ChartOfAccount, userID int64) error {
	if err := validateAccount(acc); err != nil {
		return err
	}
	if acc.ParentID != nil && *acc.ParentID > 0 {
		parent, err := u.repo.GetByID(ctx, *acc.ParentID)
		if err != nil {
			return notFoundAs(err, domain.ErrIndukAkunNotFound)
		}
		if !parent.IsHeader {
			return domain.NewBusinessRuleError("akun induk harus akun bertipe header")
		}
	} else {
		acc.ParentID = nil
	}

	existing, err := u.repo.GetByID(ctx, acc.ID)
	if err != nil {
		return notFoundAs(err, domain.ErrAccountNotFound)
	}
	_ = existing

	acc.UpdatedBy = &userID
	return u.repo.Update(ctx, acc)
}

func (u *chartOfAccountUsecase) DeleteAccount(ctx context.Context, id int64, userID int64) error {
	if _, err := u.repo.GetByID(ctx, id); err != nil {
		return notFoundAs(err, domain.ErrAccountNotFound)
	}
	return u.repo.Delete(ctx, id, userID)
}
