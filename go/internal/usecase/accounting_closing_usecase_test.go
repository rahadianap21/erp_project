package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"go_backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

// --- Fakes khusus jurnal penutup ---

// fakeClosingManager mencatat pemanggilan jurnal penutup oleh periode fiskal.
type fakeClosingManager struct {
	postedPeriodID   int64
	reversedPeriodID int64
	postErr          error
	reverseErr       error
}

func (f *fakeClosingManager) PostClosingEntryTx(_ context.Context, _ domain.AccountingTx, period *domain.AccountingPeriod, _ int64) error {
	f.postedPeriodID = period.ID
	return f.postErr
}

func (f *fakeClosingManager) ReverseClosingEntryTx(_ context.Context, _ domain.AccountingTx, period *domain.AccountingPeriod, _ int64) error {
	f.reversedPeriodID = period.ID
	return f.reverseErr
}

func closingAccounts() map[int64]domain.ChartOfAccount {
	return map[int64]domain.ChartOfAccount{
		4:  {ID: 4, Code: "4-1100", IsActive: true},
		5:  {ID: 5, Code: "5-2100", IsActive: true},
		31: {ID: 31, Code: "3-1300", IsActive: true},
	}
}

func closingMapping() map[string]int64 {
	return map[string]int64{"RETAINED_EARNINGS": 31}
}

func newClosingUsecase(repo *fakeJournalRepo, balances []domain.AccountBalance, mapping map[string]int64) domain.JournalUsecase {
	return NewJournalUsecase(repo,
		&fakeAccountRepo{accounts: closingAccounts()},
		&fakePeriodRepo{},
		&fakeSettingsRepo{mapping: mapping},
		&fakeReportRepo{balances: balances})
}

func period2026(month int, status string) *domain.AccountingPeriod {
	return &domain.AccountingPeriod{ID: 7, PeriodYear: 2026, PeriodMonth: month, Status: status}
}

// akunDari mencari baris jurnal untuk satu akun pada jurnal yang tercatat.
func akunDari(entry *domain.JournalEntry, accountID int64) (domain.JournalLine, bool) {
	for _, l := range entry.Lines {
		if l.AccountID == accountID {
			return l, true
		}
	}
	return domain.JournalLine{}, false
}

// TestPostClosingEntryLabaMemindahkanKeLabaDitahan: pendapatan 300 dan beban 100
// harus disanatkan ke nol, laba 200 dikreditkan ke Laba Ditahan.
func TestPostClosingEntryLabaMemindahkanKeLabaDitahan(t *testing.T) {
	repo := &fakeJournalRepo{}
	uc := newClosingUsecase(repo, skenarioNeraca(), closingMapping())

	if err := uc.PostClosingEntry(context.Background(), period2026(3, domain.PeriodStatusOpen), 99); err != nil {
		t.Fatalf("jurnal penutup harus terbit: %v", err)
	}
	if repo.created == nil {
		t.Fatal("repository tidak menerima jurnal penutup")
	}
	entry := repo.created
	if entry.SourceType != domain.JournalSourceClosing || entry.SourceID == nil || *entry.SourceID != 7 {
		t.Fatalf("sumber jurnal salah: %q %v", entry.SourceType, entry.SourceID)
	}
	if entry.Status != domain.JournalStatusPosted {
		t.Fatalf("status harus POSTED, dapat %q", entry.Status)
	}
	if want := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC); !entry.EntryDate.Equal(want) {
		t.Fatalf("tanggal jurnal = %v, ingin %v", entry.EntryDate, want)
	}

	// Akun aset/kewajiban/ekuitas tidak boleh ikut ditutup.
	if _, ok := akunDari(entry, 1); ok {
		t.Error("akun aset tidak boleh masuk jurnal penutup")
	}
	if _, ok := akunDari(entry, 2); ok {
		t.Error("akun kewajiban tidak boleh masuk jurnal penutup")
	}

	rev, ok := akunDari(entry, 4)
	if !ok || rev.Debit.Cmp(domain.Rupiah(300)) != 0 || !rev.Credit.IsZero() {
		t.Errorf("pendapatan harus didebit 300, dapat %+v", rev)
	}
	exp, ok := akunDari(entry, 5)
	if !ok || exp.Credit.Cmp(domain.Rupiah(100)) != 0 || !exp.Debit.IsZero() {
		t.Errorf("beban harus dikredit 100, dapat %+v", exp)
	}
	re, ok := akunDari(entry, 31)
	if !ok || re.Credit.Cmp(domain.Rupiah(200)) != 0 || !re.Debit.IsZero() {
		t.Errorf("laba ditahan harus dikredit 200, dapat %+v", re)
	}

	if entry.TotalDebit.Cmp(domain.Rupiah(300)) != 0 || entry.TotalCredit.Cmp(domain.Rupiah(300)) != 0 {
		t.Fatalf("jurnal harus seimbang 300/300, dapat %s/%s", entry.TotalDebit, entry.TotalCredit)
	}
}

