package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"go_backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

<<<<<<< HEAD:go/internal/usecase/journal_usecase.go
=======
// notFoundAs menerjemahkan "data tidak ditemukan" PostgreSQL menjadi error
// domain yang spesifik agar handler bisa memilih status HTTP yang tepat.
func notFoundAs(err error, fallback error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fallback
	}
	return err
}

// --- Bagan Akun ---

type chartOfAccountUsecase struct {
	repo domain.ChartOfAccountRepository
}

func NewChartOfAccountUsecase(repo domain.ChartOfAccountRepository) domain.ChartOfAccountUsecase {
	return &chartOfAccountUsecase{repo: repo}
}

// balanceForType mengembalikan saldo normal yang wajib untuk suatu tipe akun.
func balanceForType(accountType string) (string, bool) {
	switch accountType {
	case domain.AccountTypeAsset, domain.AccountTypeExpense:
		return domain.NormalBalanceDebit, true
	case domain.AccountTypeLiability, domain.AccountTypeEquity, domain.AccountTypeRevenue:
		return domain.NormalBalanceCredit, true
	}
	return "", false
}

// validateAccount memvalidasi payload akun dan menormalkan nilainya. Dipakai
// bersama oleh pembuatan dan koreksi akun.
func validateAccount(acc *domain.ChartOfAccount) error {
	acc.Code = strings.TrimSpace(strings.ToUpper(acc.Code))
	acc.Name = strings.TrimSpace(acc.Name)
	acc.AccountType = strings.ToUpper(strings.TrimSpace(acc.AccountType))
	acc.NormalBalance = strings.ToUpper(strings.TrimSpace(acc.NormalBalance))

	if acc.Code == "" {
		return domain.ErrKodeAkunWajib
	}
	if len(acc.Code) > 20 {
		return domain.NewBusinessRuleError("kode akun maksimal 20 karakter")
	}
	if acc.Name == "" {
		return domain.ErrNamaAkunWajib
	}
	if len(acc.Name) > 150 {
		return domain.NewBusinessRuleError("nama akun maksimal 150 karakter")
	}
	expected, ok := balanceForType(acc.AccountType)
	if !ok {
		return domain.ErrTipeAkunInvalid
	}
	if acc.NormalBalance == "" {
		acc.NormalBalance = expected
	}
	if acc.NormalBalance != expected {
		return domain.ErrSaldoNormalMismatch
	}
	return nil
}

func (u *chartOfAccountUsecase) CreateAccount(ctx context.Context, acc *domain.ChartOfAccount, userID int64) error {
	if err := validateAccount(acc); err != nil {
		return err
	}
	// Hierarki: induk wajib ada. Pengecekan eksplisit memberikan pesan yang
	// lebih jelas daripada error foreign key mentah dari database.
	if acc.ParentID != nil && *acc.ParentID > 0 {
		parent, err := u.repo.GetByID(ctx, *acc.ParentID)
		if err != nil {
			return notFoundAs(err, domain.ErrIndukAkunNotFound)
		}
		if parent.IsHeader == false {
			return domain.NewBusinessRuleError("akun induk harus akun bertipe header")
		}
	} else {
		acc.ParentID = nil
	}

	acc.CreatedBy = &userID
	acc.UpdatedBy = &userID
	return u.repo.Create(ctx, acc)
}

func (u *chartOfAccountUsecase) FindAllAccounts(ctx context.Context, param domain.ChartOfAccountParam) ([]domain.ChartOfAccount, int64, error) {
	if param.Limit <= 0 {
		param.Limit = 20
	}
	return u.repo.GetListCursor(ctx, param)
}

func (u *chartOfAccountUsecase) FindAccountByID(ctx context.Context, id int64) (*domain.ChartOfAccount, error) {
	acc, err := u.repo.GetByID(ctx, id)
	return acc, notFoundAs(err, domain.ErrAccountNotFound)
}

func (u *chartOfAccountUsecase) UpdateAccount(ctx context.Context, acc *domain.ChartOfAccount, userID int64) error {
	if err := validateAccount(acc); err != nil {
		return err
	}
	if acc.ParentID != nil && *acc.ParentID > 0 {
		parent, err := u.repo.GetByID(ctx, *acc.ParentID)
		if err != nil {
			return notFoundAs(err, domain.ErrIndukAkunNotFound)
		}
		if !parent.IsHeader {
			return domain.NewBusinessRuleError("akun induk harus akun bertipe header")
		}
	} else {
		acc.ParentID = nil
	}

	existing, err := u.repo.GetByID(ctx, acc.ID)
	if err != nil {
		return notFoundAs(err, domain.ErrAccountNotFound)
	}
	_ = existing

	acc.UpdatedBy = &userID
	return u.repo.Update(ctx, acc)
}

func (u *chartOfAccountUsecase) DeleteAccount(ctx context.Context, id int64, userID int64) error {
	if _, err := u.repo.GetByID(ctx, id); err != nil {
		return notFoundAs(err, domain.ErrAccountNotFound)
	}
	return u.repo.Delete(ctx, id, userID)
}

