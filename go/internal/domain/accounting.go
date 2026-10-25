package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Tipe akun pada bagan akun. Nilai dipaksa juga oleh CHECK constraint
// m_chart_of_accounts_balance_check agar saldo normal selaras dengan tipe.
const (
	AccountTypeAsset     = "ASSET"
	AccountTypeLiability = "LIABILITY"
	AccountTypeEquity    = "EQUITY"
	AccountTypeRevenue   = "REVENUE"
	AccountTypeExpense   = "EXPENSE"
)

// Saldo normal akun. Aset & Beban → DEBIT; Kewajiban, Ekuitas, Pendapatan → CREDIT.
const (
	NormalBalanceDebit  = "DEBIT"
	NormalBalanceCredit = "CREDIT"
)

// Status dokumen jurnal umum.
const (
	JournalStatusDraft   = "DRAFT"
	JournalStatusPosted  = "POSTED"
	JournalStatusVoid    = "VOID"
	JournalSourceManual  = "MANUAL"
	JournalSourceGRN     = "GRN"
	JournalSourcePayment = "SUPPLIER_PAYMENT"
	JournalSourceCashIn  = "CASH_IN"
	JournalSourceCashOut = "CASH_OUT"
	JournalSourceClosing = "CLOSING"
)

// Status periode fiskal.
const (
	PeriodStatusOpen   = "OPEN"
	PeriodStatusClosed = "CLOSED"
)

