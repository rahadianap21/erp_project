package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go_backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

// --- Fakes khusus auto-posting ---

type fakeSettingsRepo struct {
	mapping   map[string]int64
	err       error
	listCalls int
}

func (f *fakeSettingsRepo) ListMapping(ctx context.Context) (map[string]int64, error) {
	f.listCalls++
	return f.mapping, f.err
}

type fakeAccountRepo struct {
	accounts      map[int64]domain.ChartOfAccount
	getByIDsCalls int
}

func (f *fakeAccountRepo) Create(context.Context, *domain.ChartOfAccount) error { return nil }
func (f *fakeAccountRepo) GetListCursor(context.Context, domain.ChartOfAccountParam) ([]domain.ChartOfAccount, int64, error) {
	return nil, 0, nil
}
func (f *fakeAccountRepo) GetByID(context.Context, int64) (*domain.ChartOfAccount, error) {
	return nil, nil
}
func (f *fakeAccountRepo) GetByIDs(ctx context.Context, ids []int64) ([]domain.ChartOfAccount, error) {
	f.getByIDsCalls++
	res := make([]domain.ChartOfAccount, 0, len(ids))
	for _, id := range ids {
		if acc, ok := f.accounts[id]; ok {
			res = append(res, acc)
		}
	}
	return res, nil
}
func (f *fakeAccountRepo) Update(context.Context, *domain.ChartOfAccount) error { return nil }
func (f *fakeAccountRepo) Delete(context.Context, int64, int64) error           { return nil }

type fakePeriodRepo struct {
	ensureErr  error
	ensured    []string
	period     *domain.AccountingPeriod
	closeCalls int
}

func (f *fakePeriodRepo) Get(context.Context, int, int) (*domain.AccountingPeriod, error) {
	return f.period, nil
}
func (f *fakePeriodRepo) EnsureOpen(ctx context.Context, year, month int, actorID int64) (*domain.AccountingPeriod, error) {
	if f.ensureErr != nil {
		return nil, f.ensureErr
	}
	f.ensured = append(f.ensured, fmt.Sprintf("%04d-%02d", year, month))
	return &domain.AccountingPeriod{PeriodYear: year, PeriodMonth: month, Status: domain.PeriodStatusOpen}, nil
}
func (f *fakePeriodRepo) Close(context.Context, int64, int64, int64) (int64, error) {
	f.closeCalls++
	return 1, nil
}
func (f *fakePeriodRepo) Reopen(context.Context, int, int) (int64, error) { return 1, nil }
func (f *fakePeriodRepo) List(context.Context, int) ([]domain.AccountingPeriod, error) {
	return nil, nil
}
func (f *fakePeriodRepo) WithTx(ctx context.Context, fn func(domain.AccountingTx) error) error {
	return fn(&fakeAccountingTx{repo: f})
}

type fakeAccountingTx struct {
	repo *fakePeriodRepo
}

