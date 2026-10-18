package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"

	"github.com/golang-jwt/jwt/v5"
)

// stubUserUsecase hanya mengimplementasikan metode yang dibutuhkan test di
// bawah. Metode lain tidak pernah dipanggil oleh test ini.
type stubUserUsecase struct {
	domain.UserUsecase

	createErr error
	updateErr error
	// actor yang diterima oleh UpdateUser, dipakai untuk memastikan handler
	// benar-benar meneruskan identitas pelaku dari JWT.
	gotActor domain.Actor
}

func (s *stubUserUsecase) CreateUser(ctx context.Context, actor domain.Actor, name, email, password, roleName string) error {
	s.gotActor = actor
	return s.createErr
}

func (s *stubUserUsecase) UpdateUser(ctx context.Context, actor domain.Actor, id int64, name, email, roleName string) error {
	s.gotActor = actor
	return s.updateErr
}

func claimsFor(permissions ...string) jwt.MapClaims {
	asInterfaces := make([]any, len(permissions))
	for i, p := range permissions {
		asInterfaces[i] = p
	}
	return jwt.MapClaims{
		"user_id":     float64(26),
		"permissions": asInterfaces,
	}
}

// withAuth menyuntikkan requestAuth ke context, meniru hasil JWTMiddlewareWithResolver.
func withAuth(r *http.Request, claims jwt.MapClaims) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserContextKey, claims)
	var perms []string
	if raw, ok := claims["permissions"].([]any); ok {
		for _, p := range raw {
			if s, ok := p.(string); ok {
				perms = append(perms, s)
			}
		}
	}
	return middleware.WithRequestAuth(r.WithContext(ctx), middleware.RequestAuth{UserID: 26, Permissions: perms})
}

func postJSON(t *testing.T, uc domain.UserUsecase, claims jwt.MapClaims, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(body))
	r = withAuth(r, claims)
	w := httptest.NewRecorder()
	(&UserHandler{UserUsecase: uc}).UsersCRUDHandler(w, r)
	return w
}

func putJSON(t *testing.T, uc domain.UserUsecase, claims jwt.MapClaims, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPut, "/api/users?id=26", strings.NewReader(body))
	r = withAuth(r, claims)
	w := httptest.NewRecorder()
	(&UserHandler{UserUsecase: uc}).UsersCRUDHandler(w, r)
	return w
}

