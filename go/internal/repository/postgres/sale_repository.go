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

type sqlSaleRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewSQLSaleRepository(pool *pgxpool.Pool) domain.SaleRepository {
	return &sqlSaleRepository{pool: pool, q: db.New(pool)}
}

// uniqueProductIDs mengumpulkan id produk unik dari baris penjualan.
func uniqueProductIDs(details []domain.SaleDetail) []int64 {
	seen := make(map[int64]struct{}, len(details))
	ids := make([]int64, 0, len(details))
	for _, d := range details {
		if _, ok := seen[d.ProductID]; ok {
			continue
		}
		seen[d.ProductID] = struct{}{}
		ids = append(ids, d.ProductID)
	}
	return ids
}

// insertPostedJournal menulis jurnal POSTED beserta barisnya memakai penghitung
// nomor jurnal di dalam transaksi yang sedang berjalan. Mengembalikan id jurnal.
func insertPostedJournal(ctx context.Context, qtx *db.Queries, plan *domain.SaleJournalPlan, sourceType string, sourceID *int64, sourceReference string, actorID int64) (int64, error) {
	datePrefix := plan.EntryDate.Format("20060102")
	seq, err := qtx.NextJournalSeq(ctx, datePrefix)
	if err != nil {
		return 0, err
	}
	entryNumber := fmt.Sprintf("JV/%s/%04d", datePrefix, seq)

	actor := int8FromID(actorID)
	ref := sourceReference
	if ref == "" {
		ref = plan.SourceReference
	}

	var srcType, srcRef, desc pgtype.Text
	if sourceType != "" {
		srcType = pgtype.Text{String: sourceType, Valid: true}
	}
	if ref != "" {
		srcRef = pgtype.Text{String: ref, Valid: true}
	}
	if plan.Description != "" {
		desc = pgtype.Text{String: plan.Description, Valid: true}
	}

	header, err := qtx.CreatePostedJournalEntryHeader(ctx, db.CreatePostedJournalEntryHeaderParams{
		EntryNumber:     entryNumber,
		EntryDate:       pgtype.Date{Time: plan.EntryDate, Valid: true},
		Description:     desc,
		SourceType:      srcType,
		SourceID:        nullableInt8(sourceID),
		SourceReference: srcRef,
		TotalDebit:      numericFromFloat(plan.TotalDebit),
		TotalCredit:     numericFromFloat(plan.TotalCredit),
		PeriodYear:      int32(plan.EntryDate.Year()),
		PeriodMonth:     int32(plan.EntryDate.Month()),
		ReversalOfID:    nullableInt8(plan.ReversalOfID),
		PostedBy:        actor,
		CreatedBy:       actor,
		UpdatedBy:       actor,
	})
	if err != nil {
		if name, ok := uniqueViolationConstraint(err); ok {
			if name == journalSourceUniqueIndex {
				return 0, domain.ErrJournalAlreadyPost
			}
			return 0, domain.ErrNomorJurnalBentrok
		}
		return 0, err
	}

	for i, l := range plan.Lines {
		var lineDesc pgtype.Text
		if l.Description != "" {
			lineDesc = pgtype.Text{String: l.Description, Valid: true}
		}
		if err := qtx.CreateJournalEntryLine(ctx, db.CreateJournalEntryLineParams{
			JournalEntryID: header.ID,
			LineNo:         int32(i + 1),
			AccountID:      l.AccountID,
			Description:    lineDesc,
			Debit:          numericFromFloat(l.Debit),
			Credit:         numericFromFloat(l.Credit),
		}); err != nil {
			return 0, err
		}
	}
	return header.ID, nil
}