func (f *fakeAccountingTx) LockPeriod(context.Context, int, int) (*domain.AccountingPeriod, error) {
	return f.repo.period, nil
}
func (f *fakeAccountingTx) CreatePeriod(_ context.Context, year, month int, _ int64) (*domain.AccountingPeriod, error) {
	return &domain.AccountingPeriod{PeriodYear: year, PeriodMonth: month, Status: domain.PeriodStatusOpen}, nil
}
func (f *fakeAccountingTx) AccountBalances(context.Context, time.Time, time.Time) ([]domain.AccountBalance, error) {
	return nil, nil
}
func (f *fakeAccountingTx) GetAccountsByIDs(context.Context, []int64) ([]domain.ChartOfAccount, error) {
	return nil, nil
}
func (f *fakeAccountingTx) ListAccountingSettings(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (f *fakeAccountingTx) NextJournalSeq(context.Context, string) (int64, error) { return 1, nil }
func (f *fakeAccountingTx) GetPostedJournalIDBySource(context.Context, string, int64) (int64, error) {
	return 0, pgx.ErrNoRows
}
func (f *fakeAccountingTx) CreatePostedJournal(context.Context, *domain.JournalEntry) error {
	return nil
}
func (f *fakeAccountingTx) ReverseJournal(context.Context, int64, time.Time, string, int64) (*domain.JournalEntry, error) {
	return nil, nil
}
func (f *fakeAccountingTx) ClosePeriod(context.Context, int64, int64, int64) (int64, error) {
	return f.repo.Close(context.Background(), 0, 0, 0)
}
func (f *fakeAccountingTx) ReopenPeriod(context.Context, int, int) (int64, error) {
	return f.repo.Reopen(context.Background(), 0, 0)
}

type fakeJournalRepo struct {
	seq         int64
	nextErr     error
	createErr   error
	created     *domain.JournalEntry
	createCalls int

	sourceID     int64
	sourceErr    error
	reverseCalls int
	reversedID   int64
}

func (f *fakeJournalRepo) CreateDraft(context.Context, *domain.JournalEntry) error { return nil }
func (f *fakeJournalRepo) CreatePosted(ctx context.Context, entry *domain.JournalEntry) error {
	f.createCalls++
	if f.createErr != nil {
		return f.createErr
	}
	snapshot := *entry
	f.created = &snapshot
	return nil
}
func (f *fakeJournalRepo) GetListCursor(context.Context, domain.JournalEntryParam) ([]domain.JournalEntry, int64, error) {
	return nil, 0, nil
}
func (f *fakeJournalRepo) GetByID(context.Context, int64) (*domain.JournalEntry, error) {
	return nil, nil
}
func (f *fakeJournalRepo) Post(context.Context, int64, int64) (int64, error)         { return 1, nil }
func (f *fakeJournalRepo) Void(context.Context, int64, int64, string) (int64, error) { return 1, nil }
func (f *fakeJournalRepo) Reverse(_ context.Context, originalID int64, _ time.Time, _ string, _ int64) (*domain.JournalEntry, error) {
	f.reverseCalls++
	f.reversedID = originalID
	return nil, nil
}
func (f *fakeJournalRepo) NextSeq(context.Context, string) (int64, error) {
	if f.nextErr != nil {
		return 0, f.nextErr
	}
	f.seq++
	return f.seq, nil
}
func (f *fakeJournalRepo) GetPostedIDBySource(context.Context, string, int64) (int64, error) {
	if f.sourceErr != nil {
		return 0, f.sourceErr
	}
	if f.sourceID == 0 {
		return 0, pgx.ErrNoRows
	}
	return f.sourceID, nil
}

func postableAccounts() map[int64]domain.ChartOfAccount {
	return map[int64]domain.ChartOfAccount{
		11: {ID: 11, Code: "1-1400", IsActive: true},
		12: {ID: 12, Code: "1-1500", IsActive: true},
		21: {ID: 21, Code: "2-1100", IsActive: true},
	}
}

func defaultMapping() map[string]int64 {
	return map[string]int64{"INVENTORY": 11, "VAT_INPUT": 12, "AP": 21}
}

func newPostingUsecase(repo *fakeJournalRepo, settings *fakeSettingsRepo, period *fakePeriodRepo) domain.JournalUsecase {
	return NewJournalUsecase(repo, &fakeAccountRepo{accounts: postableAccounts()}, period, settings, &fakeReportRepo{})
}

func receivedGRN(id int64) *domain.GoodsReceiptNote {
	return &domain.GoodsReceiptNote{
		ID:           id,
		GRNNumber:    "GRN/20261014/0001",
		ReceivedDate: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC),
		Items:        []domain.GoodsReceiptNoteDetail{{PurchaseOrderDetailID: 100, QtyReceived: 10}},
	}
}

func poWithTax(taxType string, unitPrice domain.Money) *domain.PurchaseOrder {
	return &domain.PurchaseOrder{
		ID:       7,
		PoNumber: "PO/20261013/0001",
		TaxType:  taxType,
		Items:    []domain.PurchaseOrderDetail{{ID: 100, UnitPrice: unitPrice}},
	}
}

