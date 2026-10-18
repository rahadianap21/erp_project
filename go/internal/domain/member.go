package domain

import (
	"context"
	"time"
)

// =========================================================================
// KONSTANTA POIN
// =========================================================================

const (
	// MemberPointValueRupiah: nilai tukar 1 poin saat ditukar = Rp 100.
	MemberPointValueRupiah = 100
	// MemberPointEarnPerRupiah: 1 poin diperoleh tiap Rp 10.000 total nota akhir.
	MemberPointEarnPerRupiah = 10000
	// MemberRedeemMaxRatio: tukar poin maksimum 50% dari total nota.
	MemberRedeemMaxRatio = 0.5

	MemberPointEarn           = "EARN"
	MemberPointRedeem         = "REDEEM"
	MemberPointEarnReversal   = "EARN_REVERSAL"
	MemberPointRedeemReversal = "REDEEM_REVERSAL"
	MemberPointAdjust         = "ADJUST"
)

// =========================================================================
// MASTER MEMBER
// =========================================================================

type Member struct {
	ID              int64     `json:"id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	Phone           string    `json:"phone"`
	Email           string    `json:"email"`
	Address         string    `json:"address"`
	DefaultTermDays int       `json:"default_term_days"`
	Points          int64     `json:"points"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
}

type MemberInput struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Phone           string `json:"phone"`
	Email           string `json:"email"`
	Address         string `json:"address"`
	DefaultTermDays int    `json:"default_term_days"`
	IsActive        bool   `json:"is_active"`
}

// MemberPointEntry adalah satu baris buku besar poin. Points bertanda positif
// untuk penambahan (earn / redeem reversal) dan negatif untuk pengurangan.
type MemberPointEntry struct {
	ID           int64     `json:"id"`
	MemberID     int64     `json:"member_id"`
	SaleID       *int64    `json:"sale_id,omitempty"`
	SaleNumber   string    `json:"sale_number,omitempty"`
	EntryType    string    `json:"entry_type"`
	Points       int64     `json:"points"`
	BalanceAfter int64     `json:"balance_after"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
}

type MemberRepository interface {
	Create(ctx context.Context, in MemberInput, actorID int64) (*Member, error)
	Update(ctx context.Context, id int64, in MemberInput, actorID int64) error
	Delete(ctx context.Context, id, actorID int64) error
	GetByID(ctx context.Context, id int64) (*Member, error)
	GetListCursor(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]Member, int64, error)
	LookupForPOS(ctx context.Context, search string, limit int) ([]Member, error)
	ListPointEntries(ctx context.Context, memberID int64, limit int) ([]MemberPointEntry, error)
}

type MemberUsecase interface {
	Create(ctx context.Context, in MemberInput, actorID int64) (*Member, error)
	Update(ctx context.Context, id int64, in MemberInput, actorID int64) error
	Delete(ctx context.Context, id, actorID int64) error
	GetByID(ctx context.Context, id int64) (*Member, error)
	GetAll(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]Member, int64, error)
	LookupForPOS(ctx context.Context, search string, limit int) ([]Member, error)
	ListPointEntries(ctx context.Context, memberID int64, limit int) ([]MemberPointEntry, error)
}