// --- Periode Fiskal ---

type accountingPeriodUsecase struct {
	repo domain.AccountingPeriodRepository
	// closing mengurus jurnal penutup saat periode ditutup/dibuka kembali. Boleh
	// nil; bila nil, penutupan periode hanya mengunci status (tanpa jurnal).
	closing periodClosingManager
}

// periodClosingManager adalah irisan perilaku domain.JournalUsecase yang dipakai
// periode fiskal. Didefinisikan sempit supaya periode tidak bergantung pada
// seluruh permukaan jurnal umum.
type periodClosingManager interface {
	PostClosingEntry(ctx context.Context, period *domain.AccountingPeriod, actorID int64) error
	ReverseClosingEntry(ctx context.Context, period *domain.AccountingPeriod, actorID int64) error
}

func NewAccountingPeriodUsecase(repo domain.AccountingPeriodRepository, closing periodClosingManager) domain.AccountingPeriodUsecase {
	return &accountingPeriodUsecase{repo: repo, closing: closing}
}

func validatePeriode(year, month int) error {
	if year < 2000 || year > 2100 || month < 1 || month > 12 {
		return domain.ErrPeriodeInvalid
	}
	return nil
}

func (u *accountingPeriodUsecase) GetPeriod(ctx context.Context, year, month int64) (*domain.AccountingPeriod, error) {
	if err := validatePeriode(int(year), int(month)); err != nil {
		return nil, err
	}
	period, err := u.repo.Get(ctx, int(year), int(month))
	return period, notFoundAs(err, domain.ErrPeriodNotFound)
}

func (u *accountingPeriodUsecase) OpenPeriod(ctx context.Context, year, month, userID int64) (*domain.AccountingPeriod, error) {
	y, m := int(year), int(month)
	if err := validatePeriode(y, m); err != nil {
		return nil, err
	}

	period, err := u.repo.Get(ctx, y, m)
	if errors.Is(err, pgx.ErrNoRows) {
		created, err := u.repo.EnsureOpen(ctx, y, m, userID)
		if err != nil {
			return nil, err
		}
		return created, nil
	}
	if err != nil {
		return nil, err
	}
	if period.Status == domain.PeriodStatusOpen {
		return period, nil
	}
	rows, err := u.repo.Reopen(ctx, y, m)
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, domain.ErrPeriodeBerubah
	}
	return u.repo.Get(ctx, y, m)
}

func (u *accountingPeriodUsecase) ClosePeriod(ctx context.Context, year, month, userID int64) error {
	y, m := int(year), int(month)
	if err := validatePeriode(y, m); err != nil {
		return err
	}
	period, err := u.repo.Get(ctx, y, m)
	if err != nil {
		return notFoundAs(err, domain.ErrPeriodNotFound)
	}
	if period.Status == domain.PeriodStatusClosed {
		return nil
	}
	// Terbitkan jurnal penutup SEBELUM periode dikunci. Bila ini gagal, periode
	// tetap terbuka sehingga tidak pernah ada periode tertutup tanpa penutupan
	// buku. Jurnal penutup idempoten, jadi pengulangan penutupan aman.
	if u.closing != nil {
		if err := u.closing.PostClosingEntry(ctx, period, userID); err != nil {
			return err
		}
	}
	rows, err := u.repo.Close(ctx, int64(y), int64(m), userID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrPeriodeBerubah
	}
	return nil
}

func (u *accountingPeriodUsecase) ReopenPeriod(ctx context.Context, year, month int, userID int64) error {
	if err := validatePeriode(year, month); err != nil {
		return err
	}
	period, err := u.repo.Get(ctx, year, month)
	if err != nil {
		return notFoundAs(err, domain.ErrPeriodNotFound)
	}
	if period.Status != domain.PeriodStatusOpen {
		rows, err := u.repo.Reopen(ctx, year, month)
		if err != nil {
			return err
		}
		if rows == 0 {
			return domain.ErrPeriodeBerubah
		}
		period.Status = domain.PeriodStatusOpen
	}
	// Batalkan jurnal penutup periode ini (bila ada) supaya pembukuan kembali
	// seperti sebelum tutup buku. Selalu dicoba, termasuk saat periode memang
	// sudah OPEN, agar percobaan ulang setelah kegagalan parsial tetap beres.
	if u.closing != nil {
		if err := u.closing.ReverseClosingEntry(ctx, period, userID); err != nil {
			return err
		}
	}
	return nil
}

func (u *accountingPeriodUsecase) ListPeriods(ctx context.Context, limit int) ([]domain.AccountingPeriod, error) {
	return u.repo.List(ctx, limit)
}

// --- Jurnal Umum ---

>>>>>>> e47ff22 (refactor(go): sentralisasi pemetaan error domain ke handler.WriteError):go/internal/usecase/accounting_usecase.go
type journalUsecase struct {
	repo        domain.JournalRepository
	accountRepo domain.ChartOfAccountRepository
	periodRepo  domain.AccountingPeriodRepository
	settings    domain.AccountingSettingsRepository
	reportRepo  domain.AccountingReportRepository
}

