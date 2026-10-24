package postgres

import (
	"errors"

	"go_backend/internal/domain"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgUniqueViolation adalah kode SQLSTATE PostgreSQL untuk pelanggaran unique
// constraint.
const pgUniqueViolation = "23505"

<<<<<<< HEAD
// journalSourceUniqueIndex adalah nama unique index parsial yang menjamin hanya
// satu jurnal POSTED BELUM DIBALIK untuk tiap (source_type, source_id) — inilah
// pengaman idempotensi auto-posting (GRN) dan jurnal penutup. Jurnal yang sudah
// dibalik dikecualikan, sehingga sumber yang sama boleh menerbitkan jurnal baru
// setelah pembalikan (mis. periode ditutup ulang).
const journalSourceUniqueIndex = "idx_journal_source_unique"

// uniqueViolationConstraint mengekstrak nama constraint ketika error adalah
// pelanggaran unique. false bila bukan unique violation.
func uniqueViolationConstraint(err error) (string, bool) {
	if pgErr, ok := asUniqueViolation(err); ok {
		return pgErr.ConstraintName, true
	}
	return "", false
}

=======
>>>>>>> e47ff22 (refactor(go): sentralisasi pemetaan error domain ke handler.WriteError)
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
