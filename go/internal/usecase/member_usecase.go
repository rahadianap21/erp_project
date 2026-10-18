package usecase

import (
	"context"
	"errors"
	"strings"

	"go_backend/internal/domain"
)

type memberUsecase struct {
	repo domain.MemberRepository
}

func NewMemberUsecase(repo domain.MemberRepository) domain.MemberUsecase {
	return &memberUsecase{repo: repo}
}

func (u *memberUsecase) normalize(in domain.MemberInput) domain.MemberInput {
	in.Code = strings.TrimSpace(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Email = strings.TrimSpace(in.Email)
	in.Address = strings.TrimSpace(in.Address)
	return in
}

func (u *memberUsecase) Create(ctx context.Context, in domain.MemberInput, actorID int64) (*domain.Member, error) {
	in = u.normalize(in)
	if in.Code == "" {
		return nil, errors.New("kode member wajib diisi")
	}
	if in.Name == "" {
		return nil, errors.New("nama member wajib diisi")
	}
	if in.DefaultTermDays < 0 {
		return nil, errors.New("tempo member tidak boleh negatif")
	}
	return u.repo.Create(ctx, in, actorID)
}

func (u *memberUsecase) Update(ctx context.Context, id int64, in domain.MemberInput, actorID int64) error {
	if id <= 0 {
		return errors.New("id member tidak valid")
	}
	in = u.normalize(in)
	if in.Code == "" || in.Name == "" {
		return errors.New("kode dan nama member wajib diisi")
	}
	if in.DefaultTermDays < 0 {
		return errors.New("tempo member tidak boleh negatif")
	}
	return u.repo.Update(ctx, id, in, actorID)
}

func (u *memberUsecase) Delete(ctx context.Context, id, actorID int64) error {
	if id <= 0 {
		return errors.New("id member tidak valid")
	}
	return u.repo.Delete(ctx, id, actorID)
}

func (u *memberUsecase) GetByID(ctx context.Context, id int64) (*domain.Member, error) {
	if id <= 0 {
		return nil, errors.New("id member tidak valid")
	}
	return u.repo.GetByID(ctx, id)
}

func (u *memberUsecase) GetAll(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]domain.Member, int64, error) {
	if direction != "prev" {
		direction = "next"
	}
	return u.repo.GetListCursor(ctx, strings.TrimSpace(search), cursorID, limit, direction)
}

func (u *memberUsecase) LookupForPOS(ctx context.Context, search string, limit int) ([]domain.Member, error) {
	return u.repo.LookupForPOS(ctx, strings.TrimSpace(search), limit)
}

func (u *memberUsecase) ListPointEntries(ctx context.Context, memberID int64, limit int) ([]domain.MemberPointEntry, error) {
	if memberID <= 0 {
		return nil, errors.New("id member tidak valid")
	}
	return u.repo.ListPointEntries(ctx, memberID, limit)
}
