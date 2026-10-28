package domain

import (
	"context"
	"errors"
	"time"
)

// Kunci pengaturan aplikasi umum (tabel s_app_settings).
const (
	SettingKeyPPNRate      = "PPN_RATE"
	SettingKeyPPNPriceMode = "PPN_PRICE_MODE"
)

// Nilai bawaan bila pengaturan belum ada di database.
const DefaultPPNPriceMode = TaxModeInclusive

// DefaultPPNRate adalah tarif PPN bawaan (12%).
var DefaultPPNRate = MustDec("0.12")

var (
	ErrSettingPPNRateInvalid = errors.New("tarif PPN harus antara 0 dan 1 (mis. 0.12 untuk 12%)")
	ErrSettingPPNModeInvalid = errors.New("mode harga PPN harus INCLUSIVE atau EXCLUSIVE")
)

// AppSetting adalah satu baris pengaturan key-value.
type AppSetting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   *int64    `json:"updated_by,omitempty"`
}

// SettingsView adalah tampilan pengaturan yang dipakai frontend.
type SettingsView struct {
	PPNRate      Decimal `json:"ppn_rate"`
	PPNPriceMode string  `json:"ppn_price_mode"`
}

// UpdateSettingsInput memakai pointer supaya field yang tidak dikirim tidak
// ikut diubah.
type UpdateSettingsInput struct {
	PPNRate      *Decimal `json:"ppn_rate"`
	PPNPriceMode *string  `json:"ppn_price_mode"`
}

type SettingsRepository interface {
	List(ctx context.Context) ([]AppSetting, error)
	Get(ctx context.Context, key string) (*AppSetting, error)
	Upsert(ctx context.Context, key, value, description string, actorID int64) error
}

// TaxConfigProvider menyediakan konfigurasi pajak penjualan. Implementasinya
// adalah SettingsUsecase; dipisah agar usecase penjualan tidak bergantung pada
// seluruh kontrak pengaturan.
type TaxConfigProvider interface {
	TaxConfig(ctx context.Context) (TaxConfig, error)
}

type SettingsUsecase interface {
	TaxConfigProvider
	GetAll(ctx context.Context) (*SettingsView, error)
	Update(ctx context.Context, in UpdateSettingsInput, actorID int64) (*SettingsView, error)
}
