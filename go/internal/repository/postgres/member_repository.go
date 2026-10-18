package postgres

import (
	"context"
	"errors"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlMemberRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewSQLMemberRepository(pool *pgxpool.Pool) domain.MemberRepository {
	return &sqlMemberRepository{pool: pool, q: db.New(pool)}
}

// int8FromID mengubah id pengguna menjadi pgtype.Int8; id <= 0 berarti NULL.
func int8FromID(id int64) pgtype.Int8 {
	if id <= 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: id, Valid: true}
}

func mapMember(id int64, code, name, phone, email, address string, term int32, points int64, active bool, createdAt pgtype.Timestamptz) domain.Member {
	return domain.Member{
		ID:              id,
		Code:            code,
		Name:            name,
		Phone:           phone,
		Email:           email,
		Address:         address,
		DefaultTermDays: int(term),
		Points:          points,
		IsActive:        active,
		CreatedAt:       createdAt.Time,
	}
}

func (r *sqlMemberRepository) Create(ctx context.Context, in domain.MemberInput, actorID int64) (*domain.Member, error) {
	row, err := r.q.CreateMember(ctx, db.CreateMemberParams{
		Code:            in.Code,
		Name:            in.Name,
		Phone:           in.Phone,
		Email:           in.Email,
		Address:         in.Address,
		DefaultTermDays: int32(in.DefaultTermDays),
		CreatedBy:       int8FromID(actorID),
		UpdatedBy:       int8FromID(actorID),
	})
	if err != nil {
		return nil, err
	}
	m := mapMember(row.ID, row.Code, row.Name, row.Phone, row.Email, row.Address, row.DefaultTermDays, row.Points, row.IsActive, row.CreatedAt)
	return &m, nil
}

func (r *sqlMemberRepository) Update(ctx context.Context, id int64, in domain.MemberInput, actorID int64) error {
	return r.q.UpdateMember(ctx, db.UpdateMemberParams{
		Code:            in.Code,
		Name:            in.Name,
		Phone:           in.Phone,
		Email:           in.Email,
		Address:         in.Address,
		DefaultTermDays: int32(in.DefaultTermDays),
		IsActive:        in.IsActive,
		UpdatedBy:       int8FromID(actorID),
		ID:              id,
	})
}

func (r *sqlMemberRepository) Delete(ctx context.Context, id, actorID int64) error {
	return r.q.DeleteMember(ctx, db.DeleteMemberParams{
		DeletedBy: int8FromID(actorID),
		ID:        id,
	})
}

func (r *sqlMemberRepository) GetByID(ctx context.Context, id int64) (*domain.Member, error) {
	row, err := r.q.GetMemberByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	m := mapMember(row.ID, row.Code, row.Name, row.Phone, row.Email, row.Address, row.DefaultTermDays, row.Points, row.IsActive, row.CreatedAt)
	return &m, nil
}

func (r *sqlMemberRepository) GetListCursor(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]domain.Member, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	total, err := r.q.CountMembers(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	if direction == "prev" {
		rows, err := r.q.GetMembersCursorPrev(ctx, db.GetMembersCursorPrevParams{
			SearchQuery: search,
			CursorID:    cursorID,
			ReqLimit:    int32(limit),
		})
		if err != nil {
			return nil, 0, err
		}
		results := make([]domain.Member, 0, len(rows))
		for _, v := range rows {
			results = append(results, mapMember(v.ID, v.Code, v.Name, v.Phone, v.Email, v.Address, v.DefaultTermDays, v.Points, v.IsActive, v.CreatedAt))
		}
		for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
			results[i], results[j] = results[j], results[i]
		}
		return results, total, nil
	}

	rows, err := r.q.GetMembersCursorNext(ctx, db.GetMembersCursorNextParams{
		SearchQuery: search,
		CursorID:    cursorID,
		ReqLimit:    int32(limit),
	})
	if err != nil {
		return nil, 0, err
	}
	results := make([]domain.Member, 0, len(rows))
	for _, v := range rows {
		results = append(results, mapMember(v.ID, v.Code, v.Name, v.Phone, v.Email, v.Address, v.DefaultTermDays, v.Points, v.IsActive, v.CreatedAt))
	}
	return results, total, nil
}

func (r *sqlMemberRepository) LookupForPOS(ctx context.Context, search string, limit int) ([]domain.Member, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := r.q.LookupMembersForPOS(ctx, db.LookupMembersForPOSParams{
		SearchQuery: search,
		ReqLimit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}
	results := make([]domain.Member, 0, len(rows))
	for _, v := range rows {
		results = append(results, domain.Member{
			ID:              v.ID,
			Code:            v.Code,
			Name:            v.Name,
			Phone:           v.Phone,
			DefaultTermDays: int(v.DefaultTermDays),
			Points:          v.Points,
			IsActive:        true,
		})
	}
	return results, nil
}

func (r *sqlMemberRepository) ListPointEntries(ctx context.Context, memberID int64, limit int) ([]domain.MemberPointEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.q.ListMemberPointEntries(ctx, db.ListMemberPointEntriesParams{
		MemberID: memberID,
		ReqLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	results := make([]domain.MemberPointEntry, 0, len(rows))
	for _, v := range rows {
		e := domain.MemberPointEntry{
			ID:           v.ID,
			MemberID:     v.MemberID,
			EntryType:    v.EntryType,
			Points:       v.Points,
			BalanceAfter: v.BalanceAfter,
			Note:         v.Note,
			CreatedAt:    v.CreatedAt.Time,
		}
		if v.SaleID.Valid {
			id := v.SaleID.Int64
			e.SaleID = &id
		}
		if v.SaleNumber.Valid {
			e.SaleNumber = v.SaleNumber.String
		}
		results = append(results, e)
	}
	return results, nil
}
