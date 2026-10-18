package usecase

// Pengujian generator slug kategori produk.
//
// Bug yang dicegah test ini: regex lama `[^a-z0-0\s-_]` memakai rentang `0-0`
// yang hanya cocok pada digit 0, sehingga digit 1-9 ikut terhapus oleh
// ReplaceAllString. Akibatnya "Kopi Robusta 2024" menjadi "kopi-robusta-",
// dan jika angka menghasilkan strip di akhir, slug dua kategori berbeda bisa
// jadi identik lalu menabrak unique constraint.
//
// Regex benar: `[^a-z0-9\s-_]` — sisa karakter dihapus, digit tetap dipertahankan.

import "testing"

func TestGenerateSlug(t *testing.T) {
	uc := &productCategoryUsecase{}

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"digit harus bertahan", "Kopi Robusta 2024", "kopi-robusta-2024"},
		{"digit di awal", "Grosir 5kg", "grosir-5kg"},
		{"digit dan tanda di tengah", "Aroma #1 Kopi", "aroma-1-kopi"},
		{"ampersand dan spasi ganda", "Sirup & Es", "sirup-es"},
		{"trim dan buang strip", "--Tanda--Strip--", "tanda-strip"},
		{"huruf kapital", "SPASI", "spasi"},
		{"tanpa spasi", "TanpaSpasi", "tanpaspasi"},
		{"kosong", "", ""},
		{"hanya angka", "2024", "2024"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := uc.generateSlug(tc.input); got != tc.want {
				t.Errorf("generateSlug(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
