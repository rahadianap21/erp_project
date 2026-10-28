package domain

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// =========================================================================
// MONEY — nominal rupiah dalam satuan sen
// =========================================================================

// ErrMoneyInvalid dikembalikan saat teks/nilai yang diberikan bukan nominal
// uang yang sah.
var ErrMoneyInvalid = errors.New("nilai uang tidak valid")

// Money menyimpan nominal rupiah dalam satuan sen (1/100 rupiah). Penjumlahan,
// pengurangan, dan perkalian dengan kuantitas bulat menjadi eksak sehingga
// tidak ada galat pembulatan floating point.
//
// Nilai nol dari Money berarti Rp 0, jadi struct ini aman dipakai sebagai field
// tanpa inisialisasi khusus.
type Money struct {
	cents int64
}

// NewMoney membuat Money dari jumlah sen.
func NewMoney(cents int64) Money {
	return Money{cents: cents}
}

// Rupiah membuat Money dari rupiah bulat.
func Rupiah(rupiah int64) Money {
	return Money{cents: rupiah * 100}
}

// MoneyFromDecimal mengubah decimal menjadi Money, dibulatkan ke sen terdekat
// dengan metode setengah menjauhi nol (half away from zero) seperti kebiasaan
// pembulatan rupiah.
func MoneyFromDecimal(d decimal.Decimal) Money {
	return Money{cents: d.Mul(decimal.NewFromInt(100)).Round(0).IntPart()}
}

// MoneyFromFloat mengubah float64 menjadi Money. Hanya dipakai untuk data lama
// atau batas API; perhitungan internal wajib memakai decimal/Money.
func MoneyFromFloat(v float64) Money {
	return MoneyFromDecimal(decimal.NewFromFloat(v))
}

// MoneyFromString membaca nominal dari teks, mis. "1.234,56" sudah
// dinormalisasi pemanggil menjadi "1234.56".
func MoneyFromString(s string) (Money, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return Money{}, fmt.Errorf("%w: %q", ErrMoneyInvalid, s)
	}
	return MoneyFromDecimal(d), nil
}

// Cents mengembalikan jumlah sen.
func (m Money) Cents() int64 { return m.cents }

// Decimal mengembalikan representasi decimal dari nominal ini.
func (m Money) Decimal() decimal.Decimal {
	return decimal.New(m.cents, -2)
}

// IsZero melaporkan apakah nominal nol.
func (m Money) IsZero() bool { return m.cents == 0 }

// IsNegative melaporkan apakah nominal negatif.
func (m Money) IsNegative() bool { return m.cents < 0 }

// IsPositive melaporkan apakah nominal lebih besar dari nol.
func (m Money) IsPositive() bool { return m.cents > 0 }

// Add menjumlahkan dua nominal.
func (m Money) Add(o Money) Money { return Money{cents: m.cents + o.cents} }

// Sub mengurangkan dua nominal.
func (m Money) Sub(o Money) Money { return Money{cents: m.cents - o.cents} }

// Neg membalik tanda nominal.
func (m Money) Neg() Money { return Money{cents: -m.cents} }

// Abs mengembalikan nilai absolut.
func (m Money) Abs() Money {
	if m.cents < 0 {
		return Money{cents: -m.cents}
	}
	return m
}

// MulInt mengalikan nominal dengan bilangan bulat (mis. kuantitas).
func (m Money) MulInt(n int64) Money { return Money{cents: m.cents * n} }

// MulDecimal mengalikan nominal dengan rasio/persentase dan membulatkan hasil
// ke sen terdekat.
func (m Money) MulDecimal(rate decimal.Decimal) Money {
	if rate.IsZero() || m.cents == 0 {
		return Money{}
	}
	return Money{cents: decimal.NewFromInt(m.cents).Mul(rate).Round(0).IntPart()}
}

// DivDecimal membagi nominal dengan pembagi dan membulatkan hasil ke sen.
func (m Money) DivDecimal(divisor decimal.Decimal) Money {
	if divisor.IsZero() {
		return Money{}
	}
	return Money{cents: decimal.NewFromInt(m.cents).DivRound(divisor, 0).IntPart()}
}

// Cmp membandingkan dua nominal: -1 bila m < o, 0 bila sama, 1 bila m > o.
func (m Money) Cmp(o Money) int {
	switch {
	case m.cents < o.cents:
		return -1
	case m.cents > o.cents:
		return 1
	default:
		return 0
	}
}

// Max mengembalikan nominal terbesar.
func (m Money) Max(o Money) Money {
	if m.cents >= o.cents {
		return m
	}
	return o
}

// Min mengembalikan nominal terkecil.
func (m Money) Min(o Money) Money {
	if m.cents <= o.cents {
		return m
	}
	return o
}

// String mengembalikan nominal sebagai angka desimal tanpa pemisah ribuan,
// mis. "1234.56" atau "1234" bila tanpa sen.
func (m Money) String() string { return m.Decimal().String() }

// MarshalJSON mengirim nominal sebagai ANGKA rupiah (bukan sen dan bukan
// string) supaya kontrak API ke frontend tidak berubah.
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalJSON menerima angka rupiah maupun string angka.
func (m *Money) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*m = Money{}
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if strings.EqualFold(s, "NaN") || strings.Contains(strings.ToLower(s), "inf") {
		return fmt.Errorf("%w: %q", ErrMoneyInvalid, s)
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrMoneyInvalid, s)
	}
	*m = MoneyFromDecimal(d)
	return nil
}

// Value mengimplementasikan driver.Valuer untuk kolom NUMERIC.
func (m Money) Value() (driver.Value, error) { return m.String(), nil }