func NewJournalUsecase(repo domain.JournalRepository, accountRepo domain.ChartOfAccountRepository, periodRepo domain.AccountingPeriodRepository, settings domain.AccountingSettingsRepository, reportRepo domain.AccountingReportRepository) domain.JournalUsecase {
	return &journalUsecase{repo: repo, accountRepo: accountRepo, periodRepo: periodRepo, settings: settings, reportRepo: reportRepo}
}

// accountIDsFromLines mengumpulkan id akun dari baris jurnal.
func accountIDsFromLines(lines []domain.JournalLineInput) []int64 {
	ids := make([]int64, 0, len(lines))
	for _, l := range lines {
<<<<<<< HEAD:go/internal/usecase/journal_usecase.go
=======
		if l.AccountID <= 0 {
			return nil, domain.NewBusinessRuleError("akun pada baris jurnal tidak valid")
		}
>>>>>>> e47ff22 (refactor(go): sentralisasi pemetaan error domain ke handler.WriteError):go/internal/usecase/accounting_usecase.go
		ids = append(ids, l.AccountID)
	}
	return ids
}

// validatePostableAccountsFrom memastikan seluruh akun baris jurnal ada dan
// layak diposting (bukan header, aktif, tidak terhapus).
func validatePostableAccountsFrom(accounts []domain.ChartOfAccount, lines []domain.JournalLineInput) (map[int64]domain.ChartOfAccount, error) {
	byID := make(map[int64]domain.ChartOfAccount, len(accounts))
	for _, a := range accounts {
		byID[a.ID] = a
	}
	postable := make(map[int64]domain.ChartOfAccount, len(lines))
	for _, l := range lines {
		if l.AccountID <= 0 {
			return nil, domain.NewBusinessRuleError("akun pada baris jurnal tidak valid")
		}
		acc, ok := byID[l.AccountID]
		if !ok {
			return nil, &domain.JournalNotPostableError{AccountID: l.AccountID}
		}
		if !acc.IsActive || acc.IsHeader {
			return nil, &domain.JournalNotPostableError{AccountID: acc.ID, Code: acc.Code}
		}
		postable[l.AccountID] = acc
	}
	return postable, nil
}

// validatePostableAccounts memastikan seluruh akun baris jurnal ada dan layak
// diposting (bukan header, aktif, tidak terhapus). Mengembalikan peta id→akun.
func (u *journalUsecase) validatePostableAccounts(ctx context.Context, lines []domain.JournalLineInput) (map[int64]domain.ChartOfAccount, error) {
	for _, l := range lines {
		if l.AccountID <= 0 {
			return nil, domain.NewBusinessRuleError("akun pada baris jurnal tidak valid")
		}
	}
	accounts, err := u.accountRepo.GetByIDs(ctx, accountIDsFromLines(lines))
	if err != nil {
		return nil, err
	}
	return validatePostableAccountsFrom(accounts, lines)
}

func (u *journalUsecase) CreateJournal(ctx context.Context, in domain.JournalCreateInput, userID int64) (*domain.JournalEntry, error) {
	if in.EntryDate.IsZero() {
		return nil, domain.ErrJournalDateInvalid
	}
	if len(in.Lines) == 0 {
		return nil, domain.ErrJournalLineKosong
	}

	if _, err := u.validatePostableAccounts(ctx, in.Lines); err != nil {
		return nil, err
	}

	var totalDebit, totalCredit float64
	for i, l := range in.Lines {
		if l.Debit < 0 || l.Credit < 0 {
			return nil, domain.NewBusinessRuleError("debit/kredit tidak boleh negatif")
		}
		if l.Debit > 0 && l.Credit > 0 {
			return nil, domain.ErrJournalLineDouble
		}
		if l.Debit == 0 && l.Credit == 0 {
			return nil, domain.NewBusinessRuleError("baris ke-%d: debit atau kredit wajib diisi", i+1)
		}
		totalDebit += l.Debit
		totalCredit += l.Credit
	}
	if math.Abs(totalDebit-totalCredit) > 0.005 {
		return nil, domain.ErrJournalUnbalanced
	}

	period, err := u.periodRepo.EnsureOpen(ctx, in.EntryDate.Year(), int(in.EntryDate.Month()), userID)
	if err != nil {
		return nil, err
	}

	datePrefix := in.EntryDate.Format("20060102")
	const maksPercobaanNomor = 5
	for attempt := 1; attempt <= maksPercobaanNomor; attempt++ {
		seq, err := u.repo.NextSeq(ctx, datePrefix)
		if err != nil {
			return nil, err
		}

		entry := &domain.JournalEntry{
			EntryNumber: fmt.Sprintf("JV/%s/%04d", datePrefix, seq),
			EntryDate:   in.EntryDate,
			Description: strings.TrimSpace(in.Description),
			SourceType:  domain.JournalSourceManual,
			Status:      domain.JournalStatusDraft,
			TotalDebit:  totalDebit,
			TotalCredit: totalCredit,
			PeriodYear:  in.EntryDate.Year(),
			PeriodMonth: int(in.EntryDate.Month()),
			CreatedBy:   &userID,
			Lines:       make([]domain.JournalLine, 0, len(in.Lines)),
		}
		for _, l := range in.Lines {
			entry.Lines = append(entry.Lines, domain.JournalLine{
				LineNo:      len(entry.Lines) + 1,
				AccountID:   l.AccountID,
				Description: l.Description,
				Debit:       l.Debit,
				Credit:      l.Credit,
			})
		}

		err = u.repo.CreateDraft(ctx, entry)
		if err == nil {
			_ = period
			loaded, loadErr := u.repo.GetByID(ctx, entry.ID)
			if loadErr != nil {
				return nil, loadErr
			}
			return loaded, nil
		}
		if !errors.Is(err, domain.ErrNomorJurnalBentrok) {
			return nil, err
		}
		if attempt == maksPercobaanNomor {
			return nil, domain.NewBusinessRuleError("jurnal ditolak: nomor dokumen bentrok berulang, mohon hubungi administrator")
		}
	}
	return nil, domain.NewBusinessRuleError("jurnal ditolak: gagal menerbitkan nomor dokumen")
}

