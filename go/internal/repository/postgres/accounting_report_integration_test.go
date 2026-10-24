//go:build integration

package postgres

// Test integrasi laporan keuangan (Fase 2C).
//
// Membuktikan bahwa kueri laporan membaca jurnal berstatus POSTED dengan benar
// terhadap PostgreSQL sungguhan, mencakup Neraca Saldo, Neraca, Laba Rugi, dan
// Buku Besar. Jurnal uji dibuat langsung lewat CreatePosted pada rentang
// tanggal 2099 agar tidak bercampur dengan data operasional, lalu dihapus di
// t.Cleanup. Butuh PostgreSQL nyata; dilewati bila DATABASE_URL kosong.

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/usecase"
)

// cariAkunLaporan mencari baris akun berdasarkan id di dalam hasil laporan.
func cariAkunLaporan(accounts []domain.ReportAccount, id int64) *domain.ReportAccount {
	for i := range accounts {
		if accounts[i].AccountID == id {
			return &accounts[i]
		}
	}
	return nil
}

func TestLaporanKeuanganMembacaJurnalPosted(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	settingsRepo := NewSQLAccountingSettingsRepository(pool)
	mapping, err := settingsRepo.ListMapping(ctx)
	if err != nil {
		t.Fatalf("ListMapping gagal: %v", err)
	}
	inventoryID, ok := mapping["INVENTORY"]
	if !ok {
		t.Fatal("pemetaan INVENTORY belum ada di s_accounting_settings")
	}
	apID, ok := mapping["AP"]
	if !ok {
		t.Fatal("pemetaan AP belum ada di s_accounting_settings")
	}

	journalRepo := NewSQLJournalRepository(pool)
	coaRepo := NewSQLChartOfAccountRepository(pool)
	periodRepo := NewSQLAccountingPeriodRepository(pool)
	reportRepo := NewSQLAccountingReportRepository(pool)
	reportUC := usecase.NewAccountingReportUsecase(reportRepo, coaRepo)

	if _, err := periodRepo.EnsureOpen(ctx, 2099, 6, 0); err != nil {
		t.Fatalf("EnsureOpen periode uji gagal: %v", err)
	}

	entryNumber := fmt.Sprintf("JV-UJI-LAPORAN-%d", time.Now().UnixNano())
	entryDate := time.Date(2099, time.June, 15, 0, 0, 0, 0, time.UTC)
	entry := &domain.JournalEntry{
		EntryNumber: entryNumber,
		EntryDate:   entryDate,
		Description: "uji laporan keuangan",
		SourceType:  "MANUAL",
		Status:      domain.JournalStatusPosted,
		TotalDebit:  7000,
		TotalCredit: 7000,
		PeriodYear:  2099,
		PeriodMonth: 6,
		Lines: []domain.JournalLine{
			{LineNo: 1, AccountID: inventoryID, Debit: 7000, Description: "uji persediaan"},
			{LineNo: 2, AccountID: apID, Credit: 7000, Description: "uji hutang"},
		},
	}
	if err := journalRepo.CreatePosted(ctx, entry); err != nil {
		t.Fatalf("CreatePosted gagal: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM t_journal_entry_lines WHERE journal_entry_id = $1`, entry.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM t_journal_entries WHERE id = $1`, entry.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM m_accounting_periods WHERE period_year = 2099 AND period_month = 6`)
	})

	from := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2099, time.December, 31, 0, 0, 0, 0, time.UTC)

	// 1. Neraca Saldo: seimbang dan memuat kolom debit persediaan serta kredit
	// hutang dari jurnal yang baru dibuat.
	tb, err := reportUC.TrialBalance(ctx, from, to)
	if err != nil {
		t.Fatalf("TrialBalance gagal: %v", err)
	}
	if !tb.Balanced {
		t.Errorf("neraca saldo harus seimbang, debit=%v kredit=%v", tb.TotalDebit, tb.TotalCredit)
	}
	invRow := cariAkunLaporan(tb.Accounts, inventoryID)
	if invRow == nil || invRow.Debit < 7000 {
		t.Fatalf("baris persediaan harus memuat debit >= 7000, dapat %+v", invRow)
	}
	apRow := cariAkunLaporan(tb.Accounts, apID)
	if apRow == nil || apRow.Credit < 7000 {
		t.Fatalf("baris hutang usaha harus memuat kredit >= 7000, dapat %+v", apRow)
	}

	// 2. Neraca: persamaan A = K + E harus tetap terpenuhi.
	sheet, err := reportUC.BalanceSheet(ctx, to)
	if err != nil {
		t.Fatalf("BalanceSheet gagal: %v", err)
	}
	if !sheet.Balanced {
		t.Errorf("neraca harus seimbang, aset=%v kewajiban+ekuitas=%v",
			sheet.TotalAssets, sheet.TotalLiabilitiesAndEquity)
	}
	if cariAkunLaporan(sheet.Assets, inventoryID) == nil {
		t.Error("aset harus memuat akun persediaan")
	}

	// 3. Laba Rugi: jurnal ini hanya menyentuh akun neraca, jadi laba nol.
	statement, err := reportUC.IncomeStatement(ctx, from, to)
	if err != nil {
		t.Fatalf("IncomeStatement gagal: %v", err)
	}
	if statement.NetIncome != 0 {
		t.Errorf("laba bersih harus 0 (tanpa akun pendapatan/beban), dapat %v", statement.NetIncome)
	}

	// 4. Buku Besar persediaan: saldo berjalan naik tepat sebesar debit jurnal.
	ledger, err := reportUC.GeneralLedger(ctx, inventoryID, from, to)
	if err != nil {
		t.Fatalf("GeneralLedger gagal: %v", err)
	}
	if math.Abs((ledger.ClosingBalance-ledger.OpeningBalance)-7000) > 0.001 {
		t.Errorf("kenaikan saldo persediaan harus 7000, dapat %v",
			ledger.ClosingBalance-ledger.OpeningBalance)
	}
	if ledger.TotalDebit-ledger.TotalCredit < 7000 {
		t.Errorf("mutasi debit persediaan harus >= 7000, dapat %v", ledger.TotalDebit-ledger.TotalCredit)
	}
}
