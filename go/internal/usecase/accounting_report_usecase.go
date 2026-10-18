package usecase

import (
	"context"
	"math"
	"time"

	"go_backend/internal/domain"
)

// reportEpoch adalah batas bawah saldo kumulatif laporan Neraca. Tidak ada
// jurnal ber-sejar sebelum tahun 1900 pada sistem ini, sehingga rentang
// [reportEpoch, asOf] menangkap seluruh jurnal POSTED yang pernah ada.
var reportEpoch = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)

// netIncomeEquityCode adalah kode baris ekuitas sintetis pada Neraca yang
// membawa sisa pendapatan-dikurangi-beban (laba belum ditutup). Kode ini tidak
// pernah ada di bagan akun, sengaja berada di rentang ekuitas (3-xxxx) dan
// diurutkan setelah seluruh akun ekuitas nyata.
const netIncomeEquityCode = "3-1900"

// reportTolerance adalah ambang selisih pembulatan yang masih dianggap seimbang.
const reportTolerance = 0.01

// reportDate membuang komponen jam, karena laporan dibatasi pada tanggal murni
// (kolom t_journal_entries.entry_date bertipe DATE).
func reportDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// validateRange memastikan kedua tanggal terisi dan urutannya benar.
func validateRange(from, to time.Time) error {
	if from.IsZero() || to.IsZero() || reportDate(from).After(reportDate(to)) {
		return domain.ErrRentangTanggalInvalid
	}
	return nil
}

// saldoTertanda menghitung saldo bertanda sesuai saldo normal akun: akun
// ber-saldo normal DEBIT bertambah pada kolom debit, begitu pula sebaliknya.
func saldoTertanda(normalBalance string, debit, credit float64) float64 {
	if normalBalance == domain.NormalBalanceCredit {
		return credit - debit
	}
	return debit - credit
}

// toReportAccounts mengubah agregat mentah menjadi baris laporan yang sudah
// dibulatkan dua desimal, dengan saldo bertanda sesuai saldo normal akun.
func toReportAccounts(balances []domain.AccountBalance) []domain.ReportAccount {
	accounts := make([]domain.ReportAccount, 0, len(balances))
	for _, b := range balances {
		accounts = append(accounts, domain.ReportAccount{
			AccountID:   b.AccountID,
			Code:        b.Code,
			Name:        b.Name,
			AccountType: b.AccountType,
			Debit:       round2(b.TotalDebit),
			Credit:      round2(b.TotalCredit),
			Balance:     round2(saldoTertanda(b.NormalBalance, b.TotalDebit, b.TotalCredit)),
		})
	}
	return accounts
}

// sumReportBalance menjumlahkan saldo bertanda sejumlah baris laporan.
func sumReportBalance(accounts []domain.ReportAccount) float64 {
	var total float64
	for _, a := range accounts {
		total += a.Balance
	}
	return round2(total)
}

// reportUsecase menyusun Neraca, Laba Rugi, Neraca Saldo, dan Buku Besar dari
// agregat jurnal POSTED.
type reportUsecase struct {
	repo domain.AccountingReportRepository
	coa  domain.ChartOfAccountRepository
}

// NewAccountingReportUsecase menginisialisasi usecase laporan keuangan.
func NewAccountingReportUsecase(repo domain.AccountingReportRepository, coa domain.ChartOfAccountRepository) domain.AccountingReportUsecase {
	return &reportUsecase{repo: repo, coa: coa}
}

// IncomeStatement (Laba Rugi) menghitung pendapatan dan beban pada rentang
// tanggal. Angka bersaldo normal, sehingga pendapatan bernilai positif bila
// lebih besar dari beban.
func (u *reportUsecase) IncomeStatement(ctx context.Context, from, to time.Time) (*domain.IncomeStatement, error) {
	if err := validateRange(from, to); err != nil {
		return nil, err
	}

	balances, err := u.repo.AccountBalances(ctx, reportDate(from), reportDate(to))
	if err != nil {
		return nil, err
	}

	statement := &domain.IncomeStatement{
		From:     reportDate(from),
		To:       reportDate(to),
		Revenue:  make([]domain.ReportAccount, 0),
		Expenses: make([]domain.ReportAccount, 0),
	}
	for _, account := range toReportAccounts(balances) {
		switch account.AccountType {
		case domain.AccountTypeRevenue:
			statement.Revenue = append(statement.Revenue, account)
		case domain.AccountTypeExpense:
			statement.Expenses = append(statement.Expenses, account)
		}
	}

	statement.TotalRevenue = sumReportBalance(statement.Revenue)
	statement.TotalExpenses = sumReportBalance(statement.Expenses)
	statement.NetIncome = round2(statement.TotalRevenue - statement.TotalExpenses)
	return statement, nil
}

