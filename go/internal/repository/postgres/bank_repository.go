package postgres

import (
	"context"
	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlBankRepository struct {
	q *db.Queries
}

func NewSQLBankRepository(pool *pgxpool.Pool) domain.BankRepository {
	return &sqlBankRepository{q: db.New(pool)}
}

func (r *sqlBankRepository) GetAllActive(ctx context.Context) ([]domain.Bank, error) {
	rows, err := r.q.GetAllActiveBanks(ctx)
	if err != nil {
		return nil, err
	}

	var res []domain.Bank
	for _, row := range rows {
		// 🌟 SINKRONISASI DATA ROW BARU: Petakan id angka dan kode teks secara berurutan
		res = append(res, domain.Bank{
			ID:        row.ID,
			Kode:      row.Kode,
			Name:      row.Name,
			ShortName: row.ShortName,
		})
	}
	return res, nil
}
