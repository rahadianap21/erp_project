package postgres

import (
	"errors"

	"go_backend/internal/domain"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgUniqueViolation adalah kode SQLSTATE PostgreSQL untuk pelanggaran unique
// constraint.
const pgUniqueViolation = "23505"

// asUniqueViolation mengenali pelanggaran unique constraint (SQLSTATE 23505)
// dan mengembalikan error pg-nya. Dipakai mapDuplicate dan repository PO yang
// punya kebutuhan pesan khusus (ErrNomorPOBentrok untuk retry penomoran).
func asUniqueViolation(err error) (*pgconn.PgError, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return pgErr, true
	}
	return nil, false
}

// mapDuplicate menerjemahkan pelanggaran unique constraint (SQLSTATE 23505)
// menjadi domain.DuplicateError dengan pesan yang ramah untuk user.
//
// Tanpa terjemahan ini, lapisan handler harus memeriksa string mentah error
// PostgreSQL ("23505", "duplicate key value") yang bocor ke banyak tempat.
// Pesan disini spesifik per entitas karena handler sebelumnya menulis pesan
// sendiri ("kode member sudah dipakai", dsb.) — pesan itu dipindahkan ke
// repository agar pemetaan status HTTP cukup mengenali satu tipe error.
func mapDuplicate(err error, pesan string) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return domain.NewDuplicateError("%s", pesan)
	}
	return err
}