// TestPostClosingEntryRugiMendebitLabaDitahan: beban melebihi pendapatan
// menghasilkan rugi yang didebitkan ke Laba Ditahan.
func TestPostClosingEntryRugiMendebitLabaDitahan(t *testing.T) {
	repo := &fakeJournalRepo{}
	uc := newClosingUsecase(repo, []domain.AccountBalance{
		{AccountID: 4, Code: "4-1100", AccountType: domain.AccountTypeRevenue, NormalBalance: domain.NormalBalanceCredit, TotalCredit: domain.Rupiah(100)},
		{AccountID: 5, Code: "5-2100", AccountType: domain.AccountTypeExpense, NormalBalance: domain.NormalBalanceDebit, TotalDebit: domain.Rupiah(300)},
	}, closingMapping())

	if err := uc.PostClosingEntry(context.Background(), period2026(3, domain.PeriodStatusOpen), 99); err != nil {
		t.Fatalf("jurnal penutup harus terbit: %v", err)
	}
	re, ok := akunDari(repo.created, 31)
	if !ok || re.Debit.Cmp(domain.Rupiah(200)) != 0 || !re.Credit.IsZero() {
		t.Errorf("rugi 200 harus didebit ke laba ditahan, dapat %+v", re)
	}
	if repo.created.TotalDebit.Cmp(domain.Rupiah(300)) != 0 || repo.created.TotalCredit.Cmp(domain.Rupiah(300)) != 0 {
		t.Fatalf("jurnal harus tetap seimbang, dapat %s/%s", repo.created.TotalDebit, repo.created.TotalCredit)
	}
}

