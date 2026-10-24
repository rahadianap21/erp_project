package usecase

import (
	"context"
	"strings"
	"time"

	"go_backend/internal/domain"
)

type voucherUsecase struct {
	repo domain.VoucherRepository
}

func NewVoucherUsecase(repo domain.VoucherRepository) domain.VoucherUsecase {
	return &voucherUsecase{repo: repo}
}

func normalizeVoucherInput(in domain.VoucherInput) domain.VoucherInput {
	in.Code = domain.NormalizeVoucherCode(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.DiscountType == "" {
		in.DiscountType = domain.VoucherDiscountAmount
	}
	return in
}

func (u *voucherUsecase) Create(ctx context.Context, in domain.VoucherInput, actorID int64) (*domain.Voucher, error) {
	in = normalizeVoucherInput(in)
	if err := domain.ValidateVoucherInput(in); err != nil {
		return nil, err
	}
	return u.repo.Create(ctx, in, actorID)
}

func (u *voucherUsecase) Update(ctx context.Context, id int64, in domain.VoucherInput, actorID int64) error {
	if id <= 0 {
		return domain.NewBusinessRuleError("id voucher tidak valid")
	}
	in = normalizeVoucherInput(in)
	if err := domain.ValidateVoucherInput(in); err != nil {
		return err
	}
	return u.repo.Update(ctx, id, in, actorID)
}

func (u *voucherUsecase) Delete(ctx context.Context, id, actorID int64) error {
	if id <= 0 {
		return domain.NewBusinessRuleError("id voucher tidak valid")
	}
	return u.repo.Delete(ctx, id, actorID)
}

func (u *voucherUsecase) GetByID(ctx context.Context, id int64) (*domain.Voucher, error) {
	if id <= 0 {
		return nil, domain.NewBusinessRuleError("id voucher tidak valid")
	}
	return u.repo.GetByID(ctx, id)
}

// GetByCode mencari voucher dari isi barcode lalu memastikan voucher bisa
// dipakai saat ini. Dipakai endpoint lookup POS.
func (u *voucherUsecase) GetByCode(ctx context.Context, code string) (*domain.Voucher, error) {
	code = domain.NormalizeVoucherCode(code)
	if code == "" {
		return nil, domain.NewBusinessRuleError("kode voucher wajib diisi")
	}
	v, err := u.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, domain.ErrVoucherNotFound
	}
	if err := v.ValidateUsable(time.Now()); err != nil {
		return nil, err
	}
	return v, nil
}

func (u *voucherUsecase) GetAll(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]domain.Voucher, int64, error) {
	if direction != "prev" {
		direction = "next"
	}
	return u.repo.GetListCursor(ctx, strings.TrimSpace(search), cursorID, limit, direction)
}