func (u *journalUsecase) FindAllJournals(ctx context.Context, param domain.JournalEntryParam) ([]domain.JournalEntry, int64, error) {
	if param.Limit <= 0 {
		param.Limit = 10
	}
	if param.StatusFilter == "" {
		param.StatusFilter = "ALL"
	}
	return u.repo.GetListCursor(ctx, param)
}

func (u *journalUsecase) FindJournalByID(ctx context.Context, id int64) (*domain.JournalEntry, error) {
	entry, err := u.repo.GetByID(ctx, id)
	return entry, notFoundAs(err, domain.ErrJournalNotFound)
}

func (u *journalUsecase) PostJournal(ctx context.Context, id int64, userID int64) error {
	entry, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return notFoundAs(err, domain.ErrJournalNotFound)
	}
	switch entry.Status {
	case domain.JournalStatusPosted:
		return domain.ErrJournalAlreadyPost
	case domain.JournalStatusVoid:
		return domain.ErrJournalAlreadyVoid
	}

	// Buka periode sasaran; jurnal DRAFT di periode CLOSED tidak boleh diposting.
	period, err := u.periodRepo.Get(ctx, entry.PeriodYear, entry.PeriodMonth)
	if err != nil {
		return notFoundAs(err, domain.ErrPeriodNotFound)
	}
	if period.Status == domain.PeriodStatusClosed {
		return domain.ErrJournalPeriodClosed
	}

	if _, err := u.validatePostableAccounts(ctx, toLineInputs(&entry.Lines)); err != nil {
		return err
	}

	rows, err := u.repo.Post(ctx, id, userID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrJournalPostBox
	}
	return nil
}

func (u *journalUsecase) VoidJournal(ctx context.Context, id int64, userID int64, reason string) error {
	entry, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return notFoundAs(err, domain.ErrJournalNotFound)
	}
	switch entry.Status {
	case domain.JournalStatusPosted:
		return domain.ErrVoidPostedJournal
	case domain.JournalStatusVoid:
		return domain.ErrJournalAlreadyVoid
	}
	rows, err := u.repo.Void(ctx, id, userID, reason)
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrJournalNotDraft
	}
	return nil
}

func (u *journalUsecase) ReverseJournal(ctx context.Context, id int64, userID int64, reason string) error {
	_, err := u.repo.Reverse(ctx, id, time.Now(), reason, userID)
	return err
}

// toLineInputs mengubah baris jurnal domain menjadi input ringan untuk validasi akun.
func toLineInputs(lines *[]domain.JournalLine) []domain.JournalLineInput {
	inputs := make([]domain.JournalLineInput, 0, len(*lines))
	for _, l := range *lines {
		inputs = append(inputs, domain.JournalLineInput{
			AccountID:   l.AccountID,
			Description: l.Description,
			Debit:       l.Debit,
			Credit:      l.Credit,
		})
	}
	return inputs
}

// --- Auto-posting penerimaan barang (GRN) ---

// round2 membulatkan nilai ke dua desimal. Perlu dilakukan di aplikasi karena
// baris debit dan kredit disimpan sebagai NUMERIC(15,2): bila dibulatkan
// masing-masing oleh database, jumlah baris bisa menjauh dari total header yang
// ikut disimpan.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// requireMapping mengambil akun hasil pemetaan auto-posting. Error menyebut kunci
// yang kurang supaya pengguna langsung tahu baris pengaturan mana yang harus diisi.
func requireMapping(mapping map[string]int64, key string) (int64, error) {
	if id, ok := mapping[key]; ok && id > 0 {
		return id, nil
	}
	return 0, fmt.Errorf("%w: kunci %s", domain.ErrAutoPostingSettings, key)
}

