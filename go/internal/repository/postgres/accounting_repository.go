package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Modul akuntansi memakai empat struct repository terpisah: ChartOfAccount dan
// Journal sama-sama mendefinisikan GetByID/GetListCursor dengan tipe hasil
// berbeda, sehingga tidak boleh diimplementasikan oleh satu struct yang sama.
// Pemetaan akun auto-posting dipisah lagi karena membaca tabel yang sama sekali
// berbeda (s_accounting_settings).

// convertPgTimeZ mengubah pgtype.Timestamptz; null menjadi nil.
func convertPgTimeZ(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	val := v.Time
	return &val
}

// numericToFloat mengekstrak nilai NUMERIC; null menjadi 0.
func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}

// uniqueViolationConstraint mengekstrak nama constraint ketika error adalah
// pelanggaran unique. return boolean false bila error bukan unique violation.
func uniqueViolationConstraint(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return pgErr.ConstraintName, true
	}
	return "", false
}

// journalSourceUniqueIndex adalah nama unique index parsial yang menjamin hanya
// satu jurnal POSTED BELUM DIBALIK untuk tiap (source_type, source_id) — inilah
// pengaman idempotensi auto-posting (GRN) dan jurnal penutup. Jurnal yang sudah
// dibalik dikecualikan, sehingga sumber yang sama boleh menerbitkan jurnal baru
// setelah pembalikan (mis. periode ditutup ulang).
const journalSourceUniqueIndex = "idx_journal_source_unique"

// --- Chart of Accounts ---

// sqlChartOfAccountRepository mengelola bagan akun.
type sqlChartOfAccountRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewSQLChartOfAccountRepository menginisialisasi repository bagan akun.
func NewSQLChartOfAccountRepository(pool *pgxpool.Pool) domain.ChartOfAccountRepository {
	return &sqlChartOfAccountRepository{pool: pool, q: db.New(pool)}
}

func mapCOA(v db.GetChartOfAccountsCursorNextRow) domain.ChartOfAccount {
	return domain.ChartOfAccount{
		ID:            v.ID,
		Code:          v.Code,
		Name:          v.Name,
		AccountType:   v.AccountType,
		NormalBalance: v.NormalBalance,
		ParentID:      convertPgInt64(v.ParentID),
		ParentCode:    v.ParentCode.String,
		ParentName:    v.ParentName.String,
		IsHeader:      v.IsHeader,
		IsActive:      v.IsActive,
		CreatedAt:     v.CreatedAt.Time,
		UpdatedAt:     v.UpdatedAt.Time,
		DeletedAt:     convertPgTimeZ(v.DeletedAt),
	}
}

func (r *sqlChartOfAccountRepository) Create(ctx context.Context, acc *domain.ChartOfAccount) error {
	row, err := r.q.CreateChartOfAccount(ctx, db.CreateChartOfAccountParams{
		Code:          acc.Code,
		Name:          acc.Name,
		AccountType:   acc.AccountType,
		NormalBalance: acc.NormalBalance,
		ParentID:      nullableInt8(acc.ParentID),
		IsHeader:      acc.IsHeader,
		CreatedBy:     nullableInt8(acc.CreatedBy),
		UpdatedBy:     nullableInt8(acc.UpdatedBy),
	})
	if err != nil {
		if _, ok := uniqueViolationConstraint(err); ok {
			return domain.ErrAccountDuplicate
		}
		return err
	}
	acc.ID = row.ID
	acc.IsActive = row.IsActive
	acc.CreatedAt = row.CreatedAt.Time
	acc.UpdatedAt = row.UpdatedAt.Time
	return nil
}

