package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"go_backend/internal/domain"
)

type fakeIdempotencyRecord struct {
	hash        string
	completed   bool
	status      int
	body        []byte
	contentType string
}

type fakeIdempotencyStore struct {
	mu     sync.Mutex
	nextID int64
	rows   map[string]*fakeIdempotencyRecord
}

func newFakeIdempotencyStore() *fakeIdempotencyStore {
	return &fakeIdempotencyStore{rows: make(map[string]*fakeIdempotencyRecord)}
}

func (s *fakeIdempotencyStore) key(userID int64, scope, key string) string {
	return scope + "|" + key
}

func (s *fakeIdempotencyStore) Claim(_ context.Context, userID int64, scope, key, requestHash string) (*domain.IdempotencyRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := s.key(userID, scope, key)
	row, ok := s.rows[k]
	if !ok {
		s.nextID++
		row = &fakeIdempotencyRecord{hash: requestHash}
		s.rows[k] = row
		return &domain.IdempotencyRecord{ID: s.nextID, RequestHash: requestHash}, nil
	}
	if row.hash != requestHash {
		return nil, domain.ErrIdempotencyKeyReused
	}
	if !row.completed {
		return nil, domain.ErrIdempotencyInProgress
	}
	return &domain.IdempotencyRecord{
		ID:                  s.nextID,
		RequestHash:         row.hash,
		Completed:           true,
		ResponseStatus:      row.status,
		ResponseBody:        row.body,
		ResponseContentType: row.contentType,
	}, nil
}

func (s *fakeIdempotencyStore) Complete(_ context.Context, id int64, status int, body []byte, contentType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, row := range s.rows {
		if !row.completed {
			row.completed = true
			row.status = status
			row.body = append([]byte(nil), body...)
			row.contentType = contentType
			return nil
		}
	}
	return nil
}

func (s *fakeIdempotencyStore) Release(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, row := range s.rows {
		if !row.completed {
			delete(s.rows, k)
			return nil
		}
	}
	return nil
}

func idempotencyRequest(method, target, body, key string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
	}
	r = WithRequestAuth(r, RequestAuth{UserID: 7})
	if key != "" {
		r.Header.Set(IdempotencyKeyHeader, key)
	}
	return r
}

func TestIdempotencyMewajibkanHeaderPadaMutasi(t *testing.T) {
	store := newFakeIdempotencyStore()
	called := 0
	h := Idempotency(store, true, func(w http.ResponseWriter, _ *http.Request) {
		called++
		w.WriteHeader(http.StatusCreated)
	})

	w := httptest.NewRecorder()
	h(w, idempotencyRequest(http.MethodPost, "/api/sales", `{"x":1}`, ""))

	if w.Code != http.StatusPreconditionRequired {
		t.Fatalf("tanpa key harus 428, dapat %d", w.Code)
	}
	if called != 0 {
		t.Fatalf("handler tidak boleh dipanggil, dipanggil %d kali", called)
	}
}

func TestIdempotencyGetTidakButuhKey(t *testing.T) {
	store := newFakeIdempotencyStore()
	called := 0
	h := Idempotency(store, true, func(w http.ResponseWriter, _ *http.Request) {
		called++
		_, _ = w.Write([]byte("ok"))
	})

	w := httptest.NewRecorder()
	h(w, idempotencyRequest(http.MethodGet, "/api/sales", "", ""))

	if w.Code != http.StatusOK || w.Body.String() != "ok" {
		t.Fatalf("GET tanpa key harus lolos, dapat %d %q", w.Code, w.Body.String())
	}
	if called != 1 {
		t.Fatalf("handler harus dipanggil sekali, dapat %d", called)
	}
}

func TestIdempotencyMemutarUlangResponsSukses(t *testing.T) {
	store := newFakeIdempotencyStore()
	called := 0
	h := Idempotency(store, true, func(w http.ResponseWriter, _ *http.Request) {
		called++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":1}}`))
	})

	first := httptest.NewRecorder()
	h(first, idempotencyRequest(http.MethodPost, "/api/sales", `{"product_id":1}`, "key-1"))

	second := httptest.NewRecorder()
	h(second, idempotencyRequest(http.MethodPost, "/api/sales", `{"product_id":1}`, "key-1"))

	if called != 1 {
		t.Fatalf("handler hanya boleh sekali, dapat %d", called)
	}
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("status replay harus 201, dapat %d dan %d", first.Code, second.Code)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("body replay harus sama: %q vs %q", first.Body.String(), second.Body.String())
	}
	if second.Header().Get(IdempotencyReplayedHeader) != "true" {
		t.Fatal("respons replay harus ditandai Idempotency-Replayed")
	}
	if second.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content-type replay harus dipulihkan, dapat %q", second.Header().Get("Content-Type"))
	}
}

func TestIdempotencyMenolakKeyDenganPayloadBerbeda(t *testing.T) {
	store := newFakeIdempotencyStore()
	called := 0
	h := Idempotency(store, true, func(w http.ResponseWriter, _ *http.Request) {
		called++
		w.WriteHeader(http.StatusCreated)
	})

	first := httptest.NewRecorder()
	h(first, idempotencyRequest(http.MethodPost, "/api/sales", `{"product_id":1}`, "key-1"))

	second := httptest.NewRecorder()
	h(second, idempotencyRequest(http.MethodPost, "/api/sales", `{"product_id":2}`, "key-1"))

	if called != 1 {
		t.Fatalf("handler hanya boleh sekali, dapat %d", called)
	}
	if second.Code != http.StatusConflict {
		t.Fatalf("payload berbeda harus 409, dapat %d", second.Code)
	}
}

func TestIdempotencyMelepasKlaimSaatGagal(t *testing.T) {
	store := newFakeIdempotencyStore()
	called := 0
	h := Idempotency(store, true, func(w http.ResponseWriter, _ *http.Request) {
		called++
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"ditolak"}`))
	})

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		h(w, idempotencyRequest(http.MethodPost, "/api/sales", `{"product_id":1}`, "key-1"))
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("percobaan %d harus 422, dapat %d", i+1, w.Code)
		}
	}
	if called != 2 {
		t.Fatalf("klaim gagal harus dilepas sehingga handler dipanggil dua kali, dapat %d", called)
	}
}

func TestIdempotencyMenolakTanpaAutentikasi(t *testing.T) {
	store := newFakeIdempotencyStore()
	called := 0
	h := Idempotency(store, true, func(w http.ResponseWriter, _ *http.Request) {
		called++
	})

	r := httptest.NewRequest(http.MethodPost, "/api/sales", strings.NewReader(`{}`))
	r.Header.Set(IdempotencyKeyHeader, "key-1")
	w := httptest.NewRecorder()
	h(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa auth harus 401, dapat %d", w.Code)
	}
	if called != 0 {
		t.Fatalf("handler tidak boleh dipanggil, dipanggil %d kali", called)
	}
}