// ChartOfAccount melambangkan satu baris bagan akun.
type ChartOfAccount struct {
	ID            int64      `json:"id"`
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	AccountType   string     `json:"account_type"`
	NormalBalance string     `json:"normal_balance"`
	ParentID      *int64     `json:"parent_id,omitempty"`
	ParentCode    string     `json:"parent_code,omitempty"`
	ParentName    string     `json:"parent_name,omitempty"`
	IsHeader      bool       `json:"is_header"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	CreatedBy     *int64     `json:"created_by,omitempty"`
	UpdatedBy     *int64     `json:"updated_by,omitempty"`
	DeletedBy     *int64     `json:"deleted_by,omitempty"`
}

// ChartOfAccountParam adalah parameter kueri cursor bagan akun.
type ChartOfAccountParam struct {
	Search    string
	CursorID  int64
	Limit     int32
	Direction string // "next" atau "prev"
}

// ChartOfAccountRepository mengatur penyimpanan bagan akun.
type ChartOfAccountRepository interface {
	Create(ctx context.Context, acc *ChartOfAccount) error
	GetListCursor(ctx context.Context, param ChartOfAccountParam) ([]ChartOfAccount, int64, error)
	GetByID(ctx context.Context, id int64) (*ChartOfAccount, error)
	GetByIDs(ctx context.Context, ids []int64) ([]ChartOfAccount, error)
	Update(ctx context.Context, acc *ChartOfAccount) error
	Delete(ctx context.Context, id int64, deletedBy int64) error
}

// AccountingPeriod mewakili satu periode fiskal (bulan-tahun).
type AccountingPeriod struct {
	ID          int64      `json:"id"`
	PeriodYear  int        `json:"period_year"`
	PeriodMonth int        `json:"period_month"`
	Status      string     `json:"status"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	ClosedBy    *int64     `json:"closed_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// AccountingTx adalah permukaan sempit operasi akuntansi yang harus berjalan
// dalam satu transaksi database. Dipakai oleh tutup/buka buku agar jurnal
// penutup, pembalik, dan perubahan status periode tidak pernah terpisah.
type AccountingTx interface {
	LockPeriod(ctx context.Context, year, month int) (*AccountingPeriod, error)
	CreatePeriod(ctx context.Context, year, month int, actorID int64) (*AccountingPeriod, error)
	AccountBalances(ctx context.Context, from, to time.Time) ([]AccountBalance, error)
	GetAccountsByIDs(ctx context.Context, ids []int64) ([]ChartOfAccount, error)
	ListAccountingSettings(ctx context.Context) (map[string]int64, error)
	NextJournalSeq(ctx context.Context, prefix string) (int64, error)
	GetPostedJournalIDBySource(ctx context.Context, sourceType string, sourceID int64) (int64, error)
	CreatePostedJournal(ctx context.Context, entry *JournalEntry) error
	ReverseJournal(ctx context.Context, originalID int64, entryDate time.Time, reason string, actorID int64) (*JournalEntry, error)
	ClosePeriod(ctx context.Context, year, month, actorID int64) (int64, error)
	ReopenPeriod(ctx context.Context, year, month int) (int64, error)
}

// AccountingPeriodRepository mengatur periode fiskal.
type AccountingPeriodRepository interface {
	Get(ctx context.Context, year, month int) (*AccountingPeriod, error)
	EnsureOpen(ctx context.Context, year, month int, actorID int64) (*AccountingPeriod, error)
	Close(ctx context.Context, year, month, actorID int64) (int64, error)
	Reopen(ctx context.Context, year, month int) (int64, error)
	List(ctx context.Context, limit int) ([]AccountingPeriod, error)
	// WithTx menjalankan fn dalam satu transaksi dan menyediakan operasi
	// akuntansi yang terikat pada transaksi tersebut.
	WithTx(ctx context.Context, fn func(tx AccountingTx) error) error
}

// AccountingSetting adalah satu baris pemetaan kunci transaksi ke akun CoA.
// Nilai account_id dipakai auto-posting ketika menerbitkan jurnal otomatis.
type AccountingSetting struct {
	MappingKey  string `json:"mapping_key"`
	AccountID   int64  `json:"account_id"`
	Description string `json:"description,omitempty"`
}

// AccountingSettingsRepository menyediakan pemetaan akun untuk auto-posting.
type AccountingSettingsRepository interface {
	// ListMapping mengembalikan seluruh pemetaan mapping_key → account_id yang
	// tersimpan di s_accounting_settings.
	ListMapping(ctx context.Context) (map[string]int64, error)
}

// JournalLine mewakili satu baris debit/kredit jurnal.
type JournalLine struct {
	ID          int64   `json:"id"`
	LineNo      int     `json:"line_no"`
	AccountID   int64   `json:"account_id"`
	AccountCode string  `json:"account_code,omitempty"`
	AccountName string  `json:"account_name,omitempty"`
	Description string  `json:"description,omitempty"`
	Debit       float64 `json:"debit"`
	Credit      float64 `json:"credit"`
}

// JournalEntry mewakili satu jurnal umum dan baris-barisnya.
type JournalEntry struct {
	ID              int64         `json:"id"`
	EntryNumber     string        `json:"entry_number"`
	EntryDate       time.Time     `json:"entry_date"`
	Description     string        `json:"description,omitempty"`
	SourceType      string        `json:"source_type,omitempty"`
	SourceID        *int64        `json:"source_id,omitempty"`
	SourceReference string        `json:"source_reference,omitempty"`
	Status          string        `json:"status"`
	TotalDebit      float64       `json:"total_debit"`
	TotalCredit     float64       `json:"total_credit"`
	PeriodYear      int           `json:"period_year"`
	PeriodMonth     int           `json:"period_month"`
	PostedAt        *time.Time    `json:"posted_at,omitempty"`
	PostedBy        *int64        `json:"posted_by,omitempty"`
	VoidedAt        *time.Time    `json:"voided_at,omitempty"`
	VoidedBy        *int64        `json:"voided_by,omitempty"`
	VoidReason      string        `json:"void_reason,omitempty"`
	ReversalOfID    *int64        `json:"reversal_of_id,omitempty"`
	ReversedByID    *int64        `json:"reversed_by_id,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	CreatedBy       *int64        `json:"created_by,omitempty"`
	Lines           []JournalLine `json:"lines,omitempty"`
}

// JournalEntryParam adalah parameter kueri cursor jurnal.
type JournalEntryParam struct {
	Search       string
	StatusFilter string
	CursorID     int64
	Limit        int32
	Direction    string // "next" atau "prev"
}

// JournalCreateInput adalah payload pembuatan jurnal manual (double-entry).
type JournalCreateInput struct {
	EntryDate   time.Time
	Description string
	Lines       []JournalLineInput
}

// JournalLineInput adalah satu baris debit/kredit input dari klien.
// Debit dan kredit tidak boleh keduanya terisi pada baris yang sama.
type JournalLineInput struct {
	AccountID   int64
	Description string
	Debit       float64
	Credit      float64
}

