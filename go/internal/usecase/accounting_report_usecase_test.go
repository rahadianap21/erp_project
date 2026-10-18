package usecase

import (
	"context"
	"testing"
	"time"

	"go_backend/internal/domain"
)

// --- Fakes khusus laporan keuangan ---

type fakeReportRepo struct {
	balances   []domain.AccountBalance
	lines      []domain.AccountLedgerLine
	openDebit  float64
	openCredit float64
	balanceErr error
}

func (f *fakeReportRepo) AccountBalances(context.Context, time.Time, time.Time) ([]domain.AccountBalance, error) {
	if f.balanceErr != nil {
		return nil, f.balanceErr
	}
	return f.balances, nil
}

func (f *fakeReportRepo) AccountLedgerLines(context.Context, int64, time.Time, time.Time) ([]domain.AccountLedgerLine, error) {
	return f.lines, nil
}

func (f *fakeReportRepo) AccountOpeningBalance(context.Context, int64, time.Time) (float64, float64, error) {
	return f.openDebit, f.openCredit, nil
}

type fakeReportCOA struct {
	account *domain.ChartOfAccount
}

func (f *fakeReportCOA) Create(context.Context, *domain.ChartOfAccount) error { return nil }
func (f *fakeReportCOA) GetListCursor(context.Context, domain.ChartOfAccountParam) ([]domain.ChartOfAccount, int64, error) {
	return nil, 0, nil
}
func (f *fakeReportCOA) GetByID(context.Context, int64) (*domain.ChartOfAccount, error) {
	if f.account == nil {
		return nil, domain.ErrAccountNotFound
	}
	return f.account, nil
}
func (f *fakeReportCOA) GetByIDs(context.Context, []int64) ([]domain.ChartOfAccount, error) {
	return nil, nil
}
func (f *fakeReportCOA) Update(context.Context, *domain.ChartOfAccount) error { return nil }
func (f *fakeReportCOA) Delete(context.Context, int64, int64) error           { return nil }

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// skenarioNeraca membuat himpunan saldo yang memenuhi A = K + E bila pendapatan
// 300 dan beban 100 diakui sebagai laba belum ditutup 200.
func skenarioNeraca() []domain.AccountBalance {
	return []domain.AccountBalance{
		{AccountID: 1, Code: "1-1101", Name: "KAS TUNAI", AccountType: domain.AccountTypeAsset, NormalBalance: domain.NormalBalanceDebit, TotalDebit: 500},
		{AccountID: 2, Code: "2-1100", Name: "HUTANG USAHA", AccountType: domain.AccountTypeLiability, NormalBalance: domain.NormalBalanceCredit, TotalCredit: 200},
		{AccountID: 3, Code: "3-1100", Name: "MODAL PEMILIK", AccountType: domain.AccountTypeEquity, NormalBalance: domain.NormalBalanceCredit, TotalCredit: 100},
		{AccountID: 4, Code: "4-1100", Name: "PENDAPATAN PENJUALAN", AccountType: domain.AccountTypeRevenue, NormalBalance: domain.NormalBalanceCredit, TotalCredit: 300},
		{AccountID: 5, Code: "5-2100", Name: "BEBAN GAJI", AccountType: domain.AccountTypeExpense, NormalBalance: domain.NormalBalanceDebit, TotalDebit: 100},
	}
}

func TestIncomeStatementMenghitungLabaRugi(t *testing.T) {
	repo := &fakeReportRepo{balances: skenarioNeraca()}
	uc := NewAccountingReportUsecase(repo, &fakeReportCOA{})

	statement, err := uc.IncomeStatement(context.Background(), date(2026, time.January, 1), date(2026, time.December, 31))
	if err != nil {
		t.Fatalf("IncomeStatement gagal: %v", err)
	}
	if len(statement.Revenue) != 1 || len(statement.Expenses) != 1 {
		t.Fatalf("jumlah baris tidak sesuai: revenue=%d expenses=%d", len(statement.Revenue), len(statement.Expenses))
	}
	if statement.TotalRevenue != 300 {
		t.Errorf("TotalRevenue = %v, ingin 300", statement.TotalRevenue)
	}
	if statement.TotalExpenses != 100 {
		t.Errorf("TotalExpenses = %v, ingin 100", statement.TotalExpenses)
	}
	if statement.NetIncome != 200 {
		t.Errorf("NetIncome = %v, ingin 200", statement.NetIncome)
	}
	// Akun aset/kewajiban/ekuitas tidak boleh bocor ke laba rugi.
	if statement.Revenue[0].Code != "4-1100" {
		t.Errorf("baris pendapatan = %q, ingin 4-1100", statement.Revenue[0].Code)
	}
}

