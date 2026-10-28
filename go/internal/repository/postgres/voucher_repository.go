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

type sqlVoucherRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewSQLVoucherRepository(pool *pgxpool.Pool) domain.VoucherRepository {
	return &sqlVoucherRepository{pool: pool, q: db.New(pool)}
}

func dateOnlyToPg(v *domain.DateOnly) pgtype.Date {
	if v == nil || v.IsZero() {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: v.Time, Valid: true}
}

func dateOnlyFromPg(v pgtype.Date) *domain.DateOnly {
	if !v.Valid {
		return nil
	}
	return &domain.DateOnly{Time: v.Time}
}

func mapVoucher(
	id int64,
	code, name, description, discountType string,
	discountValue, minPurchase, maxDiscount pgtype.Numeric,
	validFrom, validUntil pgtype.Date,
	isActive bool,
	usedAt, createdAt pgtype.Timestamptz,
) domain.Voucher {
	return domain.Voucher{
		ID:            id,
		Code:          code,
		Name:          name,
		Description:   description,
		DiscountType:  discountType,
		DiscountValue: decimalFromNumeric(discountValue),
		MinPurchase:   moneyFromNumeric(minPurchase),
		MaxDiscount:   moneyFromNumeric(maxDiscount),
		ValidFrom:     dateOnlyFromPg(validFrom),
		ValidUntil:    dateOnlyFromPg(validUntil),
		IsActive:      isActive,
		UsedAt:        convertPgTimeZ(usedAt),
		CreatedAt:     createdAt.Time,
	}
}

func (r *sqlVoucherRepository) Create(ctx context.Context, in domain.VoucherInput, actorID int64) (*domain.Voucher, error) {
	row, err := r.q.CreateVoucher(ctx, db.CreateVoucherParams{
		Code:          domain.NormalizeVoucherCode(in.Code),
		Name:          in.Name,
		Description:   in.Description,
		DiscountType:  in.DiscountType,
		DiscountValue: numericFromDecimal(in.DiscountValue),
		MinPurchase:   numericFromMoney(in.MinPurchase),
		MaxDiscount:   numericFromMoney(in.MaxDiscount),
		ValidFrom:     dateOnlyToPg(in.ValidFrom),
		ValidUntil:    dateOnlyToPg(in.ValidUntil),
		IsActive:      in.IsActive,
		CreatedBy:     int8FromID(actorID),
		UpdatedBy:     int8FromID(actorID),
	})
	if err != nil {
		return nil, mapDuplicate(err, "kode voucher sudah dipakai")
	}
	v := mapVoucher(row.ID, row.Code, row.Name, row.Description, row.DiscountType,
		row.DiscountValue, row.MinPurchase, row.MaxDiscount, row.ValidFrom, row.ValidUntil,
		row.IsActive, row.UsedAt, row.CreatedAt)
	return &v, nil
}

func (r *sqlVoucherRepository) Update(ctx context.Context, id int64, in domain.VoucherInput, actorID int64) error {
	err := r.q.UpdateVoucher(ctx, db.UpdateVoucherParams{
		Code:          domain.NormalizeVoucherCode(in.Code),
		Name:          in.Name,
		Description:   in.Description,
		DiscountType:  in.DiscountType,
		DiscountValue: numericFromDecimal(in.DiscountValue),
		MinPurchase:   numericFromMoney(in.MinPurchase),
		MaxDiscount:   numericFromMoney(in.MaxDiscount),
		ValidFrom:     dateOnlyToPg(in.ValidFrom),
		ValidUntil:    dateOnlyToPg(in.ValidUntil),
		IsActive:      in.IsActive,
		UpdatedBy:     int8FromID(actorID),
		ID:            id,
	})
	return mapDuplicate(err, "kode voucher sudah dipakai")
}

func (r *sqlVoucherRepository) Delete(ctx context.Context, id, actorID int64) error {
	return r.q.DeleteVoucher(ctx, db.DeleteVoucherParams{
		DeletedBy: int8FromID(actorID),
		ID:        id,
	})
}

func (r *sqlVoucherRepository) GetByID(ctx context.Context, id int64) (*domain.Voucher, error) {
	row, err := r.q.GetVoucherByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	v := mapVoucher(row.ID, row.Code, row.Name, row.Description, row.DiscountType,
		row.DiscountValue, row.MinPurchase, row.MaxDiscount, row.ValidFrom, row.ValidUntil,
		row.IsActive, row.UsedAt, row.CreatedAt)
	return &v, nil
}

func (r *sqlVoucherRepository) GetByCode(ctx context.Context, code string) (*domain.Voucher, error) {
	row, err := r.q.GetVoucherByCode(ctx, domain.NormalizeVoucherCode(code))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	v := mapVoucher(row.ID, row.Code, row.Name, row.Description, row.DiscountType,
		row.DiscountValue, row.MinPurchase, row.MaxDiscount, row.ValidFrom, row.ValidUntil,
		row.IsActive, row.UsedAt, row.CreatedAt)
	return &v, nil
}

func (r *sqlVoucherRepository) GetListCursor(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]domain.Voucher, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	total, err := r.q.CountVouchers(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	if direction == "prev" {
		rows, err := r.q.GetVouchersCursorPrev(ctx, db.GetVouchersCursorPrevParams{
			SearchQuery: search,
			CursorID:    cursorID,
			ReqLimit:    int32(limit),
		})
		if err != nil {
			return nil, 0, err
		}
		results := make([]domain.Voucher, 0, len(rows))
		for _, v := range rows {
			results = append(results, mapVoucher(v.ID, v.Code, v.Name, v.Description, v.DiscountType,
				v.DiscountValue, v.MinPurchase, v.MaxDiscount, v.ValidFrom, v.ValidUntil,
				v.IsActive, v.UsedAt, v.CreatedAt))
		}
		for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
			results[i], results[j] = results[j], results[i]
		}
		return results, total, nil
	}

	rows, err := r.q.GetVouchersCursorNext(ctx, db.GetVouchersCursorNextParams{
		SearchQuery: search,
		CursorID:    cursorID,
		ReqLimit:    int32(limit),
	})
	if err != nil {
		return nil, 0, err
	}
	results := make([]domain.Voucher, 0, len(rows))
	for _, v := range rows {
		results = append(results, mapVoucher(v.ID, v.Code, v.Name, v.Description, v.DiscountType,
			v.DiscountValue, v.MinPurchase, v.MaxDiscount, v.ValidFrom, v.ValidUntil,
			v.IsActive, v.UsedAt, v.CreatedAt))
	}
	return results, total, nil
}