// JournalRepository mengatur penyimpanan jurnal umum.
type JournalRepository interface {
	// CreateDraft menulis header + seluruh baris jurnal berstatus DRAFT dalam
	// satu transaksi. entry_number sudah ditetapkan pemanggil (hasil penghitung).
	CreateDraft(ctx context.Context, entry *JournalEntry) error
	// CreatePosted menulis header + seluruh baris jurnal yang LANGSUNG berstatus
	// POSTED dalam satu transaksi. Dipakai auto-posting sehingga kegagalan di
	// tengah tidak pernah meninggalkan jurnal DRAFT yang menggantung.
	//
	// Idempoten pada (source_type, source_id): bila sudah ada jurnal POSTED untuk
	// sumber yang sama, mengembalikan ErrJournalAlreadyPost dan tidak menulis
	// apa pun. Bila nomor jurnal sudah dipakai, mengembalikan ErrNomorJurnalBentrok
	// sehingga pemanggil boleh mencoba nomor berikutnya.
	CreatePosted(ctx context.Context, entry *JournalEntry) error
	GetListCursor(ctx context.Context, param JournalEntryParam) ([]JournalEntry, int64, error)
	GetByID(ctx context.Context, id int64) (*JournalEntry, error)
	// Post memindahkan DRAFT → POSTED secara atomik. Mengembalikan 0 baris bila
	// jurnal bukan DRAFT lagi, periode tertutup, atau ada baris ke akun non-posting.
	Post(ctx context.Context, id, actorID int64) (int64, error)
	// Void membatalkan jurnal yang masih DRAFT (belum pernah diposting).
	Void(ctx context.Context, id, actorID int64, reason string) (int64, error)
	// Reverse membuat jurnal pembalik (posisi debit/kredit dibalik) dan langsung
	// mempostingnya, lalu menandai jurnal asli sebagai sudah dibalik. Seluruh
	// proses dalam satu transaksi. Mengembalikan jurnal pembalik yang terbit.
	Reverse(ctx context.Context, originalID int64, entryDate time.Time, reason string, actorID int64) (*JournalEntry, error)
	NextSeq(ctx context.Context, prefix string) (int64, error)
	// GetPostedIDBySource mengembalikan id jurnal POSTED yang belum dibalik untuk
	// pasangan (source_type, source_id). Mengembalikan pgx.ErrNoRows bila tidak
	// ada. Dipakai untuk menemukan jurnal penutup sebuah periode saat periode
	// dibuka kembali.
	GetPostedIDBySource(ctx context.Context, sourceType string, sourceID int64) (int64, error)
}

// ChartOfAccountUsecase menetapkan aturan bisnis bagan akun.
type ChartOfAccountUsecase interface {
	CreateAccount(ctx context.Context, acc *ChartOfAccount, userID int64) error
	FindAllAccounts(ctx context.Context, param ChartOfAccountParam) ([]ChartOfAccount, int64, error)
	FindAccountByID(ctx context.Context, id int64) (*ChartOfAccount, error)
	UpdateAccount(ctx context.Context, acc *ChartOfAccount, userID int64) error
	DeleteAccount(ctx context.Context, id int64, userID int64) error
}

// AccountingPeriodUsecase menetapkan aturan bisnis periode fiskal.
type AccountingPeriodUsecase interface {
	GetPeriod(ctx context.Context, year, month int64) (*AccountingPeriod, error)
	OpenPeriod(ctx context.Context, year, month, userID int64) (*AccountingPeriod, error)
	ClosePeriod(ctx context.Context, year, month, userID int64) error
	ReopenPeriod(ctx context.Context, year, month int, userID int64) error
	ListPeriods(ctx context.Context, limit int) ([]AccountingPeriod, error)
}

