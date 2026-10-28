package domain

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

func TestMoneyJSONMengirimAngkaRupiah(t *testing.T) {
	type payload struct {
		Total Money `json:"total"`
	}

	kasus := []struct {
		nama  string
		nilai Money
		want  string
	}{
		{"bulat", Rupiah(1234567), `{"total":1234567}`},
		{"ada sen", NewMoney(123456789), `{"total":1234567.89}`},
		{"nol", Money{}, `{"total":0}`},
		{"negatif", NewMoney(-150), `{"total":-1.5}`},
	}

	for _, tc := range kasus {
		t.Run(tc.nama, func(t *testing.T) {
			b, err := json.Marshal(payload{Total: tc.nilai})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(b) != tc.want {
				t.Fatalf("json = %s, ingin %s", b, tc.want)
			}
		})
	}
}

func TestMoneyUnmarshalMenerimaAngkaDanString(t *testing.T) {
	var angka struct {
		Total Money `json:"total"`
	}
	if err := json.Unmarshal([]byte(`{"total":1234.56}`), &angka); err != nil {
		t.Fatalf("unmarshal angka: %v", err)
	}
	if angka.Total.Cents() != 123456 {
		t.Fatalf("sen = %d, ingin 123456", angka.Total.Cents())
	}

	var teks struct {
		Total Money `json:"total"`
	}
	if err := json.Unmarshal([]byte(`{"total":"1234.56"}`), &teks); err != nil {
		t.Fatalf("unmarshal string: %v", err)
	}
	if teks.Total.Cents() != 123456 {
		t.Fatalf("sen = %d, ingin 123456", teks.Total.Cents())
	}
}

func TestMoneyPembulatanSetengahMenjauhiNol(t *testing.T) {
	if got := MoneyFromDecimal(decimal.RequireFromString("1000.005")).Cents(); got != 100001 {
		t.Fatalf("pembulatan = %d, ingin 100001", got)
	}
	if got := MoneyFromDecimal(decimal.RequireFromString("-1000.005")).Cents(); got != -100001 {
		t.Fatalf("pembulatan negatif = %d, ingin -100001", got)
	}
}

func TestMoneyAritmetikaEksak(t *testing.T) {
	a := MoneyFromFloat(0.1)
	b := MoneyFromFloat(0.2)
	if got := a.Add(b); got.Cents() != 30 {
		t.Fatalf("0,1 + 0,2 = %d sen, ingin 30", got.Cents())
	}

	harga := Rupiah(333)
	total := harga.MulInt(3)
	if total.Cents() != 99900 {
		t.Fatalf("total = %d sen, ingin 99900", total.Cents())
	}
}

func TestMoneyPerkalianTarif(t *testing.T) {
	dpp := Rupiah(12345)
	pajak := dpp.MulDecimal(decimal.RequireFromString("0.11"))
	if pajak.Cents() != 135795 {
		t.Fatalf("pajak = %d sen, ingin 135795", pajak.Cents())
	}
}

func TestDecimalJSONAngka(t *testing.T) {
	type payload struct {
		Rate Decimal `json:"rate"`
	}
	b, err := json.Marshal(payload{Rate: DecFromString2("0.12")})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `{"rate":0.12}` {
		t.Fatalf("json = %s", b)
	}

	// Nilai nol dari struct kosong juga harus tetap berupa angka JSON.
	b, err = json.Marshal(payload{})
	if err != nil {
		t.Fatalf("marshal nol: %v", err)
	}
	if string(b) != `{"rate":0}` {
		t.Fatalf("json nol = %s", b)
	}
}

func TestMoneyScan(t *testing.T) {
	var m Money
	if err := m.Scan("1234.56"); err != nil {
		t.Fatalf("scan string: %v", err)
	}
	if m.Cents() != 123456 {
		t.Fatalf("sen = %d", m.Cents())
	}
	if err := m.Scan(nil); err != nil {
		t.Fatalf("scan nil: %v", err)
	}
	if !m.IsZero() {
		t.Fatalf("nil harus menjadi nol")
	}
}

// DecFromString2 adalah pembantu pengujian agar kasus tidak perlu menangani
// error yang tidak mungkin terjadi pada literal tetap.
func DecFromString2(s string) Decimal {
	d, err := DecFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}
