package postgres

import (
	"math"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// File ini memusatkan seluruh helper konversi tipe pgtype ↔ tipe Go yang
// sebelumnya tersebar di purchase_order_repository, sale_repository,
// accounting_repository, dan cursor.go.

// convertPgInt64 mengubah pgtype.Int8 menjadi *int64; null menjadi nil.
func convertPgInt64(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	val := v.Int64
	return &val
}

// convertPgTime mengubah pgtype.Timestamp menjadi *time.Time; null menjadi nil.
func convertPgTime(v pgtype.Timestamp) *time.Time {
	if !v.Valid {
		return nil
	}
	val := v.Time
	return &val
}

// convertPgTimeZ mengubah pgtype.Timestamptz menjadi *time.Time; null menjadi nil.
func convertPgTimeZ(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	val := v.Time
	return &val
}

// numericFromFloat mengubah float64 menjadi NUMERIC 2 desimal tanpa pembulatan
// floating point. Numerik uang tidak boleh lewat float.
func numericFromFloat(v float64) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(int64(math.Round(v * 100))), Exp: -2, Valid: true}
}

// numericFromFloat4 mengubah float64 menjadi NUMERIC 4 desimal (cost_unit).
func numericFromFloat4(v float64) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(int64(math.Round(v * 10000))), Exp: -4, Valid: true}
}

// numericToFloat mengekstrak nilai NUMERIC menjadi float64; null menjadi 0.
func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

// nullableInt8 mengubah *int64 menjadi pgtype.Int8; nil berarti NULL.
func nullableInt8(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: *v, Valid: true}
}

// reverseRows membalik urutan slice "Prev" cursor pagination agar tiba dalam
// urutan yang sama dengan halaman "Next". Query Prev diurutkan ASC sehingga
// hasilnya tiba terbalik dan wajib dibalik sebelum dikirim ke klien.
func reverseRows[T any](rows []T) {
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
}