func TestBalanceSheetMenambahLabaBelumDitutupDanSeimbang(t *testing.T) {
	repo := &fakeReportRepo{balances: skenarioNeraca()}
	uc := NewAccountingReportUsecase(repo, &fakeReportCOA{})

	sheet, err := uc.BalanceSheet(context.Background(), date(2026, time.December, 31))
	if err != nil {
		t.Fatalf("BalanceSheet gagal: %v", err)
	}
	if sheet.TotalAssets != 500 {
		t.Errorf("TotalAssets = %v, ingin 500", sheet.TotalAssets)
	}
	if sheet.TotalLiabilities != 200 {
		t.Errorf("TotalLiabilities = %v, ingin 200", sheet.TotalLiabilities)
	}
	// 100 ekuitas nyata + 200 laba belum ditutup.
	if sheet.TotalEquity != 300 {
		t.Errorf("TotalEquity = %v, ingin 300", sheet.TotalEquity)
	}
	if sheet.TotalLiabilitiesAndEquity != 500 {
		t.Errorf("TotalLiabilitiesAndEquity = %v, ingin 500", sheet.TotalLiabilitiesAndEquity)
	}
	if !sheet.Balanced {
		t.Error("neraca seharusnya seimbang")
	}

	var synthetic *domain.ReportAccount
	for i := range sheet.Equity {
		if sheet.Equity[i].Code == netIncomeEquityCode {
			synthetic = &sheet.Equity[i]
		}
	}
	if synthetic == nil {
		t.Fatalf("baris laba belum ditutup (%s) tidak ada di ekuitas", netIncomeEquityCode)
	}
	if synthetic.Balance != 200 {
		t.Errorf("laba belum ditutup = %v, ingin 200", synthetic.Balance)
	}
}

func TestBalanceSheetTidakMenambahBarisLabaNol(t *testing.T) {
	// Pendapatan 300, beban 300 => laba nol => tidak perlu baris sintetis.
	repo := &fakeReportRepo{balances: []domain.AccountBalance{
		{AccountID: 1, Code: "1-1101", AccountType: domain.AccountTypeAsset, NormalBalance: domain.NormalBalanceDebit, TotalDebit: 100},
		{AccountID: 3, Code: "3-1100", AccountType: domain.AccountTypeEquity, NormalBalance: domain.NormalBalanceCredit, TotalCredit: 100},
		{AccountID: 4, Code: "4-1100", AccountType: domain.AccountTypeRevenue, NormalBalance: domain.NormalBalanceCredit, TotalCredit: 300},
		{AccountID: 5, Code: "5-2100", AccountType: domain.AccountTypeExpense, NormalBalance: domain.NormalBalanceDebit, TotalDebit: 300},
	}}
	uc := NewAccountingReportUsecase(repo, &fakeReportCOA{})

	sheet, err := uc.BalanceSheet(context.Background(), date(2026, time.December, 31))
	if err != nil {
		t.Fatalf("BalanceSheet gagal: %v", err)
	}
	for _, account := range sheet.Equity {
		if account.Code == netIncomeEquityCode {
			t.Fatal("baris laba belum ditutup tidak seharusnya muncul saat laba nol")
		}
	}
	if sheet.TotalEquity != 100 || !sheet.Balanced {
		t.Errorf("ekuitas=%v balanced=%v, ingin 100/true", sheet.TotalEquity, sheet.Balanced)
	}
}

func TestTrialBalanceMenjumlahDebitKreditSeimbang(t *testing.T) {
	repo := &fakeReportRepo{balances: skenarioNeraca()}
	uc := NewAccountingReportUsecase(repo, &fakeReportCOA{})

	tb, err := uc.TrialBalance(context.Background(), date(2026, time.January, 1), date(2026, time.December, 31))
	if err != nil {
		t.Fatalf("TrialBalance gagal: %v", err)
	}
	// Debit: 500 aset + 100 beban = 600. Kredit: 200 + 100 + 300 = 600.
	if tb.TotalDebit != 600 {
		t.Errorf("TotalDebit = %v, ingin 600", tb.TotalDebit)
	}
	if tb.TotalCredit != 600 {
		t.Errorf("TotalCredit = %v, ingin 600", tb.TotalCredit)
	}
	if !tb.Balanced {
		t.Error("neraca saldo seharusnya seimbang")
	}
	// Akun kredit normal hanya mengisi kolom kredit.
	for _, account := range tb.Accounts {
		if account.Code == "2-1100" && account.Debit != 0 {
			t.Errorf("hutang seharusnya tanpa kolom debit, dapat %v", account.Debit)
		}
	}
}