// TestGRNReceivedValuesMatrix menguji pemidahan pajak per tipe PO dan memastikan
// jurnal otomatis memakai angka yang sama dengan dokumen PO untuk penerimaan penuh.
func TestGRNReceivedValuesMatrix(t *testing.T) {
	cases := []struct {
		taxType string
		wantDPP domain.Money
		wantVAT domain.Money
	}{
		{"NON_PPN", domain.Rupiah(10000), domain.Money{}},
		{"TAX_EXCLUSIVE", domain.Rupiah(10000), domain.Rupiah(1200)},
		{"TAX_INCLUSIVE", domain.NewMoney(892857), domain.NewMoney(107143)},
	}
	for _, c := range cases {
		t.Run(c.taxType, func(t *testing.T) {
			// UnitPrice 1000 x qty 10 = 10000.
			dpp, vat, err := grnReceivedValues(receivedGRN(1), poWithTax(c.taxType, domain.Rupiah(1000)))
			if err != nil {
				t.Fatalf("tidak boleh gagal: %v", err)
			}
			if dpp.Cmp(c.wantDPP) != 0 || vat.Cmp(c.wantVAT) != 0 {
				t.Fatalf("DPP/PPN salah: dapat (%s, %s), ingin (%s, %s)", dpp, vat, c.wantDPP, c.wantVAT)
			}
		})
	}
}

// TestPostGRNJournalExclusiveSeimbang memastikan jurnal POSTED dibuat dengan
// akun dari pemetaan, sumber GRN, dan debit tepat sama dengan kredit.
func TestPostGRNJournalExclusiveSeimbang(t *testing.T) {
	repo := &fakeJournalRepo{}
	period := &fakePeriodRepo{}
	uc := newPostingUsecase(repo, &fakeSettingsRepo{mapping: defaultMapping()}, period)

	if err := uc.PostGRNJournal(context.Background(), receivedGRN(5), poWithTax("TAX_EXCLUSIVE", domain.Rupiah(1000)), 99); err != nil {
		t.Fatalf("jurnal harus terbit: %v", err)
	}
	if repo.created == nil {
		t.Fatal("repository tidak menerima jurnal")
	}
	got := repo.created
	if got.Status != domain.JournalStatusPosted {
		t.Fatalf("status harus POSTED, dapat %q", got.Status)
	}
	if got.SourceType != domain.JournalSourceGRN || got.SourceID == nil || *got.SourceID != 5 {
		t.Fatalf("sumber jurnal salah: %q %v", got.SourceType, got.SourceID)
	}
	if len(got.Lines) != 3 {
		t.Fatalf("harus 3 baris, dapat %d", len(got.Lines))
	}

	var debit, credit domain.Money
	for _, l := range got.Lines {
		debit = debit.Add(l.Debit)
		credit = credit.Add(l.Credit)
	}
	if debit.Cmp(credit) != 0 || debit.Cmp(domain.Rupiah(11200)) != 0 {
		t.Fatalf("jurnal tidak seimbang: debit %s kredit %s", debit, credit)
	}
	if len(period.ensured) != 1 || period.ensured[0] != "2026-10" {
		t.Fatalf("periode jurnal salah: %v", period.ensured)
	}
}

// TestPostGRNJournalTanPPNGaVATInput memastikan PO tanpa PPN tidak menyentuh
// akun PPN sehingga hanya dua baris yang terbit.
func TestPostGRNJournalTanPPNGaVATInput(t *testing.T) {
	repo := &fakeJournalRepo{}
	uc := newPostingUsecase(repo, &fakeSettingsRepo{mapping: map[string]int64{"INVENTORY": 11, "AP": 21}}, &fakePeriodRepo{})

	if err := uc.PostGRNJournal(context.Background(), receivedGRN(5), poWithTax("NON_PPN", domain.Rupiah(1000)), 99); err != nil {
		t.Fatalf("jurnal harus terbit tanpa akun PPN: %v", err)
	}
	if len(repo.created.Lines) != 2 {
		t.Fatalf("harus 2 baris, dapat %d", len(repo.created.Lines))
	}
}

// TestPostGRNJournalIdempoten memastikan bentrok pada unique index sumber
// diperlakukan sebagai "sudah terbit", bukan kegagalan.
func TestPostGRNJournalIdempoten(t *testing.T) {
	repo := &fakeJournalRepo{createErr: domain.ErrJournalAlreadyPost}
	uc := newPostingUsecase(repo, &fakeSettingsRepo{mapping: defaultMapping()}, &fakePeriodRepo{})

	if err := uc.PostGRNJournal(context.Background(), receivedGRN(5), poWithTax("NON_PPN", domain.Rupiah(1000)), 99); err != nil {
		t.Fatalf("harus mengabaikan jurnal yang sudah ada, dapat %v", err)
	}
}

