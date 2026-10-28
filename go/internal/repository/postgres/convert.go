package postgres

import (
	"math"
	"math/big"
	"time"

	"go_backend/internal/domain"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
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

// numericFromMoney mengubah Money menjadi NUMERIC tanpa melewati float.
// Nilai selalu diekspresikan dalam sen (Exp -2) sehingga tidak ada kehilangan
// presisi.
func numericFromMoney(m domain.Money) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(m.Cents()), Exp: -2, Valid: true}
}

// numericFromDecimal mengubah Decimal menjadi NUMERIC 4 desimal. Empat desimal
// dipakai agar kolom NUMERIC(15,4) seperti avg_cost/cost_unit tetap utuh.
func numericFromDecimal(d domain.Decimal) pgtype.Numeric {
	return numericFromDecimalScale(d, -4)
}

// numericFromDecimalScale mengubah Decimal menjadi NUMERIC dengan skala exp
// (mis. -6 untuk faktor biaya perolehan). Pembulatan dilakukan eksplisit agar
// hasilnya tidak bergantung pada database.
func numericFromDecimalScale(d domain.Decimal, exp int32) pgtype.Numeric {
	mant := d.Raw().Shift(-exp).Round(0)
	return pgtype.Numeric{Int: new(big.Int).Set(mant.BigInt()), Exp: exp, Valid: true}
}

// moneyFromNumeric mengekstrak NUMERIC menjadi Money; null menjadi Rp 0.
func moneyFromNumeric(n pgtype.Numeric) domain.Money {
	if !n.Valid {
		return domain.Money{}
	}
	return domain.MoneyFromDecimal(numericToDecimal(n))
}

// decimalFromNumeric mengekstrak NUMERIC menjadi Decimal; null menjadi 0.
func decimalFromNumeric(n pgtype.Numeric) domain.Decimal {
	if !n.Valid {
		return domain.Decimal{}
	}
	return domain.DecFromDecimal(numericToDecimal(n))
}

// numericToDecimal mengubah pgtype.Numeric menjadi decimal secara eksak tanpa
// melewati float64.
func numericToDecimal(n pgtype.Numeric) decimal.Decimal {
	if !n.Valid || n.Int == nil {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(n.Int, n.Exp)
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

// numericFromFloat dipertahankan sementara untuk data lama; setelah seluruh
// modul memakai Money/Decimal helper ini tidak boleh dipakai lagi.
func numericFromFloat(v float64) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(int64(math.Round(v * 100))), Exp: -2, Valid: true}
}

// numericToFloat dipertahankan sementara untuk data lama.
func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}