// JournalUsecase menetapkan aturan bisnis jurnal umum.
type JournalUsecase interface {
	CreateJournal(ctx context.Context, in JournalCreateInput, userID int64) (*JournalEntry, error)
	FindAllJournals(ctx context.Context, param JournalEntryParam) ([]JournalEntry, int64, error)
	FindJournalByID(ctx context.Context, id int64) (*JournalEntry, error)
	PostJournal(ctx context.Context, id int64, userID int64) error
	VoidJournal(ctx context.Context, id int64, userID int64, reason string) error
	ReverseJournal(ctx context.Context, id int64, userID int64, reason string) error
	// PostGRNJournal menerbitkan jurnal otomatis penerimaan barang untuk satu
	// GRN: Debit Persediaan + Debit PPN Masukan (bila ada), Kredit Hutang Usaha.
	// Nilai dihitung dari harga baris PO dan tipe pajak PO, bukan dari GRN.
	//
	// Idempoten: GRN yang sudah punya jurnal POSTED akan dilewati tanpa error,
	// sehingga penutupan penerimaan bisa diulang tanpa mencatat dua kali.
	PostGRNJournal(ctx context.Context, grn *GoodsReceiptNote, po *PurchaseOrder, actorID int64) error
	// PostGRNJournals adalah versi batch dari PostGRNJournal. Pemetaan akun,
	// validasi akun, dan pengecekan periode dibaca sekali untuk seluruh GRN.
	PostGRNJournals(ctx context.Context, grns []*GoodsReceiptNote, po *PurchaseOrder, actorID int64) error
	// PostClosingEntry menerbitkan jurnal penutup satu periode: seluruh akun
	// PENDAPATAN dan BEBAN disanatkan ke nol, selisihnya (laba/rugi) dipindahkan
	// ke akun Laba Ditahan (pemetaan RETAINED_EARNINGS). Jurnal diberi sumber
	// CLOSING/source_id=id periode sehingga idempoten: periode yang jurnal
	// penutupnya sudah terbit tidak dicatat dua kali.
	PostClosingEntry(ctx context.Context, period *AccountingPeriod, actorID int64) error
	// ReverseClosingEntry membalik jurnal penutup sebuah periode (bila ada)
	// sehingga pembukuan kembali seperti sebelum tutup buku. Dipanggil saat
	// periode dibuka kembali. Tidak melakukan apa pun bila periode belum pernah
	// punya jurnal penutup.
	ReverseClosingEntry(ctx context.Context, period *AccountingPeriod, actorID int64) error
	// PostClosingEntryTx dan ReverseClosingEntryTx adalah varian yang memakai
	// transaksi pemanggil, sehingga status periode dan jurnal tidak terpisah.
	PostClosingEntryTx(ctx context.Context, tx AccountingTx, period *AccountingPeriod, actorID int64) error
	ReverseClosingEntryTx(ctx context.Context, tx AccountingTx, period *AccountingPeriod, actorID int64) error
	// PlanSaleJournal merencanakan jurnal penjualan (bukan menulisnya) agar
	// repository penjualan dapat menuliskannya dalam transaksi nota yang sama.
	PlanSaleJournal(ctx context.Context, sale *Sale, paymentAccounts map[int64]int64, actorID int64) (*SaleJournalPlan, error)
	// PlanSaleReversal merencanakan jurnal pembalik untuk nota yang dibatalkan.
	PlanSaleReversal(ctx context.Context, sale *Sale, reason string, actorID int64) (*SaleJournalPlan, error)
}

// --- Laporan Keuangan ---

// AccountBalance adalah agregat debit/kredit satu akun dari jurnal POSTED dalam
// satu rentang tanggal. Dipakai sebagai bahan mentah Neraca, Laba Rugi, dan
// Neraca Saldo; saldo bertanda dihitung pemanggil sesuai saldo normal akun.
type AccountBalance struct {
	AccountID     int64
	Code          string
	Name          string
	AccountType   string
	NormalBalance string
	TotalDebit    float64
	TotalCredit   float64
}

