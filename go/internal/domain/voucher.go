package domain

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
)

// =========================================================================
// KONSTANTA VOUCHER
// =========================================================================

const (
	VoucherDiscountPercent = "PERCENT"
	VoucherDiscountAmount  = "AMOUNT"
)

// =========================================================================
// ERROR VOUCHER
// =========================================================================

var (
	ErrVoucherNotFound        = errors.New("voucher tidak ditemukan")
	ErrVoucherInactive        = errors.New("voucher tidak aktif")
	ErrVoucherUsed            = errors.New("voucher sudah dipakai")
	ErrVoucherNotStarted      = errors.New("voucher belum memasuki masa berlaku")
	ErrVoucherExpired         = errors.New("voucher sudah kedaluwarsa")
	ErrVoucherMinPurchase     = errors.New("minimal belanja voucher belum terpenuhi")
	ErrVoucherDiscountInvalid = errors.New("diskon voucher tidak valid")
)

// =========================================================================
// MASTER VOUCHER
// =========================================================================

type Voucher struct {
	ID            int64      `json:"id"`
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	DiscountType  string     `json:"discount_type"`
	DiscountValue float64    `json:"discount_value"`
	MinPurchase   float64    `json:"min_purchase"`
	MaxDiscount   float64    `json:"max_discount"`
	ValidFrom     *DateOnly  `json:"valid_from,omitempty"`
	ValidUntil    *DateOnly  `json:"valid_until,omitempty"`
	IsActive      bool       `json:"is_active"`
	UsedAt        *time.Time `json:"used_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type VoucherInput struct {
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	DiscountType  string    `json:"discount_type"`
	DiscountValue float64   `json:"discount_value"`
	MinPurchase   float64   `json:"min_purchase"`
	MaxDiscount   float64   `json:"max_discount"`
	ValidFrom     *DateOnly `json:"valid_from"`
	ValidUntil    *DateOnly `json:"valid_until"`
	IsActive      bool      `json:"is_active"`
}

type VoucherRepository interface {
	Create(ctx context.Context, in VoucherInput, actorID int64) (*Voucher, error)
	Update(ctx context.Context, id int64, in VoucherInput, actorID int64) error
	Delete(ctx context.Context, id, actorID int64) error
	GetByID(ctx context.Context, id int64) (*Voucher, error)
	GetByCode(ctx context.Context, code string) (*Voucher, error)
	GetListCursor(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]Voucher, int64, error)
}

type VoucherUsecase interface {
	Create(ctx context.Context, in VoucherInput, actorID int64) (*Voucher, error)
	Update(ctx context.Context, id int64, in VoucherInput, actorID int64) error
	Delete(ctx context.Context, id, actorID int64) error
	GetByID(ctx context.Context, id int64) (*Voucher, error)
	GetByCode(ctx context.Context, code string) (*Voucher, error)
	GetAll(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]Voucher, int64, error)
}

// NormalizeVoucherCode menyamakan kode voucher agar isi barcode, pencarian,
// dan penyimpanan selalu konsisten.
func NormalizeVoucherCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// ValidateUsable memeriksa status voucher pada waktu tertentu. Pemeriksaan
// single-use (used_at) dan masa berlaku dilakukan di sini agar usecase dan
// repository memakai aturan yang sama.
func (v *Voucher) ValidateUsable(now time.Time) error {
	if v == nil {
		return ErrVoucherNotFound
	}
	if !v.IsActive {
		return ErrVoucherInactive
	}
	if v.UsedAt != nil {
		return ErrVoucherUsed
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if v.ValidFrom != nil {
		start := time.Date(v.ValidFrom.Year(), v.ValidFrom.Month(), v.ValidFrom.Day(), 0, 0, 0, 0, now.Location())
		if today.Before(start) {
			return ErrVoucherNotStarted
		}
	}
	if v.ValidUntil != nil {
		end := time.Date(v.ValidUntil.Year(), v.ValidUntil.Month(), v.ValidUntil.Day(), 0, 0, 0, 0, now.Location())
		if today.After(end) {
			return ErrVoucherExpired
		}
	}
	return nil
}

// ValidateSubtotal memastikan minimal belanja terpenuhi. Basisnya adalah
// subtotal sebelum diskon item agar syarat mudah dipahami kasir.
func (v *Voucher) ValidateSubtotal(subtotal float64) error {
	if v == nil {
		return ErrVoucherNotFound
	}
	if subtotal+1e-9 < v.MinPurchase {
		return ErrVoucherMinPurchase
	}
	return nil
}

// DiscountAmount menghitung potongan voucher dari nilai setelah diskon item.
// Diskon persen dibatasi max_discount (bila > 0); diskon nominal tidak pernah
// melebihi nilai belanja.
func (v *Voucher) DiscountAmount(base float64) (float64, error) {
	if v == nil {
		return 0, nil
	}
	if base < 0 {
		base = 0
	}
	switch v.DiscountType {
	case VoucherDiscountPercent:
		if v.DiscountValue < 0 || v.DiscountValue > 100 {
			return 0, ErrVoucherDiscountInvalid
		}
		amount := roundMoney(base * v.DiscountValue / 100)
		if v.MaxDiscount > 0 && amount > v.MaxDiscount {
			amount = roundMoney(v.MaxDiscount)
		}
		if amount > base {
			amount = roundMoney(base)
		}
		return amount, nil
	case VoucherDiscountAmount:
		if v.DiscountValue < 0 {
			return 0, ErrVoucherDiscountInvalid
		}
		amount := roundMoney(v.DiscountValue)
		if amount > base {
			amount = roundMoney(base)
		}
		return amount, nil
	default:
		return 0, ErrVoucherDiscountInvalid
	}
}

// ValidateVoucherInput memeriksa seluruh isian master voucher. Dipakai usecase
// dan dapat diuji tanpa database.
func ValidateVoucherInput(in VoucherInput) error {
	if NormalizeVoucherCode(in.Code) == "" {
		return NewBusinessRuleError("kode voucher wajib diisi")
	}
	if strings.TrimSpace(in.Name) == "" {
		return NewBusinessRuleError("nama voucher wajib diisi")
	}
	switch in.DiscountType {
	case VoucherDiscountPercent:
		if in.DiscountValue <= 0 || in.DiscountValue > 100 {
			return NewBusinessRuleError("diskon persen harus lebih dari 0 dan maksimal 100")
		}
	case VoucherDiscountAmount:
		if in.DiscountValue <= 0 {
			return NewBusinessRuleError("diskon nominal harus lebih dari 0")
		}
	default:
		return NewBusinessRuleError("jenis diskon voucher tidak dikenal")
	}
	if in.MinPurchase < 0 {
		return NewBusinessRuleError("minimal belanja tidak boleh negatif")
	}
	if in.MaxDiscount < 0 {
		return NewBusinessRuleError("maksimal diskon tidak boleh negatif")
	}
	if in.ValidFrom != nil && in.ValidUntil != nil {
		from := in.ValidFrom.Time
		until := in.ValidUntil.Time
		if until.Before(from) {
			return NewBusinessRuleError("tanggal berakhir tidak boleh sebelum tanggal mulai")
		}
	}
	if math.IsNaN(in.DiscountValue) || math.IsInf(in.DiscountValue, 0) {
		return NewBusinessRuleError("nilai diskon tidak valid")
	}
	return nil
}