// applyMemberPointsForSale mencatat penukaran (redeem) lalu perolehan (earn)
// poin pada buku besar dan memperbarui saldo member, semua di dalam transaksi
// nota. Tukar poin mengurangi saldo lebih dulu agar constraint saldo >= 0 aman.
func applyMemberPointsForSale(ctx context.Context, qtx *db.Queries, memberID, saleID, redeemPoints, pointsEarned, actorID int64) error {
	locked, err := qtx.LockMemberPoints(ctx, memberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("member tidak ditemukan")
		}
		return err
	}
	balance := locked.Points
	saleIDPtr := &saleID

	if redeemPoints > 0 {
		if redeemPoints > balance {
			return domain.ErrSaleRedeemTanpaMember
		}
		balance -= redeemPoints
		if err := qtx.InsertMemberPointEntry(ctx, db.InsertMemberPointEntryParams{
			MemberID:     memberID,
			SaleID:       nullableInt8(saleIDPtr),
			EntryType:    domain.MemberPointRedeem,
			Points:       -redeemPoints,
			BalanceAfter: balance,
			Note:         "Tukar poin pada penjualan",
			CreatedBy:    int8FromID(actorID),
		}); err != nil {
			return err
		}
	}
	if pointsEarned > 0 {
		balance += pointsEarned
		if err := qtx.InsertMemberPointEntry(ctx, db.InsertMemberPointEntryParams{
			MemberID:     memberID,
			SaleID:       nullableInt8(saleIDPtr),
			EntryType:    domain.MemberPointEarn,
			Points:       pointsEarned,
			BalanceAfter: balance,
			Note:         "Perolehan poin dari penjualan",
			CreatedBy:    int8FromID(actorID),
		}); err != nil {
			return err
		}
	}
	if balance == locked.Points {
		return nil
	}
	return qtx.SetMemberPoints(ctx, db.SetMemberPointsParams{
		Points:    balance,
		UpdatedBy: int8FromID(actorID),
		ID:        memberID,
	})
}

