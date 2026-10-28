package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
		TotalDebit:      moneyFromNumeric(v.TotalDebit),
		TotalCredit:     moneyFromNumeric(v.TotalCredit),
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
		Debit:       moneyFromNumeric(v.Debit),
		Credit:      moneyFromNumeric(v.Credit),
	}
}

func (r *sqlJournalRepository) CreateDraft(ctx context.Context, entry *domain.JournalEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

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
		TotalDebit:      numericFromMoney(entry.TotalDebit),
		TotalCredit:     numericFromMoney(entry.TotalCredit),
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
			Debit:          numericFromMoney(line.Debit),
			Credit:         numericFromMoney(line.Credit),
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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)
	// Kunci periode lebih dulu. Ini menutup celah antara pengecekan status oleh
	// UPDATE dan transaksi tutup buku yang berjalan bersamaan.
	if _, err := qtx.LockJournalEntryPeriodForShare(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, domain.ErrJournalPeriodClosed
		}
		return 0, err
	}

	rows, err := qtx.PostJournalEntry(ctx, db.PostJournalEntryParams{ActorUserID: actorID, ID: id})
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return rows, nil
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
//     bersamaan agar hanya satu yang menang).
//
// Bila langkah mana pun gagal, seluruh transaksi digulung termasuk penghitung
// nomor jurnal yang tadi dimajukan.
func (r *sqlJournalRepository) Reverse(ctx context.Context, originalID int64, entryDate time.Time, reason string, actorID int64) (*domain.JournalEntry, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	entry, err := reverseJournal(ctx, r.q.WithTx(tx), originalID, entryDate, reason, actorID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return entry, nil
}

// reverseJournal menjalankan pembalikan jurnal dengan kueri milik transaksi
// pemanggil. Periode dikunci FOR SHARE agar penutupan buku tidak bisa berjalan
// bersamaan dengan pembalik yang masih membaca status OPEN.
func reverseJournal(ctx context.Context, q *db.Queries, originalID int64, entryDate time.Time, reason string, actorID int64) (*domain.JournalEntry, error) {
	origRow, err := q.GetJournalEntryByID(ctx, originalID)
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

	periodRow, err := q.GetAccountPeriodForShare(ctx, db.GetAccountPeriodForShareParams{
		PeriodYear:  origRow.PeriodYear,
		PeriodMonth: origRow.PeriodMonth,
	})
	if err != nil {
		return nil, domain.ErrJournalPeriodClosed
	}
	if periodRow.Status != domain.PeriodStatusOpen {
		return nil, domain.ErrJournalPeriodClosed
	}

	origLines, err := q.GetJournalEntryLines(ctx, originalID)
	if err != nil {
		return nil, err
	}

	accountIDs := make([]int64, 0, len(origLines))
	for _, l := range origLines {
		accountIDs = append(accountIDs, l.AccountID)
	}
	accounts, err := q.GetChartOfAccountsForPosting(ctx, accountIDs)
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
	seq, err := q.NextJournalSeq(ctx, datePrefix)
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
		TotalDebit:  moneyFromNumeric(origRow.TotalCredit),
		TotalCredit: moneyFromNumeric(origRow.TotalDebit),
		PeriodYear:  int(origRow.PeriodYear),
		PeriodMonth: int(origRow.PeriodMonth),
	}

	header, err := q.CreateJournalEntryHeader(ctx, db.CreateJournalEntryHeaderParams{
		EntryNumber:  rev.EntryNumber,
		EntryDate:    pgtype.Date{Time: origRow.EntryDate.Time, Valid: true},
		Description:  pgtype.Text{String: description, Valid: description != ""},
		SourceType:   pgtype.Text{String: domain.JournalSourceManual, Valid: true},
		Status:       domain.JournalStatusDraft,
		TotalDebit:   numericFromMoney(rev.TotalDebit),
		TotalCredit:  numericFromMoney(rev.TotalCredit),
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
		if err := q.CreateJournalEntryLine(ctx, db.CreateJournalEntryLineParams{
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

	rows, err := q.PostJournalEntry(ctx, db.PostJournalEntryParams{ActorUserID: actorID, ID: header.ID})
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, domain.ErrJournalPostBox
	}

	marked, err := q.SetOriginalJournalReversed(ctx, db.SetOriginalJournalReversedParams{
		ReversalID: header.ID,
		ID:         originalID,
	})
	if err != nil {
		return nil, err
	}
	if marked == 0 {
		return nil, domain.ErrJournalAlreadyRev
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
			Debit:       moneyFromNumeric(l.Credit),
			Credit:      moneyFromNumeric(l.Debit),
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
	defer func() { _ = tx.Rollback(ctx) }()

	if err := createPosted(ctx, r.q.WithTx(tx), entry); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// createPosted menulis jurnal POSTED memakai kueri milik transaksi pemanggil.
// Kueri header hanya menghasilkan baris bila periode masih OPEN; pgx.ErrNoRows
// berarti periode sudah ditutup atau belum dibuat.
func createPosted(ctx context.Context, q *db.Queries, entry *domain.JournalEntry) error {
	// id pelaku 0 berarti "tidak ada pengguna" (mis. auto-posting latar belakang);
	// disimpan sebagai NULL agar tidak melanggar foreign key ke s_users.
	actor := pgtype.Int8{}
	if entry.CreatedBy != nil && *entry.CreatedBy > 0 {
		actor = pgtype.Int8{Int64: *entry.CreatedBy, Valid: true}
	}

	header, err := q.CreatePostedJournalEntryHeader(ctx, db.CreatePostedJournalEntryHeaderParams{
		EntryNumber:     entry.EntryNumber,
		EntryDate:       pgtype.Date{Time: entry.EntryDate, Valid: true},
		Description:     pgtype.Text{String: entry.Description, Valid: entry.Description != ""},
		SourceType:      pgtype.Text{String: entry.SourceType, Valid: entry.SourceType != ""},
		SourceID:        nullableInt8(entry.SourceID),
		SourceReference: pgtype.Text{String: entry.SourceReference, Valid: entry.SourceReference != ""},
		TotalDebit:      numericFromMoney(entry.TotalDebit),
		TotalCredit:     numericFromMoney(entry.TotalCredit),
		PeriodYear:      int32(entry.PeriodYear),
		PeriodMonth:     int32(entry.PeriodMonth),
		ReversalOfID:    nullableInt8(entry.ReversalOfID),
		PostedBy:        actor,
		CreatedBy:       actor,
		UpdatedBy:       actor,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrJournalPeriodClosed
		}
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
		if err := q.CreateJournalEntryLine(ctx, db.CreateJournalEntryLineParams{
			JournalEntryID: header.ID,
			LineNo:         int32(line.LineNo),
			AccountID:      line.AccountID,
			Description:    pgtype.Text{String: line.Description, Valid: line.Description != ""},
			Debit:          numericFromMoney(line.Debit),
			Credit:         numericFromMoney(line.Credit),
		}); err != nil {
			return err
		}
	}
	return nil
}

// --- Accounting Settings ---

// sqlAccountingSettingsRepository mengelola pemetaan kunci transaksi ke akun CoA
// yang dipakai auto-posting.