// TestPenolakanWewenangCreateUserHarus403 guarding regression yang nyata:
// sebelum AuthorizationError ada, penolakan privilege escalation pada CreateUser
// keluar sebagai 400 dan pada UpdateUser sebagai 500. Keduanya salah: percobaan
// eskalasi yang sudah ditolak tidak boleh tercatat sebagai kegagalan server.
func TestPenolakanWewenangCreateUserHarus403(t *testing.T) {
	uc := &stubUserUsecase{
		createErr: domain.NewAuthorizationError("Anda tidak punya hak %s", domain.PermissionAssignRole),
	}

	w := postJSON(t, uc, claimsFor("users:create"), `{"name":"X","email":"x@y.z","password":"Rahasia123!","role_name":"Super Admin"}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("penolakan wewenang harus 403, dapat %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("respons bukan JSON valid: %v", err)
	}
	if !strings.Contains(body["message"], domain.PermissionAssignRole) {
		t.Fatalf("pesan harus menyebut permission yang hilang, dapat %q", body["message"])
	}
}

// TestPenolakanWewenangUpdateUserHarus403 mendokumentasikan the exact bug found during
// live verification: PUT yang ditolak_rules mengembalikan HTTP 500.
func TestPenolakanWewenangUpdateUserHarus403(t *testing.T) {
	uc := &stubUserUsecase{
		updateErr: domain.NewAuthorizationError("tidak boleh memberi role %q", "Super Admin"),
	}

	w := putJSON(t, uc, claimsFor("users:create", "users:update"), `{"name":"X","email":"x@y.z","role_name":"Super Admin"}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("penolakan wewenang harus 403, dapat %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "Gagal memperbarui pengguna") {
		t.Fatal("penolakan wewenang tidak boleh dibungkus prefix kegagalan internal")
	}
}

// TestErrorBukanOtorisasiTetap500 memastikan tipe baru tidak menelan error
// server sungguhan: error biasa harus tetap 500 supaya monitoring tidak buta.
func TestErrorBukanOtorisasiTetap500(t *testing.T) {
	uc := &stubUserUsecase{updateErr: errors.New("koneksi database terputus")}

	w := putJSON(t, uc, claimsFor("users:update"), `{"name":"X","email":"x@y.z","role_name":"Customer"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("error internal harus tetap 500, dapat %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "koneksi database terputus") {
		t.Fatalf("pesan error internal harus diteruskan, dapat %q", w.Body.String())
	}
}

// TestErrorValidasiTetap400 menjaga perilaku lama CreateUser untuk error input.
func TestErrorValidasiTetap400(t *testing.T) {
	uc := &stubUserUsecase{createErr: errors.New("email sudah digunakan")}

	w := postJSON(t, uc, claimsFor("users:create"), `{"name":"X","email":"x@y.z","password":"Rahasia123!"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("error validasi harus 400, dapat %d", w.Code)
	}
}

// TestHandlerMeneruskanActorKeUsecase memastikan wiring middleware.CurrentActor(r) benar:
// usecase harus menerima UserID dari JWT, bukan 0.
func TestHandlerMeneruskanActorKeUsecase(t *testing.T) {
	uc := &stubUserUsecase{}

	postJSON(t, uc, claimsFor("users:create"), `{"name":"X","email":"x@y.z","password":"Rahasia123!"}`)

	if uc.gotActor.UserID != 26 {
		t.Fatalf("UserID aktor harus 26, dapat %d", uc.gotActor.UserID)
	}
}

// TestPelakuNonAktifDipetakan403 menjaga pemetaan status untuk kasus akun yang
// dinonaktifkan sementara token lamanya masih hidup.
//
// Kondisi ini terjadi nyata: token JWT berlaku 24 jam, jadi penonaktifan akun
// tidak langsung mencabut token yang sudah terbit. permissionActor menolak
// pelaku seperti itu dengan AuthorizationError, dan handler harus
// menerjemahkannya jadi 403. Sebelumnya error itu berupa errors.New biasa sehingga
// keluar sebagai 400 "Gagal Menyimpan Data" — keliru karena percobaannya ditolak
// atas dasar wewenang, bukan karena input tidak valid.
func TestPelakuNonAktifDipetakan403(t *testing.T) {
	uc := &stubUserUsecase{
		createErr: domain.NewAuthorizationError("akun pelaku tidak ditemukan atau sudah dinonaktifkan"),
	}

	w := postJSON(t, uc, claimsFor("users:create"),
		`{"name":"X","email":"x@y.z","password":"Rahasia123!","role_name":"Customer"}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("pelaku nonaktif harus 403, dapat %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "Gagal Menyimpan Data") {
		t.Errorf("pesan tidak boleh diperlakukan sebagai kegagalan menyimpan data: %s", w.Body.String())
	}
}

// TestPelakuTidakValidDipetakan403 untuk actor dengan UserID tidak valid
// (misalnya JWT rusak). Perlakuannya harus sama: 403, bukan 400/500.
func TestPelakuTidakValidDipetakan403(t *testing.T) {
	uc := &stubUserUsecase{
		updateErr: domain.NewAuthorizationError("identitas pelaku tidak valid, silakan login ulang"),
	}

	w := putJSON(t, uc, claimsFor("users:update"),
		`{"name":"X","email":"x@y.z","role_name":"Customer"}`)

	if w.Code != http.StatusForbidden {
		t.Fatalf("aktor tidak valid harus 403, dapat %d: %s", w.Code, w.Body.String())
	}
}

// TestValidasiInputUserMenolakDataTidakLengkap membuktikan validate.Struct
// benar-benar dipakai: sebelumnya validator dideklarasikan tapi tidak pernah
// dipanggil, sehingga input divalidasi hanya oleh `if` manual yang mudah
// tertinggal (mis. tidak cek format email maupun panjang password).
func TestValidasiInputUserMenolakDataTidakLengkap(t *testing.T) {
	cases := []struct {
		nama      string
		method    string
		body      string
		wantPesan string
	}{
		{"create tanpa password", http.MethodPost, `{"name":"X","email":"x@y.z"}`, "Password wajib diisi"},
		{"create password pendek", http.MethodPost, `{"name":"X","email":"x@y.z","password":"12345"}`, "Password minimal 6 karakter"},
		{"create email tidak valid", http.MethodPost, `{"name":"X","email":"bukan-email","password":"Rahasia123!"}`, "Format email tidak valid"},
		{"create tanpa nama", http.MethodPost, `{"email":"x@y.z","password":"Rahasia123!"}`, "Nama wajib diisi"},
		{"update tanpa role", http.MethodPut, `{"name":"X","email":"x@y.z"}`, "Hak akses (role) wajib diisi"},
	}

	for _, tc := range cases {
		t.Run(tc.nama, func(t *testing.T) {
			uc := &stubUserUsecase{}
			var w *httptest.ResponseRecorder
			if tc.method == http.MethodPost {
				w = postJSON(t, uc, claimsFor("users:create"), tc.body)
			} else {
				w = putJSON(t, uc, claimsFor("users:update"), tc.body)
			}

			if w.Code != http.StatusBadRequest {
				t.Fatalf("input tidak valid harus 400, dapat %d: %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tc.wantPesan) {
				t.Fatalf("pesan validasi harus memuat %q, dapat %q", tc.wantPesan, w.Body.String())
			}
		})
	}
}
