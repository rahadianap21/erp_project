package usecase

import (
	"context"
	"strings"

	"go_backend/internal/domain"
)

type paymentMethodUsecase struct {
	repo domain.PaymentMethodRepository
}

func NewPaymentMethodUsecase(repo domain.PaymentMethodRepository) domain.PaymentMethodUsecase {
	return &paymentMethodUsecase{repo: repo}
}

var validPaymentChannels = map[string]bool{
	domain.PaymentChannelCash:    true,
	domain.PaymentChannelBank:    true,
	domain.PaymentChannelEWallet: true,
	domain.PaymentChannelCredit:  true,
}

func (u *paymentMethodUsecase) validate(in domain.PaymentMethodInput) error {
	if strings.TrimSpace(in.Code) == "" {
		return domain.NewBusinessRuleError("kode metode pembayaran wajib diisi")
	}
	if strings.TrimSpace(in.Name) == "" {
		return domain.NewBusinessRuleError("nama metode pembayaran wajib diisi")
	}
	if !validPaymentChannels[in.Channel] {
		return domain.NewBusinessRuleError("kanal metode pembayaran tidak dikenal")
	}
	if in.Channel != domain.PaymentChannelCredit && (in.AccountID == nil || *in.AccountID <= 0) {
		return domain.NewBusinessRuleError("akun kas/bank metode pembayaran wajib dipilih")
	}
	return nil
}

func (u *paymentMethodUsecase) Create(ctx context.Context, in domain.PaymentMethodInput, actorID int64) (*domain.PaymentMethod, error) {
	in.Code = strings.TrimSpace(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	if err := u.validate(in); err != nil {
		return nil, err
	}
	return u.repo.Create(ctx, in, actorID)
}

func (u *paymentMethodUsecase) Update(ctx context.Context, id int64, in domain.PaymentMethodInput, actorID int64) error {
	if id <= 0 {
		return domain.NewBusinessRuleError("id metode pembayaran tidak valid")
	}
	in.Code = strings.TrimSpace(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	if err := u.validate(in); err != nil {
		return err
	}
	return u.repo.Update(ctx, id, in, actorID)
}

func (u *paymentMethodUsecase) Delete(ctx context.Context, id, actorID int64) error {
	if id <= 0 {
		return domain.NewBusinessRuleError("id metode pembayaran tidak valid")
	}
	return u.repo.Delete(ctx, id, actorID)
}

func (u *paymentMethodUsecase) GetByID(ctx context.Context, id int64) (*domain.PaymentMethod, error) {
	if id <= 0 {
		return nil, domain.NewBusinessRuleError("id metode pembayaran tidak valid")
	}
	return u.repo.GetByID(ctx, id)
}

func (u *paymentMethodUsecase) ListActive(ctx context.Context) ([]domain.PaymentMethod, error) {
	return u.repo.ListActive(ctx)
}

func (u *paymentMethodUsecase) ListAll(ctx context.Context) ([]domain.PaymentMethod, error) {
	return u.repo.ListAll(ctx)
}