// TestPostGRNJournalPemetaanKurang menolak dengan pesan yang menyebut kunci
// pemetaan yang belum diisi.
func TestPostGRNJournalPemetaanKurang(t *testing.T) {
	repo := &fakeJournalRepo{}
	uc := newPostingUsecase(repo, &fakeSettingsRepo{mapping: map[string]int64{"AP": 21}}, &fakePeriodRepo{})

	err := uc.PostGRNJournal(context.Background(), receivedGRN(5), poWithTax("NON_PPN", domain.Rupiah(1000)), 99)
	if !errors.Is(err, domain.ErrAutoPostingSettings) {
		t.Fatalf("harus menandai pemetaan kurang, dapat %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatal("tidak boleh menulis jurnal saat pemetaan kurang")
	}
}

// TestPostGRNJournalPeriodeTertutup memastikan periode tertutup menghentikan
// penerbitan sebelum apa pun ditulis.
func TestPostGRNJournalPeriodeTertutup(t *testing.T) {
	repo := &fakeJournalRepo{}
	period := &fakePeriodRepo{ensureErr: domain.ErrJournalPeriodClosed}
	uc := newPostingUsecase(repo, &fakeSettingsRepo{mapping: defaultMapping()}, period)

	err := uc.PostGRNJournal(context.Background(), receivedGRN(5), poWithTax("NON_PPN", domain.Rupiah(1000)), 99)
	if !errors.Is(err, domain.ErrJournalPeriodClosed) {
		t.Fatalf("harus meneruskan error periode, dapat %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatal("tidak boleh menulis jurnal saat periode tertutup")
	}
}

// TestPostGRNJournalsBatchMembacaReferensiSekali memastikan batch beberapa GRN
// tidak mengulang query pengaturan, akun, dan periode per GRN.
func TestPostGRNJournalsBatchMembacaReferensiSekali(t *testing.T) {
	repo := &fakeJournalRepo{}
	settings := &fakeSettingsRepo{mapping: defaultMapping()}
	accounts := &fakeAccountRepo{accounts: postableAccounts()}
	period := &fakePeriodRepo{}
	uc := NewJournalUsecase(repo, accounts, period, settings, &fakeReportRepo{})

	grns := []*domain.GoodsReceiptNote{receivedGRN(1), receivedGRN(2), receivedGRN(3)}
	if err := uc.PostGRNJournals(context.Background(), grns, poWithTax("NON_PPN", domain.Rupiah(1000)), 99); err != nil {
		t.Fatalf("jurnal batch harus terbit: %v", err)
	}
	if settings.listCalls != 1 {
		t.Fatalf("ListMapping harus sekali, dapat %d", settings.listCalls)
	}
	if accounts.getByIDsCalls != 1 {
		t.Fatalf("GetByIDs harus sekali, dapat %d", accounts.getByIDsCalls)
	}
	if len(period.ensured) != 1 {
		t.Fatalf("EnsureOpen harus sekali untuk bulan yang sama, dapat %v", period.ensured)
	}
	if repo.createCalls != len(grns) {
		t.Fatalf("harus menulis %d jurnal, dapat %d", len(grns), repo.createCalls)
	}
}

// TestAkunHeaderDitolak memastikan baris jurnal tidak boleh menunjuk akun header.
func TestAkunHeaderDitolak(t *testing.T) {
	repo := &fakeJournalRepo{}
	accounts := map[int64]domain.ChartOfAccount{
		11: {ID: 11, Code: "1-1400", IsActive: true, IsHeader: true},
		21: {ID: 21, Code: "2-1100", IsActive: true},
	}
	uc := NewJournalUsecase(repo, &fakeAccountRepo{accounts: accounts}, &fakePeriodRepo{},
		&fakeSettingsRepo{mapping: map[string]int64{"INVENTORY": 11, "AP": 21}}, &fakeReportRepo{})

	err := uc.PostGRNJournal(context.Background(), receivedGRN(5), poWithTax("NON_PPN", domain.Rupiah(1000)), 99)
	if err == nil {
		t.Fatal("akun header tidak boleh diposting")
	}
	if repo.createCalls != 0 {
		t.Fatal("tidak boleh menulis jurnal bila akun tidak layak")
	}
}