func (r *sqlChartOfAccountRepository) GetListCursor(ctx context.Context, param domain.ChartOfAccountParam) ([]domain.ChartOfAccount, int64, error) {
	var rows []db.GetChartOfAccountsCursorNextRow
	var err error

	if param.Direction == "prev" {
		prevRows, err := r.q.GetChartOfAccountsCursorPrev(ctx, db.GetChartOfAccountsCursorPrevParams{
			SearchQuery: param.Search,
			CursorID:    param.CursorID,
			ReqLimit:    param.Limit,
		})
		if err != nil {
			return nil, 0, err
		}
		for _, v := range prevRows {
			rows = append(rows, db.GetChartOfAccountsCursorNextRow(v))
		}
		reverseRows(rows)
	} else {
		rows, err = r.q.GetChartOfAccountsCursorNext(ctx, db.GetChartOfAccountsCursorNextParams{
			SearchQuery: param.Search,
			CursorID:    param.CursorID,
			ReqLimit:    param.Limit,
		})
		if err != nil {
			return nil, 0, err
		}
	}

	count, err := r.q.CountChartOfAccounts(ctx, param.Search)
	if err != nil {
		return nil, 0, err
	}

	results := make([]domain.ChartOfAccount, 0, len(rows))
	for _, v := range rows {
		results = append(results, mapCOA(v))
	}
	return results, count, nil
}

func (r *sqlChartOfAccountRepository) GetByID(ctx context.Context, id int64) (*domain.ChartOfAccount, error) {
	row, err := r.q.GetChartOfAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	acc := mapCOA(db.GetChartOfAccountsCursorNextRow(row))
	return &acc, nil
}

func (r *sqlChartOfAccountRepository) GetByIDs(ctx context.Context, ids []int64) ([]domain.ChartOfAccount, error) {
	if len(ids) == 0 {
		return []domain.ChartOfAccount{}, nil
	}
	rows, err := r.q.GetChartOfAccountsForPosting(ctx, ids)
	if err != nil {
		return nil, err
	}
	results := make([]domain.ChartOfAccount, 0, len(rows))
	for _, v := range rows {
		results = append(results, domain.ChartOfAccount{
			ID:            v.ID,
			Code:          v.Code,
			ParentID:      convertPgInt64(v.ParentID),
			IsHeader:      v.IsHeader,
			IsActive:      v.IsActive,
			AccountType:   v.AccountType,
			NormalBalance: v.NormalBalance,
		})
	}
	return results, nil
}

func (r *sqlChartOfAccountRepository) Update(ctx context.Context, acc *domain.ChartOfAccount) error {
	err := r.q.UpdateChartOfAccount(ctx, db.UpdateChartOfAccountParams{
		Code:          acc.Code,
		Name:          acc.Name,
		AccountType:   acc.AccountType,
		NormalBalance: acc.NormalBalance,
		ParentID:      nullableInt8(acc.ParentID),
		IsHeader:      acc.IsHeader,
		IsActive:      acc.IsActive,
		UpdatedBy:     nullableInt8(acc.UpdatedBy),
		ID:            acc.ID,
	})
	if err != nil {
		if _, ok := uniqueViolationConstraint(err); ok {
			return domain.ErrAccountDuplicate
		}
		return err
	}
	return nil
}

func (r *sqlChartOfAccountRepository) Delete(ctx context.Context, id int64, deletedBy int64) error {
	return r.q.DeleteChartOfAccount(ctx, db.DeleteChartOfAccountParams{
		DeletedBy: pgtype.Int8{Int64: deletedBy, Valid: deletedBy > 0},
		ID:        id,
	})
}

// --- Accounting Period ---

// sqlAccountingPeriodRepository mengelola periode fiskal.
type sqlAccountingPeriodRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewSQLAccountingPeriodRepository menginisialisasi repository periode fiskal.
func NewSQLAccountingPeriodRepository(pool *pgxpool.Pool) domain.AccountingPeriodRepository {
	return &sqlAccountingPeriodRepository{pool: pool, q: db.New(pool)}
}

func mapPeriod(v db.GetAccountPeriodRow) domain.AccountingPeriod {
	return domain.AccountingPeriod{
		ID:          v.ID,
		PeriodYear:  int(v.PeriodYear),
		PeriodMonth: int(v.PeriodMonth),
		Status:      v.Status,
		ClosedAt:    convertPgTimeZ(v.ClosedAt),
		ClosedBy:    convertPgInt64(v.ClosedBy),
		CreatedAt:   v.CreatedAt.Time,
		UpdatedAt:   v.UpdatedAt.Time,
	}
}

