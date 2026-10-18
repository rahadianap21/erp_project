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

type sqlPaymentMethodRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewSQLPaymentMethodRepository(pool *pgxpool.Pool) domain.PaymentMethodRepository {
	return &sqlPaymentMethodRepository{pool: pool, q: db.New(pool)}
}

func accountIDOrNull(id *int64) pgtype.Int8 {
	if id == nil || *id <= 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *id, Valid: true}
}

func accountIDFromPg(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	id := v.Int64
	return &id
}

func (r *sqlPaymentMethodRepository) Create(ctx context.Context, in domain.PaymentMethodInput, actorID int64) (*domain.PaymentMethod, error) {
	row, err := r.q.CreatePaymentMethod(ctx, db.CreatePaymentMethodParams{
		Code:      in.Code,
		Name:      in.Name,
		Channel:   in.Channel,
		AccountID: accountIDOrNull(in.AccountID),
		IsCash:    in.IsCash,
		SortOrder: int32(in.SortOrder),
		CreatedBy: int8FromID(actorID),
		UpdatedBy: int8FromID(actorID),
	})
	if err != nil {
		return nil, err
	}
	return &domain.PaymentMethod{
		ID:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		Channel:   row.Channel,
		AccountID: accountIDFromPg(row.AccountID),
		IsCash:    row.IsCash,
		SortOrder: int(row.SortOrder),
		IsActive:  row.IsActive,
	}, nil
}

func (r *sqlPaymentMethodRepository) Update(ctx context.Context, id int64, in domain.PaymentMethodInput, actorID int64) error {
	return r.q.UpdatePaymentMethod(ctx, db.UpdatePaymentMethodParams{
		Code:      in.Code,
		Name:      in.Name,
		Channel:   in.Channel,
		AccountID: accountIDOrNull(in.AccountID),
		IsCash:    in.IsCash,
		SortOrder: int32(in.SortOrder),
		IsActive:  in.IsActive,
		UpdatedBy: int8FromID(actorID),
		ID:        id,
	})
}

func (r *sqlPaymentMethodRepository) Delete(ctx context.Context, id, actorID int64) error {
	return r.q.DeletePaymentMethod(ctx, db.DeletePaymentMethodParams{
		DeletedBy: int8FromID(actorID),
		ID:        id,
	})
}

func (r *sqlPaymentMethodRepository) GetByID(ctx context.Context, id int64) (*domain.PaymentMethod, error) {
	row, err := r.q.GetPaymentMethodByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.PaymentMethod{
		ID:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		Channel:   row.Channel,
		AccountID: accountIDFromPg(row.AccountID),
		IsCash:    row.IsCash,
		SortOrder: int(row.SortOrder),
		IsActive:  row.IsActive,
	}, nil
}

func (r *sqlPaymentMethodRepository) ListActive(ctx context.Context) ([]domain.PaymentMethod, error) {
	rows, err := r.q.GetActivePaymentMethods(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.PaymentMethod, 0, len(rows))
	for _, v := range rows {
		result = append(result, domain.PaymentMethod{
			ID:        v.ID,
			Code:      v.Code,
			Name:      v.Name,
			Channel:   v.Channel,
			AccountID: accountIDFromPg(v.AccountID),
			IsCash:    v.IsCash,
			SortOrder: int(v.SortOrder),
			IsActive:  v.IsActive,
		})
	}
	return result, nil
}

func (r *sqlPaymentMethodRepository) ListAll(ctx context.Context) ([]domain.PaymentMethod, error) {
	rows, err := r.q.ListPaymentMethods(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.PaymentMethod, 0, len(rows))
	for _, v := range rows {
		result = append(result, domain.PaymentMethod{
			ID:        v.ID,
			Code:      v.Code,
			Name:      v.Name,
			Channel:   v.Channel,
			AccountID: accountIDFromPg(v.AccountID),
			IsCash:    v.IsCash,
			SortOrder: int(v.SortOrder),
			IsActive:  v.IsActive,
		})
	}
	return result, nil
}