// reverseMemberPointsForSale membalik poin sebuah nota yang dibatalkan. Poin
// perolehan ditarik kembali (dibatasi agar saldo tidak negatif) dan poin yang
// ditukar dikembalikan. Urutannya dibalik dari saat dibuat.
func reverseMemberPointsForSale(ctx context.Context, qtx *db.Queries, memberID, saleID, pointsEarned, redeemPoints, actorID int64) error {
	locked, err := qtx.LockMemberPoints(ctx, memberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	balance := locked.Points
	saleIDPtr := &saleID

	if redeemPoints > 0 {
		balance += redeemPoints
		if err := qtx.InsertMemberPointEntry(ctx, db.InsertMemberPointEntryParams{
			MemberID:     memberID,
			SaleID:       nullableInt8(saleIDPtr),
			EntryType:    domain.MemberPointRedeemReversal,
			Points:       redeemPoints,
			BalanceAfter: balance,
			Note:         "Pembatalan penjualan: poin tukar dikembalikan",
			CreatedBy:    int8FromID(actorID),
		}); err != nil {
			return err
		}
	}
	if pointsEarned > 0 {
		rev := pointsEarned
		if rev > balance {
			rev = balance
		}
		balance -= rev
		if err := qtx.InsertMemberPointEntry(ctx, db.InsertMemberPointEntryParams{
			MemberID:     memberID,
			SaleID:       nullableInt8(saleIDPtr),
			EntryType:    domain.MemberPointEarnReversal,
			Points:       -rev,
			BalanceAfter: balance,
			Note:         "Pembatalan penjualan: poin perolehan ditarik",
			CreatedBy:    int8FromID(actorID),
		}); err != nil {
			return err
		}
	}
	if balance == locked.Points {
		return nil
	}
	return qtx.SetMemberPoints(ctx, db.SetMemberPointsParams{
		Points:    balance,
		UpdatedBy: int8FromID(actorID),
		ID:        memberID,
	})
}

func (r *sqlSaleRepository) GetProductsForSale(ctx context.Context, ids []int64) (map[int64]domain.SaleProductInfo, error) {
	result := make(map[int64]domain.SaleProductInfo)
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := r.q.GetProductsForSale(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, v := range rows {
		result[v.ID] = domain.SaleProductInfo{
			ID:            v.ID,
			SKU:           v.Sku,
			Name:          v.Name,
			SellingPrice:  numericToFloat(v.SellingPrice),
			AvgCost:       numericToFloat(v.AvgCost),
			CurrentStock:  int(v.CurrentStock),
			DiscountType:  v.DiscountType,
			DiscountValue: numericToFloat(v.DiscountValue),
			IsTaxable:     v.IsTaxable,
		}
	}
	return result, nil
}

func (r *sqlSaleRepository) CreateSale(ctx context.Context, sale *domain.Sale, journal *domain.SaleJournalPlan) (*domain.Sale, error) {
	if len(sale.Details) == 0 {
		return nil, domain.ErrSaleItemsRequired
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	// Kunci seluruh produk yang dijual supaya validasi stok final tidak bisa
	// dilewati oleh penjualan paralel produk yang sama.
	ids := uniqueProductIDs(sale.Details)
	locks, err := qtx.LockProductsForSale(ctx, ids)
	if err != nil {
		return nil, err
	}
	locked := make(map[int64]struct{}, len(locks))
	for _, l := range locks {
		locked[l.ID] = struct{}{}
	}
	for _, d := range sale.Details {
		if _, ok := locked[d.ProductID]; !ok {
			return nil, fmt.Errorf("%w: id %d", domain.ErrSaleProductNotFound, d.ProductID)
		}
	}

	now := time.Now()
	if sale.SaleDate.IsZero() {
		sale.SaleDate = now
	}
	datePrefix := sale.SaleDate.Format("20060102")
	seq, err := qtx.NextSaleSeq(ctx, datePrefix)
	if err != nil {
		return nil, err
	}
	saleNumber := fmt.Sprintf("POS-%s-%04d", datePrefix, seq)

	var actorID int64
	if sale.CreatedBy != nil {
		actorID = *sale.CreatedBy
	}
	actor := int8FromID(actorID)

	var dueDate pgtype.Date
	if sale.DueDate != nil && !sale.DueDate.IsZero() {
		dueDate = pgtype.Date{Time: *sale.DueDate, Valid: true}
	}

	var memberID pgtype.Int8
	if sale.MemberID > 0 {
		memberID = pgtype.Int8{Int64: sale.MemberID, Valid: true}
	}

	header, err := qtx.CreateSale(ctx, db.CreateSaleParams{
		SaleNumber:     saleNumber,
		SaleDate:       pgtype.Date{Time: sale.SaleDate, Valid: true},
		MemberID:       memberID,
		TaxType:        sale.TaxType,
		Subtotal:       numericFromFloat(sale.Subtotal),
		ItemDiscount:   numericFromFloat(sale.ItemDiscount),
		DiscountType:   sale.DiscountType,
		DiscountValue:  numericFromFloat(sale.DiscountValue),
		DiscountAmount: numericFromFloat(sale.DiscountAmount),
		DppAmount:      numericFromFloat(sale.DppAmount),
		TaxAmount:      numericFromFloat(sale.TaxAmount),
		TotalAmount:    numericFromFloat(sale.TotalAmount),
		PaidAmount:     numericFromFloat(sale.PaidAmount),
		ChangeAmount:   numericFromFloat(sale.ChangeAmount),
		CostTotal:      numericFromFloat(sale.CostTotal),
		PointsEarned:   sale.PointsEarned,
		RedeemPoints:   sale.RedeemPoints,
		RedeemAmount:   numericFromFloat(sale.RedeemAmount),
		PaymentStatus:  sale.PaymentStatus,
		Status:         domain.SaleStatusPosted,
		DueDate:        dueDate,
		Notes:          sale.Notes,
		CreatedBy:      actor,
		UpdatedBy:      actor,
	})
	if err != nil {
		return nil, err
	}

	// Poin loyalitas: tukar dulu (mengurangi saldo), lalu perolehan (menambah).
	// Balance dihitung ulang di bawah kunci baris member.
	if sale.MemberID > 0 && (sale.RedeemPoints > 0 || sale.PointsEarned > 0) {
		if err := applyMemberPointsForSale(ctx, qtx, sale.MemberID, header.ID, sale.RedeemPoints, sale.PointsEarned, actorID); err != nil {
			return nil, err
		}
	}

	// Pengurangan stok dengan penjaga. Nol baris berarti stok tidak cukup.
	for _, d := range sale.Details {
		rows, err := qtx.DecrementProductStock(ctx, db.DecrementProductStockParams{
			Qty:       int32(d.Qty),
			UpdatedBy: actor,
			ID:        d.ProductID,
		})
		if err != nil {
			return nil, err
		}
		if rows == 0 {
			return nil, fmt.Errorf("%w: %s", domain.ErrSaleStokTidakCukup, d.ProductName)
		}
	}

	for _, d := range sale.Details {
		if err := qtx.CreateSaleDetail(ctx, db.CreateSaleDetailParams{
			SaleID:         header.ID,
			ProductID:      d.ProductID,
			Sku:            d.SKU,
			ProductName:    d.ProductName,
			Qty:            int32(d.Qty),
			UnitPrice:      numericFromFloat(d.UnitPrice),
			DiscountType:   d.DiscountType,
			DiscountValue:  numericFromFloat(d.DiscountValue),
			DiscountAmount: numericFromFloat(d.DiscountAmount),
			GrossAmount:    numericFromFloat(d.GrossAmount),
			LineTotal:      numericFromFloat(d.LineTotal),
			DppAmount:      numericFromFloat(d.DppAmount),
			TaxAmount:      numericFromFloat(d.TaxAmount),
			CostUnit:       numericFromFloat4(d.CostUnit),
			CostTotal:      numericFromFloat(d.CostTotal),
		}); err != nil {
			return nil, err
		}
	}

	for _, p := range sale.Payments {
		if err := qtx.CreateSalePayment(ctx, db.CreateSalePaymentParams{
			SaleID:          header.ID,
			PaymentMethodID: p.PaymentMethodID,
			Amount:          numericFromFloat(p.Amount),
			ReferenceNumber: p.ReferenceNumber,
		}); err != nil {
			return nil, err
		}
	}

	if journal != nil && len(journal.Lines) > 0 {
		journalID, err := insertPostedJournal(ctx, qtx, journal, domain.JournalSourceSale, &header.ID, saleNumber, actorID)
		if err != nil {
			return nil, err
		}
		if err := qtx.SetSaleJournal(ctx, db.SetSaleJournalParams{
			JournalEntryID: journalID,
			ID:             header.ID,
		}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, header.ID)
}

func (r *sqlSaleRepository) VoidSale(ctx context.Context, id, actorID int64, reason string, reversal *domain.SaleJournalPlan) (*domain.Sale, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	lock, err := qtx.LockSaleForVoid(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSaleNotFound
		}
		return nil, err
	}
	if lock.Status != domain.SaleStatusPosted {
		if lock.Status == domain.SaleStatusVoid {
			return nil, domain.ErrSaleAlreadyVoid
		}
		return nil, domain.ErrSaleNotPosted
	}

	details, err := qtx.GetSaleDetails(ctx, id)
	if err != nil {
		return nil, err
	}
	actor := int8FromID(actorID)
	for _, d := range details {
		if err := qtx.RestoreProductStock(ctx, db.RestoreProductStockParams{
			Qty:       d.Qty,
			UpdatedBy: actor,
			ID:        d.ProductID,
		}); err != nil {
			return nil, err
		}
	}

	// Batalkan poin yang terlibat pada nota ini (earn ditarik, redeem dikembalikan).
	if lock.MemberID.Valid && (lock.PointsEarned > 0 || lock.RedeemPoints > 0) {
		if err := reverseMemberPointsForSale(ctx, qtx, lock.MemberID.Int64, id, lock.PointsEarned, lock.RedeemPoints, actorID); err != nil {
			return nil, err
		}
	}

	// Jurnal pembalik ditulis dulu; bila jurnal asli sudah pernah dibalik,
	// SetOriginalJournalReversed 0 baris dan transaksi digulung.
	if lock.JournalEntryID.Valid && reversal != nil && len(reversal.Lines) > 0 {
		revID, err := insertPostedJournal(ctx, qtx, reversal, "", nil, lock.SaleNumber, actorID)
		if err != nil {
			return nil, err
		}
		rows, err := qtx.SetOriginalJournalReversed(ctx, db.SetOriginalJournalReversedParams{
			ReversalID: revID,
			ID:         lock.JournalEntryID.Int64,
		})
		if err != nil {
			return nil, err
		}
		if rows == 0 {
			return nil, domain.ErrJournalAlreadyRev
		}
	}

	rows, err := qtx.VoidSale(ctx, db.VoidSaleParams{
		VoidedBy:   actor,
		VoidReason: reason,
		ID:         id,
	})
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, domain.ErrSaleAlreadyVoid
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func mapSaleDetail(row db.GetSaleDetailsRow) domain.SaleDetail {
	return domain.SaleDetail{
		ID:             row.ID,
		SaleID:         row.SaleID,
		ProductID:      row.ProductID,
		SKU:            row.Sku,
		ProductName:    row.ProductName,
		Qty:            int(row.Qty),
		UnitPrice:      numericToFloat(row.UnitPrice),
		DiscountType:   row.DiscountType,
		DiscountValue:  numericToFloat(row.DiscountValue),
		DiscountAmount: numericToFloat(row.DiscountAmount),
		GrossAmount:    numericToFloat(row.GrossAmount),
		LineTotal:      numericToFloat(row.LineTotal),
		DppAmount:      numericToFloat(row.DppAmount),
		TaxAmount:      numericToFloat(row.TaxAmount),
		CostUnit:       numericToFloat(row.CostUnit),
		CostTotal:      numericToFloat(row.CostTotal),
	}
}

func mapSalePayment(row db.GetSalePaymentsRow) domain.SalePayment {
	return domain.SalePayment{
		ID:                row.ID,
		SaleID:            row.SaleID,
		PaymentMethodID:   row.PaymentMethodID,
		PaymentMethodName: row.PaymentMethodName,
		Channel:           row.Channel,
		Amount:            numericToFloat(row.Amount),
		ReferenceNumber:   row.ReferenceNumber,
		CreatedAt:         row.CreatedAt.Time,
	}
}

func (r *sqlSaleRepository) GetByID(ctx context.Context, id int64) (*domain.Sale, error) {
	row, err := r.q.GetSaleByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	sale := &domain.Sale{
		ID:             row.ID,
		SaleNumber:     row.SaleNumber,
		SaleDate:       row.SaleDate.Time,
		MemberName:     row.MemberName,
		TaxType:        row.TaxType,
		Subtotal:       numericToFloat(row.Subtotal),
		ItemDiscount:   numericToFloat(row.ItemDiscount),
		DiscountType:   row.DiscountType,
		DiscountValue:  numericToFloat(row.DiscountValue),
		DiscountAmount: numericToFloat(row.DiscountAmount),
		DppAmount:      numericToFloat(row.DppAmount),
		TaxAmount:      numericToFloat(row.TaxAmount),
		TotalAmount:    numericToFloat(row.TotalAmount),
		PaidAmount:     numericToFloat(row.PaidAmount),
		ChangeAmount:   numericToFloat(row.ChangeAmount),
		CostTotal:      numericToFloat(row.CostTotal),
		PointsEarned:   row.PointsEarned,
		RedeemPoints:   row.RedeemPoints,
		RedeemAmount:   numericToFloat(row.RedeemAmount),
		PaymentStatus:  row.PaymentStatus,
		Status:         row.Status,
		Notes:          row.Notes,
		CreatedAt:      row.CreatedAt.Time,
	}
	if row.MemberID.Valid {
		sale.MemberID = row.MemberID.Int64
	}
	if row.DueDate.Valid {
		t := row.DueDate.Time
		sale.DueDate = &t
	}
	if row.JournalEntryID.Valid {
		v := row.JournalEntryID.Int64
		sale.JournalEntryID = &v
	}
	if row.VoidedAt.Valid {
		t := row.VoidedAt.Time
		sale.VoidedAt = &t
	}
	sale.VoidReason = row.VoidReason
	if row.CreatedBy.Valid {
		v := row.CreatedBy.Int64
		sale.CreatedBy = &v
	}

	detailRows, err := r.q.GetSaleDetails(ctx, id)
	if err != nil {
		return nil, err
	}
	sale.Details = make([]domain.SaleDetail, 0, len(detailRows))
	for _, d := range detailRows {
		sale.Details = append(sale.Details, mapSaleDetail(d))
	}

	paymentRows, err := r.q.GetSalePayments(ctx, id)
	if err != nil {
		return nil, err
	}
	sale.Payments = make([]domain.SalePayment, 0, len(paymentRows))
	for _, p := range paymentRows {
		sale.Payments = append(sale.Payments, mapSalePayment(p))
	}

	return sale, nil
}

func mapSaleListRow(id int64, saleNumber string, saleDate pgtype.Date, memberID pgtype.Int8, memberName string, total, paid pgtype.Numeric, paymentStatus, status, taxType string, createdAt pgtype.Timestamptz) domain.Sale {
	sale := domain.Sale{
		ID:            id,
		SaleNumber:    saleNumber,
		SaleDate:      saleDate.Time,
		MemberName:    memberName,
		TotalAmount:   numericToFloat(total),
		PaidAmount:    numericToFloat(paid),
		PaymentStatus: paymentStatus,
		Status:        status,
		TaxType:       taxType,
		CreatedAt:     createdAt.Time,
	}
	if memberID.Valid {
		sale.MemberID = memberID.Int64
	}
	return sale
}

func (r *sqlSaleRepository) GetListCursor(ctx context.Context, param domain.SaleListParam) ([]domain.Sale, int64, error) {
	limit := param.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	statusFilter := param.StatusFilter
	if statusFilter == "" {
		statusFilter = "ALL"
	}
	total, err := r.q.CountSales(ctx, db.CountSalesParams{
		SearchQuery:  param.Search,
		StatusFilter: statusFilter,
	})
	if err != nil {
		return nil, 0, err
	}

	if param.Direction == "prev" {
		rows, err := r.q.GetSalesCursorPrev(ctx, db.GetSalesCursorPrevParams{
			SearchQuery:  param.Search,
			StatusFilter: statusFilter,
			CursorID:     param.CursorID,
			ReqLimit:     int32(limit),
		})
		if err != nil {
			return nil, 0, err
		}
		results := make([]domain.Sale, 0, len(rows))
		for _, v := range rows {
			results = append(results, mapSaleListRow(v.ID, v.SaleNumber, v.SaleDate, v.MemberID, v.MemberName, v.TotalAmount, v.PaidAmount, v.PaymentStatus, v.Status, v.TaxType, v.CreatedAt))
		}
		for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
			results[i], results[j] = results[j], results[i]
		}
		return results, total, nil
	}

	rows, err := r.q.GetSalesCursorNext(ctx, db.GetSalesCursorNextParams{
		SearchQuery:  param.Search,
		StatusFilter: statusFilter,
		CursorID:     param.CursorID,
		ReqLimit:     int32(limit),
	})
	if err != nil {
		return nil, 0, err
	}
	results := make([]domain.Sale, 0, len(rows))
	for _, v := range rows {
		results = append(results, mapSaleListRow(v.ID, v.SaleNumber, v.SaleDate, v.MemberID, v.MemberName, v.TotalAmount, v.PaidAmount, v.PaymentStatus, v.Status, v.TaxType, v.CreatedAt))
	}
	return results, total, nil
}

func (r *sqlSaleRepository) LookupProductsForPOS(ctx context.Context, search string, categoryID, limit int) ([]domain.POSProduct, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.q.LookupProductsForPOS(ctx, db.LookupProductsForPOSParams{
		SearchQuery: search,
		CategoryID:  int32(categoryID),
		ReqLimit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.POSProduct, 0, len(rows))
	for _, v := range rows {
		result = append(result, domain.POSProduct{
			ID:            v.ID,
			SKU:           v.Sku,
			Barcode:       v.Barcode.String,
			Name:          v.Name,
			SellingPrice:  numericToFloat(v.SellingPrice),
			CurrentStock:  int(v.CurrentStock),
			CategoryName:  v.CategoryName,
			SellingUnit:   v.SellingUnit,
			DiscountType:  v.DiscountType,
			DiscountValue: numericToFloat(v.DiscountValue),
			IsTaxable:     v.IsTaxable,
		})
	}
	return result, nil
}

func (r *sqlSaleRepository) GetActiveCategories(ctx context.Context) ([]domain.POSCategory, error) {
	rows, err := r.q.GetActiveProductCategories(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.POSCategory, 0, len(rows))
	for _, v := range rows {
		result = append(result, domain.POSCategory{ID: v.ID, Name: v.Name})
	}
	return result, nil
}