func (r *sqlAccountingPeriodRepository) Get(ctx context.Context, year, month int) (*domain.AccountingPeriod, error) {
	row, err := r.q.GetAccountPeriod(ctx, db.GetAccountPeriodParams{
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
	})
	if err != nil {
		return nil, err
	}
	period := mapPeriod(row)
	return &period, nil
}

// EnsureOpen memastikan periode sasaran jurnal boleh dipakai:
//   - belum ada → periode baru dibuat (OPEN);
//   - sudah CLOSED → ditolak agar catatan periode tertutup tidak berubah.
//
// Membuat periode baru secara implisit tetap diizinkan karena UX lebih mulus
// daripada mewajibkan admin membuka periode manual sebelum jurnal pertama.
func (r *sqlAccountingPeriodRepository) EnsureOpen(ctx context.Context, year, month int, actorID int64) (*domain.AccountingPeriod, error) {
	row, err := r.q.GetAccountPeriod(ctx, db.GetAccountPeriodParams{
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		if row.Status != domain.PeriodStatusOpen {
			return nil, domain.ErrJournalPeriodClosed
		}
		period := mapPeriod(row)
		return &period, nil
	}

	created, err := r.q.CreateAccountPeriod(ctx, db.CreateAccountPeriodParams{
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
		CreatedBy:   pgtype.Int8{Int64: actorID, Valid: actorID > 0},
		UpdatedBy:   pgtype.Int8{Int64: actorID, Valid: actorID > 0},
	})
	if err != nil {
		return nil, err
	}
	if created.Status != domain.PeriodStatusOpen {
		return nil, domain.ErrJournalPeriodClosed
	}
	period := mapPeriod(db.GetAccountPeriodRow(created))
	return &period, nil
}

// Close menutup periode dengan compare-and-swap: hanya status OPEN yang berubah.
// Mengembalikan jumlah baris terpengaruh (0 = sudah CLOSED / tidak ada).
func (r *sqlAccountingPeriodRepository) Close(ctx context.Context, year, month, actorID int64) (int64, error) {
	return r.q.CloseAccountPeriod(ctx, db.CloseAccountPeriodParams{
		ActorUserID: actorID,
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
	})
}

// Reopen membuka kembali periode CLOSED. Mengembalikan jumlah baris terpengaruh.
func (r *sqlAccountingPeriodRepository) Reopen(ctx context.Context, year, month int) (int64, error) {
	return r.q.ReopenAccountPeriod(ctx, db.ReopenAccountPeriodParams{
		PeriodYear:  int32(year),
		PeriodMonth: int32(month),
	})
}

func (r *sqlAccountingPeriodRepository) List(ctx context.Context, limit int) ([]domain.AccountingPeriod, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := r.q.ListAccountPeriods(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	results := make([]domain.AccountingPeriod, 0, len(rows))
	for _, v := range rows {
		results = append(results, mapPeriod(db.GetAccountPeriodRow(v)))
	}
	return results, nil
}

// --- Journal ---

// sqlJournalRepository mengelola jurnal umum.
type sqlJournalRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewSQLJournalRepository menginisialisasi repository jurnal umum.
func NewSQLJournalRepository(pool *pgxpool.Pool) domain.JournalRepository {
	return &sqlJournalRepository{pool: pool, q: db.New(pool)}
}

func mapJournalRow(v db.GetJournalEntriesCursorNextRow) domain.JournalEntry {
	return domain.JournalEntry{
		ID:              v.ID,
		EntryNumber:     v.EntryNumber,
		EntryDate:       v.EntryDate.Time,
		Description:     v.Description.String,
		SourceType:      v.SourceType.String,
		SourceID:        convertPgInt64(v.SourceID),
		SourceReference: v.SourceReference.String,
		Status:          v.Status,
		TotalDebit:      numericToFloat(v.TotalDebit),
		TotalCredit:     numericToFloat(v.TotalCredit),
		PeriodYear:      int(v.PeriodYear),
		PeriodMonth:     int(v.PeriodMonth),
		PostedAt:        convertPgTime(v.PostedAt),
		VoidedAt:        convertPgTime(v.VoidedAt),
		ReversalOfID:    convertPgInt64(v.ReversalOfID),
		ReversedByID:    convertPgInt64(v.ReversedByID),
		CreatedAt:       v.CreatedAt.Time,
		UpdatedAt:       v.UpdatedAt.Time,
		CreatedBy:       convertPgInt64(v.CreatedBy),
	}
}

func mapJournalLine(v db.GetJournalEntryLinesRow) domain.JournalLine {
	return domain.JournalLine{
		ID:          v.ID,
		LineNo:      int(v.LineNo),
		AccountID:   v.AccountID,
		AccountCode: v.AccountCode,
		AccountName: v.AccountName,
		Description: v.Description.String,
		Debit:       numericToFloat(v.Debit),
		Credit:      numericToFloat(v.Credit),
	}
}

func (r *sqlJournalRepository) CreateDraft(ctx context.Context, entry *domain.JournalEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	actor := nullableInt8(entry.CreatedBy)

	header, err := qtx.CreateJournalEntryHeader(ctx, db.CreateJournalEntryHeaderParams{
		EntryNumber:     entry.EntryNumber,
		EntryDate:       pgtype.Date{Time: entry.EntryDate, Valid: true},
		Description:     pgtype.Text{String: entry.Description, Valid: entry.Description != ""},
		SourceType:      pgtype.Text{String: entry.SourceType, Valid: entry.SourceType != ""},
		SourceID:        nullableInt8(entry.SourceID),
		SourceReference: pgtype.Text{String: entry.SourceReference, Valid: entry.SourceReference != ""},
		Status:          entry.Status,
		TotalDebit:      numericFromFloat(entry.TotalDebit),
		TotalCredit:     numericFromFloat(entry.TotalCredit),
		PeriodYear:      int32(entry.PeriodYear),
		PeriodMonth:     int32(entry.PeriodMonth),
		ReversalOfID:    nullableInt8(entry.ReversalOfID),
		CreatedBy:       actor,
		UpdatedBy:       actor,
	})
	if err != nil {
		if _, ok := uniqueViolationConstraint(err); ok {
			return domain.ErrNomorJurnalBentrok
		}
		return err
	}

	entry.ID = header.ID
	entry.Status = header.Status
	entry.CreatedAt = header.CreatedAt.Time
	entry.UpdatedAt = header.UpdatedAt.Time

	for _, line := range entry.Lines {
		if err := qtx.CreateJournalEntryLine(ctx, db.CreateJournalEntryLineParams{
			JournalEntryID: header.ID,
			LineNo:         int32(line.LineNo),
			AccountID:      line.AccountID,
			Description:    pgtype.Text{String: line.Description, Valid: line.Description != ""},
			Debit:          numericFromFloat(line.Debit),
			Credit:         numericFromFloat(line.Credit),
		}); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *sqlJournalRepository) GetListCursor(ctx context.Context, param domain.JournalEntryParam) ([]domain.JournalEntry, int64, error) {
	var rows []db.GetJournalEntriesCursorNextRow
	var err error

	if param.Direction == "prev" {
		prevRows, err := r.q.GetJournalEntriesCursorPrev(ctx, db.GetJournalEntriesCursorPrevParams{
			SearchQuery:  param.Search,
			StatusFilter: param.StatusFilter,
			CursorID:     param.CursorID,
			ReqLimit:     param.Limit,
		})
		if err != nil {
			return nil, 0, err
		}
		for _, v := range prevRows {
			rows = append(rows, db.GetJournalEntriesCursorNextRow(v))
		}
		reverseRows(rows)
	} else {
		rows, err = r.q.GetJournalEntriesCursorNext(ctx, db.GetJournalEntriesCursorNextParams{
			SearchQuery:  param.Search,
			StatusFilter: param.StatusFilter,
			CursorID:     param.CursorID,
			ReqLimit:     param.Limit,
		})
		if err != nil {
			return nil, 0, err
		}
	}

	count, err := r.q.CountJournalEntries(ctx, db.CountJournalEntriesParams{
		SearchQuery:  param.Search,
		StatusFilter: param.StatusFilter,
	})
	if err != nil {
		return nil, 0, err
	}

	results := make([]domain.JournalEntry, 0, len(rows))
	for _, v := range rows {
		results = append(results, mapJournalRow(v))
	}
	return results, count, nil
}

func (r *sqlJournalRepository) GetByID(ctx context.Context, id int64) (*domain.JournalEntry, error) {
	row, err := r.q.GetJournalEntryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	entry := mapJournalRow(db.GetJournalEntriesCursorNextRow(row))

	lines, err := r.q.GetJournalEntryLines(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, l := range lines {
		entry.Lines = append(entry.Lines, mapJournalLine(l))
	}
	return &entry, nil
}

// Post memindahkan DRAFT → POSTED lewat perbandingan CAS di SQL. Nol baris
// terpengaruh berarti status bukan DRAFT lagi, periode tertutup, atau ada baris
// ke akun non-posting — usecase memastikan pesannya lebih dulu sebelum CAS ini.
func (r *sqlJournalRepository) Post(ctx context.Context, id, actorID int64) (int64, error) {
	return r.q.PostJournalEntry(ctx, db.PostJournalEntryParams{ActorUserID: actorID, ID: id})
}

func (r *sqlJournalRepository) Void(ctx context.Context, id, actorID int64, reason string) (int64, error) {
	return r.q.VoidJournalEntry(ctx, db.VoidJournalEntryParams{
		ActorUserID: actorID,
		VoidReason:  pgtype.Text{String: reason, Valid: reason != ""},
		ID:          id,
	})
}

// Reverse membuat jurnal pembalik secara atomik dalam satu transaksi:
//  1. memuat jurnal asli dan meyakinkan statusnya POSTED;
//  2. memastikan akun barisnya masih layak posting & periode masih OPEN;
//  3. membuat jurnal pembalik (debit↔kredit) lalu mempostingnya → CAS;
//  4. menandai jurnal asli sebagai sudah dibalik → CAS (jaga dua pembalikan
//     paralel agar hanya satu yang menang).
//
// Bila langkah mana pun gagal, seluruh transaksi digulung termasuk penghitung
// nomor jurnal yang tadi dimajukan.
func (r *sqlJournalRepository) Reverse(ctx context.Context, originalID int64, entryDate time.Time, reason string, actorID int64) (*domain.JournalEntry, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	origRow, err := qtx.GetJournalEntryByID(ctx, originalID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrJournalNotFound
		}
		return nil, err
	}
	if origRow.Status != domain.JournalStatusPosted {
		return nil, domain.ErrJournalNotPosted
	}
	if origRow.ReversedByID.Valid {
		return nil, domain.ErrJournalAlreadyRev
	}

	periodRow, err := qtx.GetAccountPeriod(ctx, db.GetAccountPeriodParams{
		PeriodYear:  origRow.PeriodYear,
		PeriodMonth: origRow.PeriodMonth,
	})
	if err != nil {
		return nil, domain.ErrJournalPeriodClosed
	}
	if periodRow.Status != domain.PeriodStatusOpen {
		return nil, domain.ErrJournalPeriodClosed
	}

	origLines, err := qtx.GetJournalEntryLines(ctx, originalID)
	if err != nil {
		return nil, err
	}

	accountIDs := make([]int64, 0, len(origLines))
	for _, l := range origLines {
		accountIDs = append(accountIDs, l.AccountID)
	}
	accounts, err := qtx.GetChartOfAccountsForPosting(ctx, accountIDs)
	if err != nil {
		return nil, err
	}
	postable := make(map[int64]bool, len(accounts))
	for _, a := range accounts {
		postable[a.ID] = !a.IsHeader && a.IsActive
	}
	for _, l := range origLines {
		if !postable[l.AccountID] {
			return nil, &domain.JournalNotPostableError{AccountID: l.AccountID}
		}
	}

	datePrefix := origRow.EntryDate.Time.Format("20060102")
	seq, err := qtx.NextJournalSeq(ctx, datePrefix)
	if err != nil {
		return nil, err
	}
	reversalNumber := fmt.Sprintf("JV/%s/%04d", datePrefix, seq)

	description := reason
	if description == "" {
		description = fmt.Sprintf("Pembalikan %s", origRow.EntryNumber)
	}

	rev := &domain.JournalEntry{
		EntryNumber: reversalNumber,
		EntryDate:   origRow.EntryDate.Time,
		Description: description,
		SourceType:  domain.JournalSourceManual,
		Status:      domain.JournalStatusDraft,
		TotalDebit:  numericToFloat(origRow.TotalCredit),
		TotalCredit: numericToFloat(origRow.TotalDebit),
		PeriodYear:  int(origRow.PeriodYear),
		PeriodMonth: int(origRow.PeriodMonth),
	}

	header, err := qtx.CreateJournalEntryHeader(ctx, db.CreateJournalEntryHeaderParams{
		EntryNumber:  rev.EntryNumber,
		EntryDate:    pgtype.Date{Time: origRow.EntryDate.Time, Valid: true},
		Description:  pgtype.Text{String: description, Valid: description != ""},
		SourceType:   pgtype.Text{String: domain.JournalSourceManual, Valid: true},
		Status:       domain.JournalStatusDraft,
		TotalDebit:   numericFromFloat(rev.TotalDebit),
		TotalCredit:  numericFromFloat(rev.TotalCredit),
		PeriodYear:   origRow.PeriodYear,
		PeriodMonth:  origRow.PeriodMonth,
		ReversalOfID: pgtype.Int8{Int64: originalID, Valid: true},
		CreatedBy:    pgtype.Int8{Int64: actorID, Valid: true},
		UpdatedBy:    pgtype.Int8{Int64: actorID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	rev.ID = header.ID

	for _, l := range origLines {
		desc := l.Description.String
		if desc != "" {
			desc = "Pembalikan: " + desc
		}
		if err := qtx.CreateJournalEntryLine(ctx, db.CreateJournalEntryLineParams{
			JournalEntryID: header.ID,
			LineNo:         l.LineNo,
			AccountID:      l.AccountID,
			Description:    pgtype.Text{String: desc, Valid: desc != ""},
			Debit:          l.Credit,
			Credit:         l.Debit,
		}); err != nil {
			return nil, err
		}
	}

	rows, err := qtx.PostJournalEntry(ctx, db.PostJournalEntryParams{ActorUserID: actorID, ID: header.ID})
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, domain.ErrJournalPostBox
	}

	marked, err := qtx.SetOriginalJournalReversed(ctx, db.SetOriginalJournalReversedParams{
		ReversalID: header.ID,
		ID:         originalID,
	})
	if err != nil {
		return nil, err
	}
	if marked == 0 {
		return nil, domain.ErrJournalAlreadyRev
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	rev.Status = domain.JournalStatusPosted
	now := time.Now()
	rev.PostedBy = &actorID
	rev.PostedAt = &now
	rev.CreatedAt = now
	rev.UpdatedAt = now
	rev.Lines = make([]domain.JournalLine, 0, len(origLines))
	for _, l := range origLines {
		rev.Lines = append(rev.Lines, domain.JournalLine{
			LineNo:      int(l.LineNo),
			AccountID:   l.AccountID,
			Description: l.Description.String,
			Debit:       numericToFloat(l.Credit),
			Credit:      numericToFloat(l.Debit),
		})
	}
	return rev, nil
}

// NextSeq memajukan penghitung nomor jurnal untuk prefiks tanggal tertentu.
func (r *sqlJournalRepository) NextSeq(ctx context.Context, prefix string) (int64, error) {
	return r.q.NextJournalSeq(ctx, prefix)
}

// GetPostedIDBySource mencari id jurnal POSTED yang belum dibalik pada pasangan
// (source_type, source_id). Mengembalikan pgx.ErrNoRows bila tidak ada. Dipakai
// untuk menemukan jurnal penutup sebuah periode saat periode dibuka kembali.
func (r *sqlJournalRepository) GetPostedIDBySource(ctx context.Context, sourceType string, sourceID int64) (int64, error) {
	return r.q.GetPostedJournalIDBySource(ctx, db.GetPostedJournalIDBySourceParams{
		SourceType: sourceType,
		SourceID:   sourceID,
	})
}

// CreatePosted menulis jurnal beserta seluruh barisnya dalam SATU transaksi dan
// statusnya langsung POSTED. Dua jalur pelanggaran unique dibedakan karena
// maknanya berbeda: nomor jurnal bentrok berarti penghitung perlu maju, sedangkan
// idx_journal_source_unique bentrok berarti jurnal untuk sumber ini sudah pernah
// terbit sehingga perulangan berikutnya cukup dilewati (idempoten).
func (r *sqlJournalRepository) CreatePosted(ctx context.Context, entry *domain.JournalEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	// id pelaku 0 berarti "tidak ada pengguna" (mis. auto-posting latar belakang);
	// disimpan sebagai NULL agar tidak melanggar foreign key ke s_users.
	actor := pgtype.Int8{}
	if entry.CreatedBy != nil && *entry.CreatedBy > 0 {
		actor = pgtype.Int8{Int64: *entry.CreatedBy, Valid: true}
	}

	header, err := qtx.CreatePostedJournalEntryHeader(ctx, db.CreatePostedJournalEntryHeaderParams{
		EntryNumber:     entry.EntryNumber,
		EntryDate:       pgtype.Date{Time: entry.EntryDate, Valid: true},
		Description:     pgtype.Text{String: entry.Description, Valid: entry.Description != ""},
		SourceType:      pgtype.Text{String: entry.SourceType, Valid: entry.SourceType != ""},
		SourceID:        nullableInt8(entry.SourceID),
		SourceReference: pgtype.Text{String: entry.SourceReference, Valid: entry.SourceReference != ""},
		TotalDebit:      numericFromFloat(entry.TotalDebit),
		TotalCredit:     numericFromFloat(entry.TotalCredit),
		PeriodYear:      int32(entry.PeriodYear),
		PeriodMonth:     int32(entry.PeriodMonth),
		ReversalOfID:    nullableInt8(entry.ReversalOfID),
		PostedBy:        actor,
		CreatedBy:       actor,
		UpdatedBy:       actor,
	})
	if err != nil {
		if name, ok := uniqueViolationConstraint(err); ok {
			if name == journalSourceUniqueIndex {
				return domain.ErrJournalAlreadyPost
			}
			return domain.ErrNomorJurnalBentrok
		}
		return err
	}

	entry.ID = header.ID
	entry.Status = header.Status
	entry.CreatedAt = header.CreatedAt.Time
	entry.UpdatedAt = header.UpdatedAt.Time

	for _, line := range entry.Lines {
		if err := qtx.CreateJournalEntryLine(ctx, db.CreateJournalEntryLineParams{
			JournalEntryID: header.ID,
			LineNo:         int32(line.LineNo),
			AccountID:      line.AccountID,
			Description:    pgtype.Text{String: line.Description, Valid: line.Description != ""},
			Debit:          numericFromFloat(line.Debit),
			Credit:         numericFromFloat(line.Credit),
		}); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// --- Accounting Settings ---

// sqlAccountingSettingsRepository mengelola pemetaan kunci transaksi ke akun CoA
// yang dipakai auto-posting.
type sqlAccountingSettingsRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewSQLAccountingSettingsRepository menginisialisasi repository pengaturan akun.
func NewSQLAccountingSettingsRepository(pool *pgxpool.Pool) domain.AccountingSettingsRepository {
	return &sqlAccountingSettingsRepository{pool: pool, q: db.New(pool)}
}

// ListMapping mengembalikan pemetaan mapping_key → account_id.
func (r *sqlAccountingSettingsRepository) ListMapping(ctx context.Context) (map[string]int64, error) {
	rows, err := r.q.ListAccountingSettings(ctx)
	if err != nil {
		return nil, err
	}
	mapping := make(map[string]int64, len(rows))
	for _, row := range rows {
		mapping[row.MappingKey] = row.AccountID
	}
	return mapping, nil
}
