package domain

import (
	"errors"
	"testing"
	"time"
)

func tanggalUji(t time.Time, hari int) *DateOnly {
	d := t.AddDate(0, 0, hari)
	return &DateOnly{Time: d}
}

func TestNormalizeVoucherCode(t *testing.T) {
	if got := NormalizeVoucherCode("  promo10 "); got != "PROMO10" {
		t.Fatalf("harus PROMO10, dapat %q", got)
	}
}

func TestValidateVoucherInput(t *testing.T) {
	valid := VoucherInput{
		Code:          "promo",
		Name:          "Promo",
		DiscountType:  VoucherDiscountPercent,
		DiscountValue: DecInt(10),
		IsActive:      true,
	}
	if err := ValidateVoucherInput(valid); err != nil {
		t.Fatalf("input valid tidak boleh error: %v", err)
	}
	cases := []struct {
		name string
		in   VoucherInput
	}{
		{"kode kosong", VoucherInput{Name: "X", DiscountType: VoucherDiscountAmount, DiscountValue: DecInt(1000)}},
		{"nama kosong", VoucherInput{Code: "X", DiscountType: VoucherDiscountAmount, DiscountValue: DecInt(1000)}},
		{"persen nol", VoucherInput{Code: "X", Name: "X", DiscountType: VoucherDiscountPercent}},
		{"persen lebih 100", VoucherInput{Code: "X", Name: "X", DiscountType: VoucherDiscountPercent, DiscountValue: DecInt(101)}},
		{"nominal nol", VoucherInput{Code: "X", Name: "X", DiscountType: VoucherDiscountAmount}},
		{"jenis salah", VoucherInput{Code: "X", Name: "X", DiscountType: "BOGUS", DiscountValue: DecInt(1)}},
	}
	for _, c := range cases {
		if err := ValidateVoucherInput(c.in); err == nil {
			t.Fatalf("%s seharusnya ditolak", c.name)
		}
	}
}

func TestVoucherValidateUsable(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	base := Voucher{Code: "X", Name: "X", DiscountType: VoucherDiscountAmount, DiscountValue: DecInt(1000), IsActive: true}

	inactive := base
	inactive.IsActive = false
	if !errors.Is(inactive.ValidateUsable(now), ErrVoucherInactive) {
		t.Fatal("voucher nonaktif harus ditolak")
	}

	used := base
	usedAt := now.Add(-time.Hour)
	used.UsedAt = &usedAt
	if !errors.Is(used.ValidateUsable(now), ErrVoucherUsed) {
		t.Fatal("voucher terpakai harus ditolak")
	}

	future := base
	future.ValidFrom = tanggalUji(now, 1)
	if !errors.Is(future.ValidateUsable(now), ErrVoucherNotStarted) {
		t.Fatal("voucher belum berlaku harus ditolak")
	}

	expired := base
	expired.ValidUntil = tanggalUji(now, -1)
	if !errors.Is(expired.ValidateUsable(now), ErrVoucherExpired) {
		t.Fatal("voucher kedaluwarsa harus ditolak")
	}

	today := base
	today.ValidFrom = tanggalUji(now, 0)
	today.ValidUntil = tanggalUji(now, 0)
	if err := today.ValidateUsable(now); err != nil {
		t.Fatalf("voucher yang berlaku hari ini harus lolos: %v", err)
	}
}
