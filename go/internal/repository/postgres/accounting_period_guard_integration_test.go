//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go_backend/internal/domain"
)

func TestPenulisanJurnalDitolakSaatPeriodeDitutup(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	const (
		year  = 2098
		month = 5
	)

	var actorID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM s_users ORDER BY id LIMIT 1`).Scan(&actorID); err != nil {
		t.Fatalf("tidak ada pengguna untuk aktor uji: %v", err)
	}

	periodRepo := NewSQLAccountingPeriodRepository(pool)
	journalRepo := NewSQLJournalRepository(pool)
	settingsRepo := NewSQLAccountingSettingsRepository(pool)

	period, err := periodRepo.EnsureOpen(ctx, year, month, actorID)
	if err != nil {
		t.Fatalf("EnsureOpen gagal: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM t_journal_entry_lines WHERE journal_entry_id IN
			(SELECT id FROM t_journal_entries WHERE period_year = $1 AND period_month = $2)`, year, month)
		_, _ = pool.Exec(ctx, `DELETE FROM t_journal_entries WHERE period_year = $1 AND period_month = $2`, year, month)
		_, _ = pool.Exec(ctx, `DELETE FROM m_accounting_periods WHERE id = $1`, period.ID)
	})

	mapping, err := settingsRepo.ListMapping(ctx)
	if err != nil {
		t.Fatalf("ListMapping gagal: %v", err)
	}
	inventoryID, ok := mapping["INVENTORY"]
	if !ok {
		t.Fatal("pemetaan INVENTORY belum ada")
	}
	apID, ok := mapping["AP"]
	if !ok {
		t.Fatal("pemetaan AP belum ada")
	}

	newEntry := func(suffix string, status string) *domain.JournalEntry {
		return &domain.JournalEntry{
			EntryNumber: fmt.Sprintf("JV-UJI-GUARD-%s-%d", suffix, time.Now().UnixNano()),
			EntryDate:   time.Date(year, time.May, 15, 0, 0, 0, 0, time.UTC),
			Description: "uji guard periode",
			SourceType:  domain.JournalSourceManual,
			Status:      status,
			TotalDebit:  100,
			TotalCredit: 100,
			PeriodYear:  year,
			PeriodMonth: month,
			Lines: []domain.JournalLine{
				{LineNo: 1, AccountID: inventoryID, Debit: 100},
				{LineNo: 2, AccountID: apID, Credit: 100},
			},
		}
	}

	if err := journalRepo.CreatePosted(ctx, newEntry("awal", domain.JournalStatusPosted)); err != nil {
		t.Fatalf("CreatePosted saat periode OPEN gagal: %v", err)
	}

	rows, err := periodRepo.Close(ctx, year, month, actorID)
	if err != nil {
		t.Fatalf("Close periode gagal: %v", err)
	}
	if rows != 1 {
		t.Fatalf("periode harus tertutup satu baris, dapat %d", rows)
	}

	if err := journalRepo.CreatePosted(ctx, newEntry("posted", domain.JournalStatusPosted)); !errors.Is(err, domain.ErrJournalPeriodClosed) {
		t.Fatalf("CreatePosted periode CLOSED harus ErrJournalPeriodClosed, dapat %v", err)
	}

	draft := newEntry("draft", domain.JournalStatusDraft)
	if err := journalRepo.CreateDraft(ctx, draft); err != nil {
		t.Fatalf("CreateDraft gagal: %v", err)
	}
	if _, err := journalRepo.Post(ctx, draft.ID, actorID); !errors.Is(err, domain.ErrJournalPeriodClosed) {
		t.Fatalf("Post periode CLOSED harus ErrJournalPeriodClosed, dapat %v", err)
	}

	rows, err = periodRepo.Reopen(ctx, year, month)
	if err != nil {
		t.Fatalf("Reopen periode gagal: %v", err)
	}
	if rows != 1 {
		t.Fatalf("periode harus terbuka satu baris, dapat %d", rows)
	}
	if err := journalRepo.CreatePosted(ctx, newEntry("reopen", domain.JournalStatusPosted)); err != nil {
		t.Fatalf("CreatePosted setelah Reopen harus sukses: %v", err)
	}
}