// grnReceivedValues menghitung nilai penerimaan satu GRN dalam rupiah: DPP dan PPN.
// Harga selalu diambil dari baris PO karena GRN tidak memuat harga dan harga beli
// hanya boleh berasal dari PO yang sudah disetujui. Pemidahan PPN memakai matriks
// pajak dan taxRatePPN yang sama dengan penerbitan PO, sehingga jurnal otomatis
// tidak pernah menghasilkan angka yang berbeda dari angka PO untuk penerimaan penuh.
func grnReceivedValues(grn *domain.GoodsReceiptNote, po *domain.PurchaseOrder) (dpp, vat float64, err error) {
	if len(grn.Items) == 0 {
		return 0, 0, domain.ErrGRNPenerimaanKosong
	}

	priceByDetail := make(map[int64]float64, len(po.Items))
	for _, item := range po.Items {
		priceByDetail[item.ID] = item.UnitPrice
	}

	var gross float64
	for _, g := range grn.Items {
		if g.QtyReceived <= 0 {
			continue
		}
		price, ok := priceByDetail[g.PurchaseOrderDetailID]
		if !ok {
			return 0, 0, domain.ErrGRNBarisPONotFound
		}
		gross += float64(g.QtyReceived) * price
	}

	switch po.TaxType {
	case "TAX_EXCLUSIVE":
		dpp = gross
		vat = math.Round(dpp * taxRatePPN)
	case "TAX_INCLUSIVE":
		total := gross
		dpp = math.Round(total / (1.0 + taxRatePPN))
		vat = total - dpp
	default: // NON_PPN
		dpp = gross
		vat = 0
	}
	return round2(dpp), round2(vat), nil
}

// PostGRNJournal menerbitkan jurnal otomatis satu GRN: Debit Persediaan,
// Debit PPN Masukan (bila ada), Kredit Hutang Usaha.
//
// Pemetaan akun dan keterbukaan periode diperiksa SEBELUM menulis apa pun, dan
// penomoran jurnal diputar dengan mekanisme retry yang sama seperti jurnal
// manual. Idempotensi diserahkan ke unique index parsial (source_type, source_id)
// sehingga penutupan penerimaan yang diulang tidak pernah mencatat dua kali.
func (u *journalUsecase) PostGRNJournal(ctx context.Context, grn *domain.GoodsReceiptNote, po *domain.PurchaseOrder, actorID int64) error {
	if grn == nil || grn.ID <= 0 {
		return domain.NewBusinessRuleError("jurnal penerimaan ditolak: grn tidak valid")
	}
	if po == nil || po.ID <= 0 {
		return domain.NewBusinessRuleError("jurnal penerimaan ditolak: po tidak valid")
	}

	dpp, vat, err := grnReceivedValues(grn, po)
	if err != nil {
		return err
	}
	if round2(dpp+vat) <= 0 {
		// Tidak ada nilai ekonomis, sehingga tidak ada jurnal yang pantas terbit.
		return nil
	}

	var mapping map[string]int64
	if u.settings != nil {
		mapping, err = u.settings.ListMapping(ctx)
		if err != nil {
			return err
		}
	}
	inventoryID, err := requireMapping(mapping, "INVENTORY")
	if err != nil {
		return err
	}
	apID, err := requireMapping(mapping, "AP")
	if err != nil {
		return err
	}
	var vatInputID int64
	if vat > 0 {
		if vatInputID, err = requireMapping(mapping, "VAT_INPUT"); err != nil {
			return err
		}
	}

	entryDate := grn.ReceivedDate
	if entryDate.IsZero() {
		entryDate = time.Now()
	}

	lines := []domain.JournalLine{
		{
			LineNo:      1,
			AccountID:   inventoryID,
			Description: fmt.Sprintf("Persediaan masuk - %s", grn.GRNNumber),
			Debit:       dpp,
		},
	}
	if vat > 0 {
		lines = append(lines, domain.JournalLine{
			LineNo:      len(lines) + 1,
			AccountID:   vatInputID,
			Description: fmt.Sprintf("PPN masukan - %s", grn.GRNNumber),
			Debit:       vat,
		})
	}
	// Total kredit dijumlahkan dari nilai yang sudah dibulatkan per baris supaya
	// persis sama dengan total debit.
	ap := dpp + vat
	lines = append(lines, domain.JournalLine{
		LineNo:      len(lines) + 1,
		AccountID:   apID,
		Description: fmt.Sprintf("Hutang usaha - PO %s", po.PoNumber),
		Credit:      ap,
	})

	if _, err := u.validatePostableAccounts(ctx, toLineInputs(&lines)); err != nil {
		return err
	}
	if _, err := u.periodRepo.EnsureOpen(ctx, entryDate.Year(), int(entryDate.Month()), actorID); err != nil {
		return err
	}

	entry := &domain.JournalEntry{
		EntryDate:       entryDate,
		Description:     fmt.Sprintf("Penerimaan barang %s dari PO %s", grn.GRNNumber, po.PoNumber),
		SourceType:      domain.JournalSourceGRN,
		SourceID:        &grn.ID,
		SourceReference: grn.GRNNumber,
		Status:          domain.JournalStatusPosted,
		TotalDebit:      ap,
		TotalCredit:     ap,
		PeriodYear:      entryDate.Year(),
		PeriodMonth:     int(entryDate.Month()),
		CreatedBy:       &actorID,
		Lines:           lines,
	}

	datePrefix := entryDate.Format("20060102")
	const maksPercobaanNomor = 5
	for attempt := 1; attempt <= maksPercobaanNomor; attempt++ {
		seq, err := u.repo.NextSeq(ctx, datePrefix)
		if err != nil {
			return err
		}
		entry.EntryNumber = fmt.Sprintf("JV/%s/%04d", datePrefix, seq)

		err = u.repo.CreatePosted(ctx, entry)
		if err == nil {
			return nil
		}
		// GRN ini sudah punya jurnal POSTED; perulangan penutupan penerimaan
		// tidak boleh mencatat persediaan untuk kedua kalinya.
		if errors.Is(err, domain.ErrJournalAlreadyPost) {
			return nil
		}
		if !errors.Is(err, domain.ErrNomorJurnalBentrok) {
			return err
		}
		if attempt == maksPercobaanNomor {
			return domain.NewBusinessRuleError("jurnal penerimaan gagal diterbitkan: nomor dokumen bentrok berulang, mohon hubungi administrator")
		}
	}
	return domain.NewBusinessRuleError("jurnal penerimaan gagal diterbitkan: gagal menerbitkan nomor dokumen")
}

