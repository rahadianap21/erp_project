package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"go_backend/internal/domain"
)

type voucherRepoPalsu struct {
	domain.VoucherRepository
	created    *domain.VoucherInput
	lastLookup string
	lookup     *domain.Voucher
}

func (r *voucherRepoPalsu) Create(_ context.Context, in domain.VoucherInput, _ int64) (*domain.Voucher, error) {
	r.created = &in
	return &domain.Voucher{ID: 1, Code: in.Code, Name: in.Name}, nil
}

func (r *voucherRepoPalsu) GetByCode(_ context.Context, code string) (*domain.Voucher, error) {
	r.lastLookup = code
	return r.lookup, nil
}

func TestCreateVoucherMenormalkanInput(t *testing.T) {
	repo := &voucherRepoPalsu{}
	uc := NewVoucherUsecase(repo)

	if _, err := uc.Create(context.Background(), domain.VoucherInput{
		Code:          "  promo10 ",
		Name:          "  Promo  ",
		Description:   "  Keterangan  ",
		DiscountValue: 10,
	}, 1); err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("repo harus dipanggil")
	}
	if repo.created.Code != "PROMO10" || repo.created.Name != "Promo" || repo.created.Description != "Keterangan" {
		t.Fatalf("input tidak ternormalisasi: %+v", *repo.created)
	}
	if repo.created.DiscountType != domain.VoucherDiscountAmount {
		t.Fatalf("jenis diskon default harus AMOUNT, dapat %q", repo.created.DiscountType)
	}
}

func TestCreateVoucherMenolakInputTidakValidTanpaMemanggilRepo(t *testing.T) {
	repo := &voucherRepoPalsu{}
	uc := NewVoucherUsecase(repo)

	if _, err := uc.Create(context.Background(), domain.VoucherInput{
		Code:          "PROMO",
		Name:          "Promo",
		DiscountType:  domain.VoucherDiscountPercent,
		DiscountValue: 150,
	}, 1); err == nil {
		t.Fatal("persen di atas 100 harus ditolak")
	}
	if repo.created != nil {
		t.Fatal("repo tidak boleh dipanggil untuk input tidak valid")
	}
}

func TestGetByCodeVoucherMenormalkanDanMemeriksaStatus(t *testing.T) {
	repo := &voucherRepoPalsu{
		lookup: &domain.Voucher{
			ID:           1,
			Code:         "PROMO",
			Name:         "Promo",
			DiscountType: domain.VoucherDiscountAmount,
			IsActive:     true,
		},
	}
	uc := NewVoucherUsecase(repo)

	if _, err := uc.GetByCode(context.Background(), " promo "); err != nil {
		t.Fatalf("voucher aktif harus lolos: %v", err)
	}
	if repo.lastLookup != "PROMO" {
		t.Fatalf("lookup harus memakai kode ternormalisasi, dapat %q", repo.lastLookup)
	}

	used := time.Now()
	repo.lookup.UsedAt = &used
	if _, err := uc.GetByCode(context.Background(), "PROMO"); !errors.Is(err, domain.ErrVoucherUsed) {
		t.Fatalf("voucher terpakai harus ErrVoucherUsed, dapat %v", err)
	}

	repo.lookup = nil
	if _, err := uc.GetByCode(context.Background(), "PROMO"); !errors.Is(err, domain.ErrVoucherNotFound) {
		t.Fatalf("voucher tidak ada harus ErrVoucherNotFound, dapat %v", err)
	}

	if _, err := uc.GetByCode(context.Background(), "   "); err == nil {
		t.Fatal("kode kosong harus ditolak")
	}
}
