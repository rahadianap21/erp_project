package domain

import (
	"encoding/json"
	"testing"
)

// TestDateOnlyUnmarshal memastikan tanggal dari <input type="date"> (YYYY-MM-DD)
// dan RFC3339 sama-sama diterima, sementara nilai ngawur ditolak.
func TestDateOnlyUnmarshal(t *testing.T) {
	kasus := []struct {
		nama     string
		masuk    string
		want     string
		inginErr bool
	}{
		{"date only", `"2026-10-04"`, "2026-10-04", false},
		{"RFC3339", `"2026-10-04T15:30:00Z"`, "2026-10-04", false},
		{"kosong", `""`, "", false},
		{"null", `null`, "", false},
		{"ngawur", `"kemarin"`, "", true},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			var d DateOnly
			err := json.Unmarshal([]byte(k.masuk), &d)
			if k.inginErr {
				if err == nil {
					t.Fatalf("input %s seharusnya ditolak", k.masuk)
				}
				return
			}
			if err != nil {
				t.Fatalf("input %s tidak boleh gagal: %v", k.masuk, err)
			}
			if k.want == "" {
				if !d.IsZero() {
					t.Fatalf("input %s harus menghasilkan waktu nol, dapat %v", k.masuk, d.Time)
				}
				return
			}
			if got := d.Format("2006-01-02"); got != k.want {
				t.Fatalf("input %s harus %s, dapat %s", k.masuk, k.want, got)
			}
		})
	}
}
