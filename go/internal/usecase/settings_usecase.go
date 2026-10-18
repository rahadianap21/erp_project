package usecase

import (
	"context"
	"strconv"
	"strings"

	"go_backend/internal/domain"
)

type settingsUsecase struct {
	repo domain.SettingsRepository
}

func NewSettingsUsecase(repo domain.SettingsRepository) domain.SettingsUsecase {
	return &settingsUsecase{repo: repo}
}

const (
	ppnRateDescription = "Tarif PPN dalam fraksi (0.12 = 12%)"
	ppnModeDescription = "Mode harga jual: INCLUSIVE (harga sudah termasuk PPN) / EXCLUSIVE (PPN ditambahkan ke harga)"
)

// TaxConfig membaca tarif dan mode PPN dari pengaturan, memakai nilai bawaan
// bila baris pengaturan belum ada atau isinya tidak valid.
func (u *settingsUsecase) TaxConfig(ctx context.Context) (domain.TaxConfig, error) {
	cfg := domain.TaxConfig{Rate: domain.DefaultPPNRate, Mode: domain.DefaultPPNPriceMode}

	rate, err := u.repo.Get(ctx, domain.SettingKeyPPNRate)
	if err != nil {
		return cfg, err
	}
	if rate != nil {
		if v, perr := strconv.ParseFloat(strings.TrimSpace(rate.Value), 64); perr == nil && v >= 0 {
			cfg.Rate = v
		}
	}

	mode, err := u.repo.Get(ctx, domain.SettingKeyPPNPriceMode)
	if err != nil {
		return cfg, err
	}
	if mode != nil {
		switch strings.ToUpper(strings.TrimSpace(mode.Value)) {
		case domain.TaxModeExclusive:
			cfg.Mode = domain.TaxModeExclusive
		case domain.TaxModeInclusive:
			cfg.Mode = domain.TaxModeInclusive
		}
	}
	return cfg, nil
}

func (u *settingsUsecase) GetAll(ctx context.Context) (*domain.SettingsView, error) {
	cfg, err := u.TaxConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &domain.SettingsView{PPNRate: cfg.Rate, PPNPriceMode: cfg.Mode}, nil
}

func (u *settingsUsecase) Update(ctx context.Context, in domain.UpdateSettingsInput, actorID int64) (*domain.SettingsView, error) {
	if in.PPNRate != nil {
		rate := *in.PPNRate
		if rate < 0 || rate > 1 {
			return nil, domain.ErrSettingPPNRateInvalid
		}
		value := strconv.FormatFloat(rate, 'f', -1, 64)
		if err := u.repo.Upsert(ctx, domain.SettingKeyPPNRate, value, ppnRateDescription, actorID); err != nil {
			return nil, err
		}
	}
	if in.PPNPriceMode != nil {
		mode := strings.ToUpper(strings.TrimSpace(*in.PPNPriceMode))
		if mode != domain.TaxModeInclusive && mode != domain.TaxModeExclusive {
			return nil, domain.ErrSettingPPNModeInvalid
		}
		if err := u.repo.Upsert(ctx, domain.SettingKeyPPNPriceMode, mode, ppnModeDescription, actorID); err != nil {
			return nil, err
		}
	}
	return u.GetAll(ctx)
}
