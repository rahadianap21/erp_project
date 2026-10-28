package usecase

import (
	"context"
	"strings"

	"go_backend/internal/domain"
)

type accountingSettingsUsecase struct {
	repo    domain.AccountingSettingsAdminRepository
	coaRepo domain.ChartOfAccountRepository
}

// NewAccountingSettingsUsecase membuat usecase pengaturan akun otomatis.
func NewAccountingSettingsUsecase(repo domain.AccountingSettingsAdminRepository, coaRepo domain.ChartOfAccountRepository) domain.AccountingSettingsUsecase {
	return &accountingSettingsUsecase{repo: repo, coaRepo: coaRepo}
}

// allowedAccountingSettingKeys adalah kunci yang dikenal auto-posting. Kunci di
// luar daftar ini ditolak supaya tabel tidak terisi nilai yang tidak pernah
// dibaca oleh jurnal otomatis.
var allowedAccountingSettingKeys = map[string]bool{
	"INVENTORY":         true,
	"AP":                true,
	"AR":                true,
	"COGS":              true,
	"SALES_REVENUE":     true,
	"VAT_INPUT":         true,
	"VAT_OUTPUT":        true,
	"CASH":              true,
	"RETAINED_EARNINGS": true,
}

func (u *accountingSettingsUsecase) ListSettings(ctx context.Context) ([]domain.AccountingSettingDetail, error) {
	return u.repo.ListDetails(ctx)
}

// UpdateSettings memvalidasi seluruh kunci dan akun terlebih dahulu, lalu
// menyimpannya sekaligus. Akun header dan akun nonaktif ditolak karena tidak
// bisa dipakai posting, sehingga jurnal otomatis tidak gagal di kemudian hari.
func (u *accountingSettingsUsecase) UpdateSettings(ctx context.Context, in []domain.AccountingSettingUpdateInput, actorID int64) ([]domain.AccountingSettingDetail, error) {
	if actorID <= 0 {
		return nil, domain.NewBusinessRuleError("tidak terautentikasi")
	}
	if len(in) == 0 {
		return nil, domain.NewBusinessRuleError("tidak ada pengaturan yang dikirim")
	}

	settings := make([]domain.AccountingSetting, 0, len(in))
	accountIDs := make([]int64, 0, len(in))
	seen := make(map[string]bool, len(in))

	for _, item := range in {
		key := strings.ToUpper(strings.TrimSpace(item.MappingKey))
		if !allowedAccountingSettingKeys[key] {
			return nil, domain.NewBusinessRuleError("kunci pemetaan %q tidak dikenal", item.MappingKey)
		}
		if seen[key] {
			return nil, domain.NewBusinessRuleError("kunci pemetaan %s dikirim lebih dari sekali", key)
		}
		if item.AccountID <= 0 {
			return nil, domain.NewBusinessRuleError("akun untuk %s belum dipilih", key)
		}
		seen[key] = true
		accountIDs = append(accountIDs, item.AccountID)
		settings = append(settings, domain.AccountingSetting{
			MappingKey: key,
			AccountID:  item.AccountID,
		})
	}

	accounts, err := u.coaRepo.GetByIDs(ctx, accountIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]domain.ChartOfAccount, len(accounts))
	for _, acc := range accounts {
		byID[acc.ID] = acc
	}

	for _, setting := range settings {
		acc, ok := byID[setting.AccountID]
		if !ok {
			return nil, domain.NewBusinessRuleError("akun id %d tidak ditemukan", setting.AccountID)
		}
		if acc.IsHeader {
			return nil, domain.NewBusinessRuleError("akun %s adalah akun header dan tidak bisa dipakai posting", acc.Code)
		}
		if !acc.IsActive {
			return nil, domain.NewBusinessRuleError("akun %s sudah nonaktif", acc.Code)
		}
	}

	if err := u.repo.UpsertMany(ctx, settings, actorID); err != nil {
		return nil, err
	}
	return u.repo.ListDetails(ctx)
}