// ReportAccount adalah satu baris akun di dalam laporan: saldo debit/kredit
// dalam cakupan laporan, plus Balance (sudah bertanda sesuai saldo normal,
// sehingga angka positif selalu berarti sesuai sifat akun).
type ReportAccount struct {
	AccountID   int64   `json:"account_id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	AccountType string  `json:"account_type"`
	Debit       float64 `json:"debit"`
	Credit      float64 `json:"credit"`
	Balance     float64 `json:"balance"`
}

// IncomeStatement (Laba Rugi) membandingkan pendapatan dan beban pada satu
// rentang tanggal.
type IncomeStatement struct {
	From          time.Time       `json:"from"`
	To            time.Time       `json:"to"`
	Revenue       []ReportAccount `json:"revenue"`
	Expenses      []ReportAccount `json:"expenses"`
	TotalRevenue  float64         `json:"total_revenue"`
	TotalExpenses float64         `json:"total_expenses"`
	NetIncome     float64         `json:"net_income"`
}

// BalanceSheet (Neraca) menyajikan Aset, Kewajiban, dan Ekuitas pada satu titik
// waktu. Karena sistem ini tidak menutup buku laba/rugi ke akun ekuitas, sisa
// pendapatan-beban sepanjang masa sampai tanggal laporan disajikan sebagai satu
// baris ekuitas sintetis ("Laba (Rugi) Berjalan") supaya persamaan akuntansi
// tetap seimbang.
type BalanceSheet struct {
	AsOf                      time.Time       `json:"as_of"`
	Assets                    []ReportAccount `json:"assets"`
	Liabilities               []ReportAccount `json:"liabilities"`
	Equity                    []ReportAccount `json:"equity"`
	TotalAssets               float64         `json:"total_assets"`
	TotalLiabilities          float64         `json:"total_liabilities"`
	TotalEquity               float64         `json:"total_equity"`
	TotalLiabilitiesAndEquity float64         `json:"total_liabilities_and_equity"`
	Balanced                  bool            `json:"balanced"`
}

// TrialBalance (Neraca Saldo) mendaftar seluruh akun beraktivitas dengan saldo
// debit/kredit pada rentang tanggal; jumlah keduanya harus sama.
type TrialBalance struct {
	From        time.Time       `json:"from"`
	To          time.Time       `json:"to"`
	Accounts    []ReportAccount `json:"accounts"`
	TotalDebit  float64         `json:"total_debit"`
	TotalCredit float64         `json:"total_credit"`
	Balanced    bool            `json:"balanced"`
}

// GeneralLedgerLine adalah satu baris jurnal POSTED dalam Buku Besar, dengan
// saldo berjalan yang sudah bertanda sesuai saldo normal akun.
type GeneralLedgerLine struct {
	EntryNumber string    `json:"entry_number"`
	EntryDate   time.Time `json:"entry_date"`
	Description string    `json:"description"`
	Debit       float64   `json:"debit"`
	Credit      float64   `json:"credit"`
	Balance     float64   `json:"balance"`
}

// GeneralLedger (Buku Besar) menampilkan mutasi satu akun pada rentang tanggal,
// diawali saldo awal (semua mutasi sebelum rentang) dan diakhiri saldo akhir.
type GeneralLedger struct {
	Account        ReportAccount       `json:"account"`
	From           time.Time           `json:"from"`
	To             time.Time           `json:"to"`
	OpeningBalance float64             `json:"opening_balance"`
	Lines          []GeneralLedgerLine `json:"lines"`
	TotalDebit     float64             `json:"total_debit"`
	TotalCredit    float64             `json:"total_credit"`
	ClosingBalance float64             `json:"closing_balance"`
}

// AccountLedgerLine adalah bahan mentah baris buku besar sebelum saldo berjalan
// dihitung oleh usecase.
type AccountLedgerLine struct {
	EntryNumber string
	EntryDate   time.Time
	Description string
	Debit       float64
	Credit      float64
}

// AccountingReportRepository menyediakan agregat dan mutasi jurnal POSTED untuk
// laporan keuangan.
type AccountingReportRepository interface {
	// AccountBalances menjumlahkan debit/kredit per akun dari jurnal POSTED pada
	// rentang [from, to] inklusif.
	AccountBalances(ctx context.Context, from, to time.Time) ([]AccountBalance, error)
	// AccountLedgerLines mengembalikan mutasi satu akun pada rentang [from, to].
	AccountLedgerLines(ctx context.Context, accountID int64, from, to time.Time) ([]AccountLedgerLine, error)
	// AccountOpeningBalance menjumlahkan debit/kredit satu akun sebelum tanggal
	// "before" (mutasi yang menjadi saldo awal buku besar).
	AccountOpeningBalance(ctx context.Context, accountID int64, before time.Time) (debit, credit float64, err error)
}

// AccountingReportUsecase menetapkan aturan penyusunan laporan keuangan.
type AccountingReportUsecase interface {
	BalanceSheet(ctx context.Context, asOf time.Time) (*BalanceSheet, error)
	IncomeStatement(ctx context.Context, from, to time.Time) (*IncomeStatement, error)
	TrialBalance(ctx context.Context, from, to time.Time) (*TrialBalance, error)
	GeneralLedger(ctx context.Context, accountID int64, from, to time.Time) (*GeneralLedger, error)
}

// --- Error bisnis akuntansi ---

// ErrJournalLineKosong menandai jurnal tanpa baris.
var (
	ErrJournalLineKosong = errors.New("jurnal harus memuat minimal satu baris debit/kredit")
	ErrAccountDuplicate  = errors.New("kode akun sudah dipakai akun lain")

	ErrJournalUnbalanced   = errors.New("jurnal tidak seimbang: jumlah debit harus sama dengan jumlah kredit")
	ErrJournalAccountAdmin = errors.New("jurnal memuat akun header atau akun yang tidak aktif")
	ErrJournalPeriodClosed = errors.New("jurnal ditolak: periode fiskal sudah ditutup")
	ErrJournalNotFound     = errors.New("jurnal tidak ditemukan")
	ErrJournalNotDraft     = errors.New("jurnal tidak dalam status DRAFT")
	ErrJournalAlreadyPost  = errors.New("jurnal sudah diposting")
	ErrJournalAlreadyVoid  = errors.New("jurnal sudah dibatalkan")
	ErrJournalNotPosted    = errors.New("hanya jurnal berstatus POSTED yang dapat dibatalkan")
	ErrVoidPostedJournal   = errors.New("jurnal sudah diposting, gunakan jurnal pembalik untuk membatalkan")
	ErrJournalAlreadyRev   = errors.New("jurnal sudah dibatalkan oleh jurnal pembalik")
	ErrJournalPostBox      = errors.New("jurnal gagal diposting, status atau periode berubah")
	ErrNomorJurnalBentrok  = errors.New("nomor jurnal sudah dipakai jurnal lain")
	ErrPeriodNotFound      = errors.New("periode tidak ditemukan")
	ErrAccountNotFound     = errors.New("akun tidak ditemukan")
	ErrPeriodClosed        = errors.New("periode sudah ditutup")
	ErrPeriodOpen          = errors.New("periode masih terbuka")
	ErrPeriodeInvalid      = errors.New("tahun atau bulan periode tidak valid")
	ErrKodeAkunWajib       = errors.New("kode akun wajib diisi")
	ErrNamaAkunWajib       = errors.New("nama akun wajib diisi")
	ErrTipeAkunInvalid     = errors.New("tipe akun tidak dikenal")
	ErrSaldoNormalMismatch = errors.New("saldo normal tidak sesuai dengan tipe akun")
	ErrIndukAkunNotFound   = errors.New("akun induk tidak ditemukan")
	ErrJournalDateInvalid  = errors.New("tanggal jurnal tidak valid")
	ErrJournalLineDouble   = errors.New("satu baris jurnal tidak boleh memuat debit dan kredit sekaligus")
	ErrPeriodeBerubah      = errors.New("periode sedang berubah oleh permintaan lain, coba lagi")
	// ErrRentangTanggalInvalid dipakai laporan bila tanggal awal melewati tanggal
	// akhir atau salah satu tanggal kosong.
	ErrRentangTanggalInvalid = errors.New("rentang tanggal laporan tidak valid")
	// ErrAutoPostingSettings dipakai bila pemetaan akun di s_accounting_settings
	// belum lengkap sehingga jurnal otomatis tidak bisa disusun.
	ErrAutoPostingSettings = errors.New("pemetaan akun untuk auto-posting belum diatur")
)

// JournalNotPostableError menandai baris jurnal yang menunjuk akun header /
// non-aktif / terhapus. Dipakai handler untuk memetakan ke 422.
type JournalNotPostableError struct {
	AccountID int64
	Code      string
}

func (e *JournalNotPostableError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("akun %s bukan akun posting (header/nonaktif)", e.Code)
	}
	return "jurnal memuat akun yang bukan akun posting"
}