// TestPostClosingEntryTanpaSaldoTemporer: periode tanpa saldo pendapatan/beban
// tidak menerbitkan jurnal sama sekali.
func TestPostClosingEntryTanpaSaldoTemporer(t *testing.T) {
	repo := &fakeJournalRepo{}
	uc := newClosingUsecase(repo, []domain.AccountBalance{
		{AccountID: 1, Code: "1-1101", AccountType: domain.AccountTypeAsset, NormalBalance: domain.NormalBalanceDebit, TotalDebit: domain.Rupiah(500)},
		{AccountID: 2, Code: "2-1100", AccountType: domain.AccountTypeLiability, NormalBalance: domain.NormalBalanceCredit, TotalCredit: domain.Rupiah(500)},
	}, closingMapping())

	if err := uc.PostClosingEntry(context.Background(), period2026(3, domain.PeriodStatusOpen), 99); err != nil {
		t.Fatalf("tidak ada saldo temporer, tidak boleh gagal: %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatal("tidak boleh menulis jurnal tanpa saldo pendapatan/beban")
	}
}

// TestPostClosingEntryIdempoten: jurnal penutup POSTED yang sudah ada harus
// dideteksi sebelum menulis, bukan lewat pelanggaran unique index.
func TestPostClosingEntryIdempoten(t *testing.T) {
	repo := &fakeJournalRepo{sourceID: 77}
	uc := newClosingUsecase(repo, skenarioNeraca(), closingMapping())

	if err := uc.PostClosingEntry(context.Background(), period2026(3, domain.PeriodStatusOpen), 99); err != nil {
		t.Fatalf("jurnal penutup yang sudah ada harus diabaikan, dapat %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatal("tidak boleh mencoba menulis jurnal penutup kedua kali")
	}
}

// TestPostClosingEntryPemetaanLabaDitahanKurang menolak bila pemetaan Laba
// Ditahan belum diatur, sebelum apa pun ditulis.
func TestPostClosingEntryPemetaanLabaDitahanKurang(t *testing.T) {
	repo := &fakeJournalRepo{}
	uc := newClosingUsecase(repo, skenarioNeraca(), map[string]int64{})

	err := uc.PostClosingEntry(context.Background(), period2026(3, domain.PeriodStatusOpen), 99)
	if !errors.Is(err, domain.ErrAutoPostingSettings) {
		t.Fatalf("harus menandai pemetaan kurang, dapat %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatal("tidak boleh menulis jurnal saat pemetaan kurang")
	}
}

// TestPostClosingEntryPeriodeTidakValid menolak periode tanpa id.
func TestPostClosingEntryPeriodeTidakValid(t *testing.T) {
	repo := &fakeJournalRepo{}
	uc := newClosingUsecase(repo, skenarioNeraca(), closingMapping())

	if err := uc.PostClosingEntry(context.Background(), &domain.AccountingPeriod{}, 99); err == nil {
		t.Fatal("periode tanpa id harus ditolak")
	}
	if repo.createCalls != 0 {
		t.Fatal("tidak boleh menulis jurnal untuk periode tidak valid")
	}
}

// TestReverseClosingEntryTanpaJurnalPenutup tidak melakukan apa pun bila periode
// belum pernah ditutup buku.
func TestReverseClosingEntryTanpaJurnalPenutup(t *testing.T) {
	repo := &fakeJournalRepo{sourceErr: pgx.ErrNoRows}
	uc := newClosingUsecase(repo, nil, closingMapping())

	if err := uc.ReverseClosingEntry(context.Background(), period2026(3, domain.PeriodStatusOpen), 99); err != nil {
		t.Fatalf("tanpa jurnal penutup tidak boleh gagal: %v", err)
	}
	if repo.reverseCalls != 0 {
		t.Fatal("tidak boleh membalik jurnal yang tidak ada")
	}
}

// TestReverseClosingEntryMembalikJurnalPenutup membalik jurnal penutup periode.
func TestReverseClosingEntryMembalikJurnalPenutup(t *testing.T) {
	repo := &fakeJournalRepo{sourceID: 77}
	uc := newClosingUsecase(repo, nil, closingMapping())

	if err := uc.ReverseClosingEntry(context.Background(), period2026(3, domain.PeriodStatusOpen), 99); err != nil {
		t.Fatalf("pembalikan jurnal penutup gagal: %v", err)
	}
	if repo.reverseCalls != 1 || repo.reversedID != 77 {
		t.Fatalf("harus membalik jurnal 77 sekali, dapat calls=%d id=%d", repo.reverseCalls, repo.reversedID)
	}
}

// TestClosePeriodMenerbitkanJurnalPenutup: menutup periode memanggil penutupan
// buku sebelum mengunci status.
func TestClosePeriodMenerbitkanJurnalPenutup(t *testing.T) {
	periodRepo := &fakePeriodRepo{period: period2026(3, domain.PeriodStatusOpen)}
	closer := &fakeClosingManager{}
	uc := NewAccountingPeriodUsecase(periodRepo, closer)

	if err := uc.ClosePeriod(context.Background(), 2026, 3, 99); err != nil {
		t.Fatalf("ClosePeriod gagal: %v", err)
	}
	if closer.postedPeriodID != 7 {
		t.Fatalf("jurnal penutup tidak diterbitkan untuk periode 7, dapat %d", closer.postedPeriodID)
	}
	if periodRepo.closeCalls != 1 {
		t.Fatalf("periode harus dikunci sekali, dapat %d", periodRepo.closeCalls)
	}
}

// TestClosePeriodGagalPenutupTidakMengunci: bila jurnal penutup gagal, periode
// tidak boleh dikunci.
func TestClosePeriodGagalPenutupTidakMengunci(t *testing.T) {
	periodRepo := &fakePeriodRepo{period: period2026(3, domain.PeriodStatusOpen)}
	closer := &fakeClosingManager{postErr: errors.New("gagal")}
	uc := NewAccountingPeriodUsecase(periodRepo, closer)

	if err := uc.ClosePeriod(context.Background(), 2026, 3, 99); err == nil {
		t.Fatal("ClosePeriod harus meneruskan kegagalan jurnal penutup")
	}
	if periodRepo.closeCalls != 0 {
		t.Fatal("periode tidak boleh dikunci bila jurnal penutup gagal")
	}
}

// TestClosePeriodSudahTutupTidakMengulang: periode yang sudah CLOSED tidak
// menerbitkan jurnal penutup kedua kali.
func TestClosePeriodSudahTutupTidakMengulang(t *testing.T) {
	periodRepo := &fakePeriodRepo{period: period2026(3, domain.PeriodStatusClosed)}
	closer := &fakeClosingManager{}
	uc := NewAccountingPeriodUsecase(periodRepo, closer)

	if err := uc.ClosePeriod(context.Background(), 2026, 3, 99); err != nil {
		t.Fatalf("ClosePeriod periode tertutup harus sukses, dapat %v", err)
	}
	if closer.postedPeriodID != 0 || periodRepo.closeCalls != 0 {
		t.Fatal("periode tertutup tidak boleh diproses ulang")
	}
}

// TestReopenPeriodMembalikJurnalPenutup: membuka kembali periode membalik
// jurnal penutup agar pembukuan kembali seperti semula.
func TestReopenPeriodMembalikJurnalPenutup(t *testing.T) {
	periodRepo := &fakePeriodRepo{period: period2026(3, domain.PeriodStatusClosed)}
	closer := &fakeClosingManager{}
	uc := NewAccountingPeriodUsecase(periodRepo, closer)

	if err := uc.ReopenPeriod(context.Background(), 2026, 3, 99); err != nil {
		t.Fatalf("ReopenPeriod gagal: %v", err)
	}
	if closer.reversedPeriodID != 7 {
		t.Fatalf("jurnal penutup harus dibalik untuk periode 7, dapat %d", closer.reversedPeriodID)
	}
}

// existingClosingTx mensimulasikan transaksi yang sudah memiliki jurnal penutup
// POSTED. Pre-check harus mencegah percobaan insert kedua yang akan memicu
// unique violation dan membatalkan transaksi.
type existingClosingTx struct {
	domain.AccountingTx
	createCalls int
}

func (f *existingClosingTx) GetPostedJournalIDBySource(context.Context, string, int64) (int64, error) {
	return 55, nil
}

func (f *existingClosingTx) CreatePostedJournal(context.Context, *domain.JournalEntry) error {
	f.createCalls++
	return nil
}

func TestPostClosingEntryTxMelewatiJurnalYangSudahAda(t *testing.T) {
	repo := &fakeJournalRepo{}
	uc := newClosingUsecase(repo, skenarioNeraca(), closingMapping())
	tx := &existingClosingTx{}

	if err := uc.PostClosingEntryTx(context.Background(), tx, period2026(3, domain.PeriodStatusOpen), 99); err != nil {
		t.Fatalf("jurnal penutup yang sudah ada harus dilewati, dapat %v", err)
	}
	if tx.createCalls != 0 {
		t.Fatal("tidak boleh mencoba insert jurnal penutup kedua kali di dalam transaksi")
	}
}