// --- Jurnal penutup (closing entry) ---

// periodEndDate mengembalikan hari terakhir bulan periode. Go menormalkan
// "hari ke-0 bulan berikutnya" menjadi hari terakhir bulan ini.
func periodEndDate(year, month int) time.Time {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
}

// PostClosingEntry menerbitkan jurnal penutup satu periode:
//   - setiap akun PENDAPATAN dan BEBAN disanatkan (di-nol-kan) menurut saldo
//     normalnya, sehingga saldo akun temporer tidak lagi menumpuk antar periode;
//   - selisihnya (laba bila kredit, rugi bila debit) dipindahkan ke akun Laba
//     Ditahan dari pemetaan RETAINED_EARNINGS.
//
// Saldo diambil KUMULATIF sampai akhir periode, bukan hanya mutasi bulan itu.
// Ini aman karena periode yang sudah ditutup telah menihilkan akun temporernya,
// sehingga kumulatif sampai periode ini praktis hanya memuat laba/rugi periode
// berjalan — sekaligus menangani periode yang sempat dilewati.
//
// Jurnal diberi sumber CLOSING dengan source_id = id periode; unique index
// parsial (source_type, source_id) menjadikannya idempoten sehingga penutupan
// periode bisa diulang tanpa mencatat dua kali.
type closingEntryStore interface {
	AccountBalances(ctx context.Context, from, to time.Time) ([]domain.AccountBalance, error)
	AccountsByIDs(ctx context.Context, ids []int64) ([]domain.ChartOfAccount, error)
	AccountingMapping(ctx context.Context) (map[string]int64, error)
}

type repositoryClosingEntryStore struct {
	u *journalUsecase
}

func (s repositoryClosingEntryStore) AccountBalances(ctx context.Context, from, to time.Time) ([]domain.AccountBalance, error) {
	return s.u.reportRepo.AccountBalances(ctx, from, to)
}

func (s repositoryClosingEntryStore) AccountsByIDs(ctx context.Context, ids []int64) ([]domain.ChartOfAccount, error) {
	return s.u.accountRepo.GetByIDs(ctx, ids)
}

func (s repositoryClosingEntryStore) AccountingMapping(ctx context.Context) (map[string]int64, error) {
	return s.u.closingMapping(ctx)
}

type txClosingEntryStore struct {
	tx domain.AccountingTx
}

func (s txClosingEntryStore) AccountBalances(ctx context.Context, from, to time.Time) ([]domain.AccountBalance, error) {
	return s.tx.AccountBalances(ctx, from, to)
}

func (s txClosingEntryStore) AccountsByIDs(ctx context.Context, ids []int64) ([]domain.ChartOfAccount, error) {
	return s.tx.GetAccountsByIDs(ctx, ids)
}

func (s txClosingEntryStore) AccountingMapping(ctx context.Context) (map[string]int64, error) {
	return s.tx.ListAccountingSettings(ctx)
}