// BalanceSheet (Neraca) menghitung saldo kumulatif seluruh akun sampai tanggal
// asOf. Sisa pendapatan-dikurangi-beban sepanjang masa disajikan sebagai baris
// ekuitas sintetis sehingga persamaan akuntansi tetap seimbang, mengingat
// sistem ini tidak mencatat jurnal penutupan buku.
func (u *reportUsecase) BalanceSheet(ctx context.Context, asOf time.Time) (*domain.BalanceSheet, error) {
	if asOf.IsZero() {
		return nil, domain.ErrRentangTanggalInvalid
	}

	to := reportDate(asOf)
	balances, err := u.repo.AccountBalances(ctx, reportEpoch, to)
	if err != nil {
		return nil, err
	}

	sheet := &domain.BalanceSheet{
		AsOf:        to,
		Assets:      make([]domain.ReportAccount, 0),
		Liabilities: make([]domain.ReportAccount, 0),
		Equity:      make([]domain.ReportAccount, 0),
	}

	var netIncome float64
	for _, account := range toReportAccounts(balances) {
		switch account.AccountType {
		case domain.AccountTypeAsset:
			sheet.Assets = append(sheet.Assets, account)
		case domain.AccountTypeLiability:
			sheet.Liabilities = append(sheet.Liabilities, account)
		case domain.AccountTypeEquity:
			sheet.Equity = append(sheet.Equity, account)
		case domain.AccountTypeRevenue:
			netIncome += account.Balance
		case domain.AccountTypeExpense:
			netIncome -= account.Balance
		}
	}

	sheet.TotalAssets = sumReportBalance(sheet.Assets)
	sheet.TotalLiabilities = sumReportBalance(sheet.Liabilities)
	sheet.TotalEquity = sumReportBalance(sheet.Equity)

	netIncome = round2(netIncome)
	if math.Abs(netIncome) >= reportTolerance {
		sheet.Equity = append(sheet.Equity, domain.ReportAccount{
			Code:        netIncomeEquityCode,
			Name:        "LABA (RUGI) BELUM DITUTUP",
			AccountType: domain.AccountTypeEquity,
			Balance:     netIncome,
		})
		sheet.TotalEquity = round2(sheet.TotalEquity + netIncome)
	}

	sheet.TotalLiabilitiesAndEquity = round2(sheet.TotalLiabilities + sheet.TotalEquity)
	sheet.Balanced = math.Abs(sheet.TotalAssets-sheet.TotalLiabilitiesAndEquity) <= reportTolerance
	return sheet, nil
}

// TrialBalance (Neraca Saldo) mendaftar akun yang beraktivitas pada rentang
// tanggal beserta kolom debit dan kreditnya. Karena setiap jurnal seimbang,
// jumlah kedua kolom selalu sama.
func (u *reportUsecase) TrialBalance(ctx context.Context, from, to time.Time) (*domain.TrialBalance, error) {
	if err := validateRange(from, to); err != nil {
		return nil, err
	}

	balances, err := u.repo.AccountBalances(ctx, reportDate(from), reportDate(to))
	if err != nil {
		return nil, err
	}

	report := &domain.TrialBalance{
		From:     reportDate(from),
		To:       reportDate(to),
		Accounts: toReportAccounts(balances),
	}
	for _, account := range report.Accounts {
		report.TotalDebit += account.Debit
		report.TotalCredit += account.Credit
	}
	report.TotalDebit = round2(report.TotalDebit)
	report.TotalCredit = round2(report.TotalCredit)
	report.Balanced = math.Abs(report.TotalDebit-report.TotalCredit) <= reportTolerance
	return report, nil
}

// GeneralLedger (Buku Besar) menampilkan mutasi satu akun pada rentang tanggal,
// diawali saldo akumulatif sebelum rentang dan diakhiri saldo akhir. Saldo
// berjalan bertanda sesuai saldo normal akun sehingga arah angka konsisten
// dengan tampilan saldo lainnya.
func (u *reportUsecase) GeneralLedger(ctx context.Context, accountID int64, from, to time.Time) (*domain.GeneralLedger, error) {
	if err := validateRange(from, to); err != nil {
		return nil, err
	}
	if accountID <= 0 {
		return nil, domain.ErrAccountNotFound
	}

	account, err := u.coa.GetByID(ctx, accountID)
	if err != nil {
		return nil, notFoundAs(err, domain.ErrAccountNotFound)
	}

	fromDate, toDate := reportDate(from), reportDate(to)
	openingDebit, openingCredit, err := u.repo.AccountOpeningBalance(ctx, accountID, fromDate)
	if err != nil {
		return nil, err
	}
	lines, err := u.repo.AccountLedgerLines(ctx, accountID, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	running := saldoTertanda(account.NormalBalance, openingDebit, openingCredit)
	ledger := &domain.GeneralLedger{
		Account: domain.ReportAccount{
			AccountID:   account.ID,
			Code:        account.Code,
			Name:        account.Name,
			AccountType: account.AccountType,
		},
		From:           fromDate,
		To:             toDate,
		OpeningBalance: round2(running),
		Lines:          make([]domain.GeneralLedgerLine, 0, len(lines)),
	}

	for _, line := range lines {
		running += saldoTertanda(account.NormalBalance, line.Debit, line.Credit)
		ledger.Lines = append(ledger.Lines, domain.GeneralLedgerLine{
			EntryNumber: line.EntryNumber,
			EntryDate:   line.EntryDate,
			Description: line.Description,
			Debit:       round2(line.Debit),
			Credit:      round2(line.Credit),
			Balance:     round2(running),
		})
		ledger.TotalDebit += line.Debit
		ledger.TotalCredit += line.Credit
	}

	ledger.TotalDebit = round2(ledger.TotalDebit)
	ledger.TotalCredit = round2(ledger.TotalCredit)
	ledger.ClosingBalance = round2(running)
	ledger.Account.Debit = ledger.TotalDebit
	ledger.Account.Credit = ledger.TotalCredit
	ledger.Account.Balance = ledger.ClosingBalance
	return ledger, nil
}