func TestGeneralLedgerSaldoBerjalanAkunDebit(t *testing.T) {
	repo := &fakeReportRepo{
		openDebit:  1000,
		openCredit: 200,
		lines: []domain.AccountLedgerLine{
			{EntryNumber: "JV-1", EntryDate: date(2026, time.March, 1), Debit: 500},
			{EntryNumber: "JV-2", EntryDate: date(2026, time.March, 5), Credit: 300},
		},
	}
	coa := &fakeReportCOA{account: &domain.ChartOfAccount{
		ID: 11, Code: "1-1400", Name: "PERSEDIAAN", AccountType: domain.AccountTypeAsset, NormalBalance: domain.NormalBalanceDebit,
	}}
	uc := NewAccountingReportUsecase(repo, coa)

	ledger, err := uc.GeneralLedger(context.Background(), 11, date(2026, time.March, 1), date(2026, time.March, 31))
	if err != nil {
		t.Fatalf("GeneralLedger gagal: %v", err)
	}
	if ledger.OpeningBalance != 800 {
		t.Errorf("OpeningBalance = %v, ingin 800", ledger.OpeningBalance)
	}
	if len(ledger.Lines) != 2 {
		t.Fatalf("jumlah baris = %d, ingin 2", len(ledger.Lines))
	}
	if ledger.Lines[0].Balance != 1300 {
		t.Errorf("saldo berjalan baris 1 = %v, ingin 1300", ledger.Lines[0].Balance)
	}
	if ledger.Lines[1].Balance != 1000 {
		t.Errorf("saldo berjalan baris 2 = %v, ingin 1000", ledger.Lines[1].Balance)
	}
	if ledger.TotalDebit != 500 || ledger.TotalCredit != 300 {
		t.Errorf("total debit/kredit = %v/%v, ingin 500/300", ledger.TotalDebit, ledger.TotalCredit)
	}
	if ledger.ClosingBalance != 1000 {
		t.Errorf("ClosingBalance = %v, ingin 1000", ledger.ClosingBalance)
	}
}

func TestGeneralLedgerSaldoBerjalanAkunKredit(t *testing.T) {
	repo := &fakeReportRepo{
		openDebit:  100,
		openCredit: 500,
		lines: []domain.AccountLedgerLine{
			{EntryNumber: "JV-1", EntryDate: date(2026, time.March, 1), Debit: 50},
			{EntryNumber: "JV-2", EntryDate: date(2026, time.March, 5), Credit: 250},
		},
	}
	coa := &fakeReportCOA{account: &domain.ChartOfAccount{
		ID: 21, Code: "2-1100", Name: "HUTANG USAHA", AccountType: domain.AccountTypeLiability, NormalBalance: domain.NormalBalanceCredit,
	}}
	uc := NewAccountingReportUsecase(repo, coa)

	ledger, err := uc.GeneralLedger(context.Background(), 21, date(2026, time.March, 1), date(2026, time.March, 31))
	if err != nil {
		t.Fatalf("GeneralLedger gagal: %v", err)
	}
	// Kredit normal: saldo awal 400, lalu debit 50 turun ke 350, kredit 250 naik ke 600.
	if ledger.OpeningBalance != 400 {
		t.Errorf("OpeningBalance = %v, ingin 400", ledger.OpeningBalance)
	}
	if ledger.Lines[0].Balance != 350 {
		t.Errorf("saldo berjalan baris 1 = %v, ingin 350", ledger.Lines[0].Balance)
	}
	if ledger.ClosingBalance != 600 {
		t.Errorf("ClosingBalance = %v, ingin 600", ledger.ClosingBalance)
	}
}

func TestLaporanMenolakRentangTanggalTerbalik(t *testing.T) {
	uc := NewAccountingReportUsecase(&fakeReportRepo{}, &fakeReportCOA{})

	if _, err := uc.IncomeStatement(context.Background(), date(2026, time.December, 31), date(2026, time.January, 1)); err != domain.ErrRentangTanggalInvalid {
		t.Errorf("IncomeStatement err = %v, ingin ErrRentangTanggalInvalid", err)
	}
	if _, err := uc.TrialBalance(context.Background(), date(2026, time.December, 31), date(2026, time.January, 1)); err != domain.ErrRentangTanggalInvalid {
		t.Errorf("TrialBalance err = %v, ingin ErrRentangTanggalInvalid", err)
	}
	if _, err := uc.GeneralLedger(context.Background(), 11, date(2026, time.December, 31), date(2026, time.January, 1)); err != domain.ErrRentangTanggalInvalid {
		t.Errorf("GeneralLedger err = %v, ingin ErrRentangTanggalInvalid", err)
	}
	if _, err := uc.BalanceSheet(context.Background(), time.Time{}); err != domain.ErrRentangTanggalInvalid {
		t.Errorf("BalanceSheet err = %v, ingin ErrRentangTanggalInvalid", err)
	}
}

func TestGeneralLedgerAkunTidakDitemukan(t *testing.T) {
	uc := NewAccountingReportUsecase(&fakeReportRepo{}, &fakeReportCOA{})

	if _, err := uc.GeneralLedger(context.Background(), 0, date(2026, time.January, 1), date(2026, time.December, 31)); err != domain.ErrAccountNotFound {
		t.Errorf("GeneralLedger id 0 err = %v, ingin ErrAccountNotFound", err)
	}
	if _, err := uc.GeneralLedger(context.Background(), 999, date(2026, time.January, 1), date(2026, time.December, 31)); err != domain.ErrAccountNotFound {
		t.Errorf("GeneralLedger akun tak ada err = %v, ingin ErrAccountNotFound", err)
	}
}