// buildClosingEntry menyusun jurnal penutup dari saldo kumulatif sampai akhir
// periode. Mengembalikan nil bila tidak ada saldo pendapatan/beban yang perlu
// ditutup. Fungsi ini tidak menulis apa pun; pemanggil yang menentukan transaksi.
func buildClosingEntry(ctx context.Context, store closingEntryStore, period *domain.AccountingPeriod, actorID int64) (*domain.JournalEntry, error) {
	if period == nil || period.ID <= 0 {
<<<<<<< HEAD:go/internal/usecase/journal_usecase.go
		return nil, domain.NewBusinessRuleError("jurnal penutup ditolak: periode tidak valid")
=======
		return domain.NewBusinessRuleError("jurnal penutup ditolak: periode tidak valid")
>>>>>>> e47ff22 (refactor(go): sentralisasi pemetaan error domain ke handler.WriteError):go/internal/usecase/accounting_usecase.go
	}
	periodEnd := periodEndDate(period.PeriodYear, period.PeriodMonth)

	balances, err := store.AccountBalances(ctx, reportEpoch, periodEnd)
	if err != nil {
		return nil, err
	}

	description := fmt.Sprintf("Jurnal penutup periode %04d-%02d", period.PeriodYear, period.PeriodMonth)
	lines := make([]domain.JournalLine, 0, len(balances)+1)
	var totalDebit, totalCredit float64
	addLine := func(accountID int64, debit, credit float64) {
		lines = append(lines, domain.JournalLine{
			LineNo:      len(lines) + 1,
			AccountID:   accountID,
			Description: description,
			Debit:       debit,
			Credit:      credit,
		})
		totalDebit += debit
		totalCredit += credit
	}

	for _, b := range balances {
		var amount float64
		switch b.AccountType {
		case domain.AccountTypeRevenue:
			// Saldo kredit pendapatan didebit agar nol; saldo debit (kontra
			// pendapatan) justru dikredit.
			amount = round2(saldoTertanda(b.NormalBalance, b.TotalDebit, b.TotalCredit))
			if amount > 0 {
				addLine(b.AccountID, amount, 0)
			} else if amount < 0 {
				addLine(b.AccountID, 0, -amount)
			}
		case domain.AccountTypeExpense:
			amount = round2(saldoTertanda(b.NormalBalance, b.TotalDebit, b.TotalCredit))
			if amount > 0 {
				addLine(b.AccountID, 0, amount)
			} else if amount < 0 {
				addLine(b.AccountID, -amount, 0)
			}
		}
	}
	if len(lines) == 0 {
		// Tidak ada saldo pendapatan/beban yang perlu ditutup.
		return nil, nil
	}

	// Sisa selisih debit-kredit adalah laba/rugi yang dipindahkan ke Laba Ditahan.
	diff := round2(totalDebit - totalCredit)
	if diff != 0 {
		mapping, err := store.AccountingMapping(ctx)
		if err != nil {
			return nil, err
		}
		retainedID, err := requireMapping(mapping, "RETAINED_EARNINGS")
		if err != nil {
			return nil, err
		}
		if diff > 0 {
			addLine(retainedID, 0, diff)
		} else {
			addLine(retainedID, -diff, 0)
		}
	}

	inputs := toLineInputs(&lines)
	accounts, err := store.AccountsByIDs(ctx, accountIDsFromLines(inputs))
	if err != nil {
		return nil, err
	}
	if _, err := validatePostableAccountsFrom(accounts, inputs); err != nil {
		return nil, err
	}

	return &domain.JournalEntry{
		EntryDate:       periodEnd,
		Description:     description,
		SourceType:      domain.JournalSourceClosing,
		SourceID:        &period.ID,
		SourceReference: fmt.Sprintf("TUTUP BUKU %04d-%02d", period.PeriodYear, period.PeriodMonth),
		Status:          domain.JournalStatusPosted,
		TotalDebit:      round2(totalDebit),
		TotalCredit:     round2(totalCredit),
		PeriodYear:      period.PeriodYear,
		PeriodMonth:     period.PeriodMonth,
		CreatedBy:       &actorID,
		Lines:           lines,
	}, nil
}