// Scan mengimplementasikan sql.Scanner untuk kolom NUMERIC.
func (m *Money) Scan(v any) error {
	switch val := v.(type) {
	case nil:
		*m = Money{}
	case int64:
		*m = NewMoney(val)
	case float64:
		*m = MoneyFromFloat(val)
	case string:
		parsed, err := MoneyFromString(val)
		if err != nil {
			return err
		}
		*m = parsed
	case []byte:
		parsed, err := MoneyFromString(string(val))
		if err != nil {
			return err
		}
		*m = parsed
	case decimal.Decimal:
		*m = MoneyFromDecimal(val)
	default:
		return fmt.Errorf("%w: tipe %T", ErrMoneyInvalid, v)
	}
	return nil
}

// =========================================================================
// DECIMAL — angka non-uang (tarif, persen, kuantitas pecahan)
// =========================================================================

// Decimal adalah pembungkus shopspring/decimal yang tetap dikirim sebagai
// ANGKA JSON. shopspring/decimal secara bawaan mengirim string, sedangkan
// kontrak API aplikasi ini memakai number; pembungkus ini menjaga kontrak
// tersebut tanpa mengorbankan presisi desimal.
type Decimal struct {
	d decimal.Decimal
}

// DecInt membuat Decimal dari bilangan bulat.
func DecInt(v int64) Decimal { return Decimal{d: decimal.NewFromInt(v)} }

// DecFromString membaca Decimal dari teks.
func DecFromString(s string) (Decimal, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return Decimal{}, fmt.Errorf("nilai desimal tidak valid: %q", s)
	}
	return Decimal{d: d}, nil
}

// DecFromFloat mengubah float64 menjadi Decimal. Hanya untuk data lama.
func DecFromFloat(v float64) Decimal { return Decimal{d: decimal.NewFromFloat(v)} }

// DecFromDecimal membungkus decimal yang sudah ada.
func DecFromDecimal(d decimal.Decimal) Decimal { return Decimal{d: d} }

// MustDec membuat Decimal dari literal tetap; dipakai untuk nilai bawaan.
func MustDec(s string) Decimal {
	d, err := DecFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

// Raw mengembalikan nilai shopspring/decimal asli.
func (d Decimal) Raw() decimal.Decimal { return d.d }

// Add menjumlahkan dua nilai desimal.
func (d Decimal) Add(o Decimal) Decimal { return Decimal{d: d.d.Add(o.d)} }

// Sub mengurangkan dua nilai desimal.
func (d Decimal) Sub(o Decimal) Decimal { return Decimal{d: d.d.Sub(o.d)} }

// Mul mengalikan dua nilai desimal.
func (d Decimal) Mul(o Decimal) Decimal { return Decimal{d: d.d.Mul(o.d)} }

// Div membagi dua nilai desimal.
func (d Decimal) Div(o Decimal) Decimal { return Decimal{d: d.d.Div(o.d)} }

// MulInt mengalikan dengan bilangan bulat.
func (d Decimal) MulInt(n int64) Decimal {
	return Decimal{d: d.d.Mul(decimal.NewFromInt(n))}
}

// DivInt membagi dengan bilangan bulat.
func (d Decimal) DivInt(n int64) Decimal {
	return Decimal{d: d.d.Div(decimal.NewFromInt(n))}
}

// Round membulatkan ke sejumlah angka desimal.
func (d Decimal) Round(places int32) Decimal { return Decimal{d: d.d.Round(places)} }

// Cmp membandingkan dua nilai desimal.
func (d Decimal) Cmp(o Decimal) int { return d.d.Cmp(o.d) }

// IsZero melaporkan apakah nilainya nol.
func (d Decimal) IsZero() bool { return d.d.IsZero() }

// IsNegative melaporkan apakah nilainya negatif.
func (d Decimal) IsNegative() bool { return d.d.IsNegative() }

// IsPositive melaporkan apakah nilainya lebih besar dari nol.
func (d Decimal) IsPositive() bool { return d.d.IsPositive() }

// IsInteger melaporkan apakah nilainya bilangan bulat.
func (d Decimal) IsInteger() bool { return d.d.IsInteger() }

// String mengembalikan teks desimal.
func (d Decimal) String() string { return d.d.String() }

// MarshalJSON mengirim nilai sebagai angka JSON.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(d.d.String()), nil
}

// UnmarshalJSON menerima angka maupun string angka.
func (d *Decimal) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		d.d = decimal.Zero
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	parsed, err := decimal.NewFromString(s)
	if err != nil {
		return fmt.Errorf("nilai desimal tidak valid: %q", s)
	}
	d.d = parsed
	return nil
}

// Value mengimplementasikan driver.Valuer untuk kolom NUMERIC.
func (d Decimal) Value() (driver.Value, error) { return d.d.String(), nil }

// Scan mengimplementasikan sql.Scanner untuk kolom NUMERIC.
func (d *Decimal) Scan(v any) error {
	switch val := v.(type) {
	case nil:
		d.d = decimal.Zero
	case int64:
		d.d = decimal.NewFromInt(val)
	case float64:
		d.d = decimal.NewFromFloat(val)
	case string:
		parsed, err := decimal.NewFromString(val)
		if err != nil {
			return err
		}
		d.d = parsed
	case []byte:
		parsed, err := decimal.NewFromString(string(val))
		if err != nil {
			return err
		}
		d.d = parsed
	case decimal.Decimal:
		d.d = val
	default:
		return fmt.Errorf("nilai desimal tidak valid: tipe %T", v)
	}
	return nil
}
