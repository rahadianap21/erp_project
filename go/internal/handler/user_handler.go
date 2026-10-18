package handler

import (
	"encoding/json"
	"errors"
	"go_backend/internal/domain"
	"go_backend/internal/middleware"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
)

type UserHandler struct {
	UserUsecase domain.UserUsecase
}

// 2. STRUCT RESPONSE YANG DIBUTUHKAN FRONTEND
// SISI BACKEND GO (user_handler.go)
type UserDataResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	RoleName string `json:"role_name"` // 💡 Pastikan ini lowercase "role_name"
}

type GetUsersResponse struct {
	Data        []UserDataResponse `json:"data"`
	Total       int64              `json:"total"`
	RoleOptions []string           `json:"role_options,omitempty"`
	NextCursor  *int64             `json:"next_cursor"`
	PrevCursor  *int64             `json:"prev_cursor"`
	HasNext     bool               `json:"has_next"`
	HasPrev     bool               `json:"has_prev"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

var validate = validator.New()

// createUserRequest adalah payload POST /api/users.
//
// Aturan validasi dipusatkan di sini memakai go-playground/validator agar tidak
// bergantung pada deretan `if` manual yang mudah tertinggal. Panjang minimum
// password (6) sengaja disamakan dengan atribut minLength pada form frontend.
type createUserRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	RoleName string `json:"role_name"`
}

// updateUserRequest adalah payload PUT /api/users. Password tidak diubah lewat
// endpoint ini, jadi tidak ada field password sama sekali.
type updateUserRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	RoleName string `json:"role_name" validate:"required"`
}

// pesanValidasi menerjemahkan error validator ke pesan Indonesia yang ramah,
// dengan menyebut field yang bermasalah.
func pesanValidasi(err error) string {
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) || len(verrs) == 0 {
		return "Data yang dikirim tidak valid"
	}

	fe := verrs[0]
	label := map[string]string{
		"Name":     "Nama",
		"Email":    "Email",
		"Password": "Password",
		"RoleName": "Hak akses (role)",
	}[fe.Field()]
	if label == "" {
		label = fe.Field()
	}

	switch fe.Tag() {
	case "required":
		return label + " wajib diisi"
	case "email":
		return "Format email tidak valid"
	case "min":
		return label + " minimal " + fe.Param() + " karakter"
	default:
		return label + " tidak valid"
	}
}

// Tambahkan Fungsi Login Ini
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 1. Panggil Usecase yang kini sudah melakukan query JOIN antara Role dan Permissions
	token, user, permissions, err := h.UserUsecase.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 2. Kirim token, nama user, dan daftar hak akses dinamis (permissions) ke Next.js
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Login berhasil!",
		"token":       token,
		"name":        user.Name,
		"email":       user.Email,
		"role_id":     user.RoleID,
		"role_name":   user.RoleName,
		"app":         user.RoleApp,
		"permissions": permissions,
	})
}

// Tambahkan fungsi ini di bagian bawah file internal/handler/user_handler.go

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// role_id diambil dari requestAuth yang sudah divalidasi terhadap database,
	// bukan dari klaim JWT. Kalau sebuah role baru saja diubah, profil harus
	// langsung menunjukkan role terbaru — bukan role yang tertanam di token
	// berumur 24 jam.
	auth, ok := middleware.GetRequestAuth(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Sesi otorisasi tidak ditemukan"})
		return
	}

	claims, ok := r.Context().Value(middleware.UserContextKey).(jwt.MapClaims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Gagal memuat konteks pengguna"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Akses disetujui! Anda berada di area sensitif.",
		"user_id": auth.UserID,
		"name":    claims["name"],
		"email":   claims["email"],
		"role_id": auth.RoleID,
	})
}

func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Memanggil layer Usecase (mengikuti pola Clean Architecture Anda)
	err := h.UserUsecase.Logout(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Gagal memproses logout"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logout berhasil!",
	})
}

// GetAllUsers menangani request GET /api/users dengan parameter kueri search, role, cursor, direction, page, dan limit
func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// 1. Ekstrak parameter query string dari URL kueri
	query := r.URL.Query()
	search := query.Get("search")
	role := query.Get("role")

	// VALIDASI FALLBACK: Jika parameter role kosong dari frontend, ubah ke "ALL"
	if role == "" {
		role = "ALL"
	}

	// Konversi 'limit' dengan default nilai 25
	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil || limit < 1 {
		limit = 25
	}

	cursorStr := query.Get("cursor")
	pageStr := query.Get("page")
	page, _ := strconv.Atoi(pageStr)

	// Jika ada cursor atau halaman awal (page <= 1 dan tidak ada page > 1 eksplisit), gunakan cursor pagination
	if cursorStr != "" || page <= 1 {
		direction := query.Get("direction")
		if direction != "prev" {
			direction = "next"
		}

		var cursorID int64 = 0
		if cursorStr != "" {
			cursorID, _ = strconv.ParseInt(cursorStr, 10, 64)
		}

		result, err := h.UserUsecase.GetUsersCursor(r.Context(), domain.UserCursorParams{
			Search:    search,
			Role:      role,
			CursorID:  cursorID,
			Direction: direction,
			Limit:     limit,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Gagal mengambil data pengguna: " + err.Error()})
			return
		}

		var dataResponse []UserDataResponse
		for _, u := range result.Users {
			dataResponse = append(dataResponse, UserDataResponse{
				ID:       u.ID,
				Name:     u.Name,
				Email:    u.Email,
				RoleName: u.RoleName,
			})
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GetUsersResponse{
			Data:        dataResponse,
			Total:       result.Total,
			RoleOptions: result.RoleOptions,
			NextCursor:  result.NextCursor,
			PrevCursor:  result.PrevCursor,
			HasNext:     result.HasNext,
			HasPrev:     result.HasPrev,
		})
		return
	}

	// Fallback ke offset pagination jika klien sengaja mengirim page > 1 tanpa kursor (legacy)
	users, total, roleOptions, err := h.UserUsecase.GetAllUsers(r.Context(), search, role, page, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Gagal mengambil data pengguna: " + err.Error()})
		return
	}

	var dataResponse []UserDataResponse
	for _, u := range users {
		dataResponse = append(dataResponse, UserDataResponse{
			ID:       u.ID,
			Name:     u.Name,
			Email:    u.Email,
			RoleName: u.RoleName,
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GetUsersResponse{
		Data:        dataResponse,
		Total:       total,
		RoleOptions: roleOptions,
	})
}

// UsersCRUDHandler menangani operasi CRUD untuk endpoint /api/users
func (h *UserHandler) UsersCRUDHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		middleware.RequirePermission("users:read", func(w http.ResponseWriter, r *http.Request) {
			h.GetAllUsers(w, r)
		})(w, r)

	case http.MethodPost:
		middleware.RequirePermission("users:create", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			var req createUserRequest

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"message": "Format request tidak valid: " + err.Error()})
				return
			}

			if err := validate.Struct(req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"message": pesanValidasi(err)})
				return
			}

			// Panggil layer usecase CreateUser yang menangani pemilihan role dan hashing password
			if err := h.UserUsecase.CreateUser(r.Context(), middleware.CurrentActor(r), req.Name, req.Email, req.Password, req.RoleName); err != nil {
				var authErr *domain.AuthorizationError
				if errors.As(err, &authErr) {
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]string{"message": authErr.Reason})
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"message": "Gagal Menyimpan Data: " + err.Error()})
				return
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "success",
				"message": "User baru berhasil didaftarkan ke sistem",
			})
		})(w, r)

	case http.MethodPut:
		middleware.RequirePermission("users:update", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"message": "ID pengguna wajib disertakan"})
				return
			}

			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"message": "Format ID pengguna tidak valid"})
				return
			}

			var req updateUserRequest

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"message": "Format request tidak valid: " + err.Error()})
				return
			}

			if err := validate.Struct(req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"message": pesanValidasi(err)})
				return
			}

			if err := h.UserUsecase.UpdateUser(r.Context(), middleware.CurrentActor(r), id, req.Name, req.Email, req.RoleName); err != nil {
				var authErr *domain.AuthorizationError
				if errors.As(err, &authErr) {
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]string{"message": authErr.Reason})
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"message": "Gagal memperbarui pengguna: " + err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "success",
				"message": "Data pengguna berhasil diperbarui",
			})
		})(w, r)

	case http.MethodDelete:
		middleware.RequirePermission("users:delete", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"message": "ID user wajib disertakan"})
				return
			}

			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"message": "Format ID user tidak valid"})
				return
			}

			if err := h.UserUsecase.DeleteUser(r.Context(), id); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"message": "Gagal menghapus user: " + err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "success",
				"message": "User berhasil dihapus dari sistem",
			})
		})(w, r)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Metode HTTP tidak diizinkan"})
	}
}