// postedClosingEntryExists memeriksa jurnal penutup POSTED yang belum dibalik.
// Pemeriksaan dilakukan SEBELUM menulis agar idempotensi tidak bergantung pada
// pelanggaran unique index, yang akan membatalkan transaksi PostgreSQL.
func postedClosingEntryExists(
	ctx context.Context,
	get func(context.Context, string, int64) (int64, error),
	periodID int64,
) (bool, error) {
	if _, err := get(ctx, domain.JournalSourceClosing, periodID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// writeClosingEntry menerbitkan jurnal penutup dengan percobaan ulang nomor.
func writeClosingEntry(
	ctx context.Context,
	entry *domain.JournalEntry,
	nextSeq func(context.Context, string) (int64, error),
	create func(context.Context, *domain.JournalEntry) error,
) error {
	datePrefix := entry.EntryDate.Format("20060102")
	const maksPercobaanNomor = 5
	for attempt := 1; attempt <= maksPercobaanNomor; attempt++ {
		seq, err := nextSeq(ctx, datePrefix)
		if err != nil {
			return err
		}
		entry.EntryNumber = fmt.Sprintf("JV/%s/%04d", datePrefix, seq)

		err = create(ctx, entry)
		if err == nil {
			return nil
		}
		if !errors.Is(err, domain.ErrNomorJurnalBentrok) {
			return err
		}
		if attempt == maksPercobaanNomor {
			return domain.NewBusinessRuleError("jurnal penutup gagal diterbitkan: nomor dokumen bentrok berulang, mohon hubungi administrator")
		}
	}
	return domain.NewBusinessRuleError("jurnal penutup gagal diterbitkan: gagal menerbitkan nomor dokumen")
<<<<<<< HEAD:go/internal/usecase/journal_usecase.go
}

func (u *journalUsecase) PostClosingEntry(ctx context.Context, period *domain.AccountingPeriod, actorID int64) error {
	if period == nil || period.ID <= 0 {
		return domain.NewBusinessRuleError("jurnal penutup ditolak: periode tidak valid")
	}
	exists, err := postedClosingEntryExists(ctx, u.repo.GetPostedIDBySource, period.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	entry, err := buildClosingEntry(ctx, repositoryClosingEntryStore{u: u}, period, actorID)
	if err != nil {
		return err
	}
	if entry == nil {
		return nil
	}
	if _, err := u.periodRepo.EnsureOpen(ctx, period.PeriodYear, period.PeriodMonth, actorID); err != nil {
		return err
	}
	return writeClosingEntry(ctx, entry, u.repo.NextSeq, u.repo.CreatePosted)
}

// PostClosingEntryTx menerbitkan jurnal penutup di dalam transaksi pemanggil.
// Periode sudah dikunci FOR UPDATE oleh alur tutup buku, jadi fungsi ini tidak
// membuka transaksi baru dan tidak melakukan EnsureOpen terpisah.
func (u *journalUsecase) PostClosingEntryTx(ctx context.Context, tx domain.AccountingTx, period *domain.AccountingPeriod, actorID int64) error {
	if period == nil || period.ID <= 0 {
		return domain.NewBusinessRuleError("jurnal penutup ditolak: periode tidak valid")
	}
	if period.Status != domain.PeriodStatusOpen {
		return domain.ErrJournalPeriodClosed
	}
	exists, err := postedClosingEntryExists(ctx, tx.GetPostedJournalIDBySource, period.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	entry, err := buildClosingEntry(ctx, txClosingEntryStore{tx: tx}, period, actorID)
	if err != nil {
		return err
	}
	if entry == nil {
		return nil
	}
	return writeClosingEntry(ctx, entry, tx.NextJournalSeq, tx.CreatePostedJournal)
=======
>>>>>>> e47ff22 (refactor(go): sentralisasi pemetaan error domain ke handler.WriteError):go/internal/usecase/accounting_usecase.go
}

// ReverseClosingEntry membalik jurnal penutup sebuah periode bila ada, sehingga
// akun pendapatan/beban kembali bersaldo dan Laba Ditahan kembali seperti
// sebelum tutup buku. Dipanggil saat periode dibuka kembali. Tidak melakukan
// apa pun bila periode belum pernah punya jurnal penutup.
func (u *journalUsecase) ReverseClosingEntry(ctx context.Context, period *domain.AccountingPeriod, actorID int64) error {
	if period == nil || period.ID <= 0 {
		return nil
	}
	id, err := u.repo.GetPostedIDBySource(ctx, domain.JournalSourceClosing, period.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	reason := fmt.Sprintf("Pembatalan jurnal penutup periode %04d-%02d", period.PeriodYear, period.PeriodMonth)
	if _, err := u.repo.Reverse(ctx, id, time.Now(), reason, actorID); err != nil {
		// Jurnal penutup sudah dibalik sebelumnya; tidak perlu diulang.
		if errors.Is(err, domain.ErrJournalAlreadyRev) {
			return nil
		}
		return err
	}
	return nil
}

// ReverseClosingEntryTx membalik jurnal penutup di dalam transaksi pemanggil.
// Status periode sudah dibuka kembali di transaksi yang sama, sehingga jurnal
// pembalik selalu tercatat pada periode yang sedang dibuka.
func (u *journalUsecase) ReverseClosingEntryTx(ctx context.Context, tx domain.AccountingTx, period *domain.AccountingPeriod, actorID int64) error {
	if period == nil || period.ID <= 0 {
		return nil
	}
	id, err := tx.GetPostedJournalIDBySource(ctx, domain.JournalSourceClosing, period.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	reason := fmt.Sprintf("Pembatalan jurnal penutup periode %04d-%02d", period.PeriodYear, period.PeriodMonth)
	if _, err := tx.ReverseJournal(ctx, id, time.Now(), reason, actorID); err != nil {
		if errors.Is(err, domain.ErrJournalAlreadyRev) {
			return nil
		}
		return err
	}
	return nil
}

// closingMapping mengambil pemetaan akun auto-posting. Dipisah agar pesan error
// saat pengaturan kosong tetap seragam.
func (u *journalUsecase) closingMapping(ctx context.Context) (map[string]int64, error) {
	if u.settings == nil {
		return map[string]int64{}, nil
	}
	return u.settings.ListMapping(ctx)
}
