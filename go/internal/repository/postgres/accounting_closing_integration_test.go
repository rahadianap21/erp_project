package postgres

// Test integrasi jurnal penutup (Fase 2D).
//
// Membuktikan terhadap PostgreSQL sungguhan bahwa:
//
//  1. Menutup periode menerbitkan jurnal POSTED ber-source_type 'CLOSING'
//     dengan source_id = id periode.
//  2. Jurnal penutup menihilkan seluruh akun PENDAPATAN dan BEBAN kumulatif,
//     dan memindahkan laba/rugi bersih ke Laba Ditahan (3-1300).
//  3. Membuka kembali periode membalik jurnal penutup sehingga pembukuan
//     kembali seperti semula; periode yang sama kemudian bisa ditutup ulang
//     (membuktikan unique index mengabaikan jurnal yang sudah dibalik).
//
// Test memakai periode tahun 2097 bulan 7 yang tidak dipakai operasional, dan
// seluruh jejaknya dihapus di t.Cleanup. Butuh PostgreSQL nyata; dilewati bila
// DATABASE_URL kosong.

import (
	"context"
	"testing"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/usecase"

	"github.com/jackc/pgx/v5/pgxpool"
)

// saldoTemporer menjumlahkan saldo bertanda seluruh akun PENDAPATAN, BEBAN, dan
// Laba Ditahan (3-1300) dari jurnal POSTED pada rentang tanggal.
func saldoTemporer(ctx context.Context, t *testing.T, pool *pgxpool.Pool, from, to time.Time) (revenue, expense, retained float64) {
	t.Helper()
	if err := pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN a.account_type = 'REVENUE' THEN l.credit - l.debit END), 0),
			COALESCE(SUM(CASE WHEN a.account_type = 'EXPENSE' THEN l.debit - l.credit END), 0),
			COALESCE(SUM(CASE WHEN a.code = '3-1300' THEN l.credit - l.debit END), 0)
		FROM t_journal_entry_lines l
		JOIN t_journal_entries j ON j.id = l.journal_entry_id
		JOIN m_chart_of_accounts a ON a.id = l.account_id
		WHERE j.status = 'POSTED' AND j.entry_date >= $1 AND j.entry_date <= $2`,
		from, to).Scan(&revenue, &expense, &retained); err != nil {
		t.Fatalf("gagal menghitung saldo temporer: %v", err)
	}
	return
}

func TestJurnalPenutupMenutupDanMembukaKembaliPeriode(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	const (
		year  = 2097
		month = 7
	)

	lookupAccount := func(code string) int64 {
		t.Helper()
		var id int64
		if err := pool.QueryRow(ctx,
			`SELECT id FROM m_chart_of_accounts WHERE code = $1 AND deleted_at IS NULL`, code).Scan(&id); err != nil {
			t.Fatalf("akun %s tidak ditemukan: %v", code, err)
		}
		return id
	}
	revenueID := lookupAccount("4-1100")
	expenseID := lookupAccount("5-2100")
	cashID := lookupAccount("1-1101")
	retainedID := lookupAccount("3-1300")

	journalRepo := NewSQLJournalRepository(pool)
	coaRepo := NewSQLChartOfAccountRepository(pool)
	periodRepo := NewSQLAccountingPeriodRepository(pool)
	settingsRepo := NewSQLAccountingSettingsRepository(pool)
	reportRepo := NewSQLAccountingReportRepository(pool)

	journalUC := usecase.NewJournalUsecase(journalRepo, coaRepo, periodRepo, settingsRepo, reportRepo)
	periodUC := usecase.NewAccountingPeriodUsecase(periodRepo, journalUC)

	// Penutupan periode mencatat closed_by ke s_users, jadi butuh pengguna nyata.
	var actorID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM s_users ORDER BY id LIMIT 1`).Scan(&actorID); err != nil {
		t.Fatalf("tidak ada pengguna untuk aktor uji: %v", err)
	}

	// Periode uji harus ada lebih dulu.
	period, err := periodRepo.EnsureOpen(ctx, year, month, actorID)
	if err != nil {
		t.Fatalf("EnsureOpen gagal: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `UPDATE t_journal_entries SET reversal_of_id = NULL, reversed_by_id = NULL
			WHERE period_year = $1 AND period_month = $2`, year, month)
		_, _ = pool.Exec(ctx, `DELETE FROM t_journal_entry_lines WHERE journal_entry_id IN
			(SELECT id FROM t_journal_entries WHERE period_year = $1 AND period_month = $2)`, year, month)
		_, _ = pool.Exec(ctx, `DELETE FROM t_journal_entries WHERE period_year = $1 AND period_month = $2`, year, month)
		_, _ = pool.Exec(ctx, `DELETE FROM m_accounting_periods WHERE period_year = $1 AND period_month = $2`, year, month)
	})

	// Jurnal operasional: Pendapatan 1000, Beban 400, Kas 600 => laba 600.
	manual := &domain.JournalEntry{
		EntryNumber: "JV-UJI-PENUTUP-1",
		EntryDate:   time.Date(year, time.July, 15, 0, 0, 0, 0, time.UTC),
		Description: "uji jurnal penutup",
		SourceType:  domain.JournalSourceManual,
		Status:      domain.JournalStatusPosted,
		TotalDebit:  1000,
		TotalCredit: 1000,
		PeriodYear:  year,
		PeriodMonth: month,
		Lines: []domain.JournalLine{
			{LineNo: 1, AccountID: cashID, Debit: 600},
			{LineNo: 2, AccountID: expenseID, Debit: 400},
			{LineNo: 3, AccountID: revenueID, Credit: 1000},
		},
	}
	if err := journalRepo.CreatePosted(ctx, manual); err != nil {
		t.Fatalf("CreatePosted jurnal uji gagal: %v", err)
	}

	periodEnd := time.Date(year, time.August, 0, 0, 0, 0, 0, time.UTC)
	epoch := time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)

	revenueBefore, expenseBefore, retainedBefore := saldoTemporer(ctx, t, pool, epoch, periodEnd)
	netBefore := revenueBefore - expenseBefore

	// 1. Tutup periode.
	if err := periodUC.ClosePeriod(ctx, year, month, actorID); err != nil {
		t.Fatalf("ClosePeriod gagal: %v", err)
	}

	var closingStatus string
	var closingSourceID int64
	if err := pool.QueryRow(ctx, `
		SELECT status, source_id FROM t_journal_entries
		WHERE source_type = 'CLOSING' AND source_id = $1`, period.ID).
		Scan(&closingStatus, &closingSourceID); err != nil {
		t.Fatalf("jurnal penutup tidak ditemukan: %v", err)
	}
	if closingStatus != "POSTED" {
		t.Fatalf("jurnal penutup harus POSTED, dapat %q", closingStatus)
	}

	// 2. Akun temporer harus nol; Laba Ditahan naik sebesar laba bersih.
	revenueAfter, expenseAfter, retainedAfter := saldoTemporer(ctx, t, pool, epoch, periodEnd)
	if revenueAfter != 0 {
		t.Errorf("saldo pendapatan harus nol setelah penutupan, dapat %v", revenueAfter)
	}
	if expenseAfter != 0 {
		t.Errorf("saldo beban harus nol setelah penutupan, dapat %v", expenseAfter)
	}
	if diff := retainedAfter - retainedBefore; diff != netBefore {
		t.Errorf("kenaikan Laba Ditahan = %v, ingin %v", diff, netBefore)
	}

	var periodStatus string
	if err := pool.QueryRow(ctx,
		`SELECT status FROM m_accounting_periods WHERE period_year = $1 AND period_month = $2`, year, month).
		Scan(&periodStatus); err != nil {
		t.Fatalf("gagal membaca status periode: %v", err)
	}
	if periodStatus != "CLOSED" {
		t.Fatalf("periode harus CLOSED, dapat %q", periodStatus)
	}

	// 3. Buka kembali: jurnal penutup dibalik, saldo temporer kembali.
	if err := periodUC.ReopenPeriod(ctx, year, month, actorID); err != nil {
		t.Fatalf("ReopenPeriod gagal: %v", err)
	}
	var unreversed int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM t_journal_entries
		WHERE source_type = 'CLOSING' AND source_id = $1 AND reversed_by_id IS NULL`, period.ID).
		Scan(&unreversed); err != nil {
		t.Fatalf("gagal menghitung jurnal penutup belum dibalik: %v", err)
	}
	if unreversed != 0 {
		t.Fatalf("setelah buka kembali tidak boleh ada jurnal penutup aktif, dapat %d", unreversed)
	}

	revenueReopen, expenseReopen, _ := saldoTemporer(ctx, t, pool, epoch, periodEnd)
	if revenueReopen != revenueBefore || expenseReopen != expenseBefore {
		t.Errorf("saldo temporer harus kembali seperti semula: pendapatan %v→%v beban %v→%v",
			revenueBefore, revenueReopen, expenseBefore, expenseReopen)
	}

	// 4. Tutup ulang: jurnal penutup baru terbit (membuktikan index membebaskan
	// slot setelah pembalikan).
	if err := periodUC.ClosePeriod(ctx, year, month, actorID); err != nil {
		t.Fatalf("ClosePeriod ulang gagal: %v", err)
	}
	var totalClosing int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM t_journal_entries
		WHERE source_type = 'CLOSING' AND source_id = $1`, period.ID).
		Scan(&totalClosing); err != nil {
		t.Fatalf("gagal menghitung jurnal penutup: %v", err)
	}
	if totalClosing != 2 {
		t.Fatalf("harus ada 2 jurnal penutup (lama + baru), dapat %d", totalClosing)
	}
	revenueFinal, expenseFinal, _ := saldoTemporer(ctx, t, pool, epoch, periodEnd)
	if revenueFinal != 0 || expenseFinal != 0 {
		t.Errorf("setelah tutup ulang saldo temporer harus nol, dapat pendapatan %v beban %v", revenueFinal, expenseFinal)
	}

	// Akun Laba Ditahan dipakai, bukan akun lain.
	var retainedLines int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM t_journal_entry_lines l
		JOIN t_journal_entries j ON j.id = l.journal_entry_id
		WHERE j.source_type = 'CLOSING' AND l.account_id = $1`, retainedID).
		Scan(&retainedLines); err != nil {
		t.Fatalf("gagal menghitung baris Laba Ditahan: %v", err)
	}
	if retainedLines == 0 {
		t.Error("jurnal penutup harus menyentuh akun Laba Ditahan")
	}
}
