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
		TotalDebit:      numericFromMoney(plan.TotalDebit),
		TotalCredit:     numericFromMoney(plan.TotalCredit),
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
			Debit:          numericFromMoney(l.Debit),
			Credit:         numericFromMoney(l.Credit),
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
			SellingPrice:  moneyFromNumeric(v.SellingPrice),
			AvgCost:       decimalFromNumeric(v.AvgCost),
			CurrentStock:  int(v.CurrentStock),
			DiscountType:  v.DiscountType,
			DiscountValue: decimalFromNumeric(v.DiscountValue),
			IsTaxable:     v.IsTaxable,
		}
	}
	return result, nil
}

func (r *sqlSaleRepository) CreateSale(ctx context.Context, sale *domain.Sale, journal *domain.SaleJournalPlan) (*domain.Sale, error) {
	if len(sale.Details) == 0 {
		return nil, domain.ErrSaleItemsRequired
	}
	// Validasi ini juga ada di domain. Pertahanan di repository memastikan
	// pemanggil yang melewati CalculateSale tidak bisa menambah stok lewat qty
	// negatif (current_stock - (-qty)).
	for _, d := range sale.Details {
		if d.Qty <= 0 {
			return nil, domain.ErrSaleLineInvalid
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)

	// Kunci seluruh produk yang dijual supaya validasi stok final tidak bisa
	// dilewati oleh penjualan bersamaan produk yang sama. Bandingkan snapshot
	// yang dipakai menghitung nota dengan baris terkunci; bila harga, diskon,
	// pajak, atau HPP berubah, minta usecase menghitung ulang.
	ids := uniqueProductIDs(sale.Details)
	locks, err := qtx.LockProductsForSale(ctx, ids)
	if err != nil {
		return nil, err
	}
	locked := make(map[int64]domain.SaleProductInfo, len(locks))
	for _, l := range locks {
		locked[l.ID] = domain.SaleProductInfo{
			ID:            l.ID,
			SKU:           l.Sku,
			Name:          l.Name,
			SellingPrice:  moneyFromNumeric(l.SellingPrice),
			AvgCost:       decimalFromNumeric(l.AvgCost),
			CurrentStock:  int(l.CurrentStock),
			DiscountType:  l.DiscountType,
			DiscountValue: decimalFromNumeric(l.DiscountValue),
			IsTaxable:     l.IsTaxable,
		}
	}
	for _, d := range sale.Details {
		p, ok := locked[d.ProductID]
		if !ok {
			return nil, fmt.Errorf("%w: id %d", domain.ErrSaleProductNotFound, d.ProductID)
		}
		if saleProductSnapshotChanged(d, p) {
			return nil, fmt.Errorf("%w: %s", domain.ErrSaleSnapshotChanged, d.ProductName)
		}
	}

	var actorID int64
	if sale.CreatedBy != nil {
		actorID = *sale.CreatedBy
	}

	// Validasi voucher di bawah kunci baris. Pemeriksaan usecase bisa saja
	// sudah basi, sehingga status single-use, masa berlaku, dan minimal
	// belanja diperiksa ulang tepat sebelum nota ditulis.
	if sale.VoucherID > 0 {
		lock, err := qtx.LockVoucherForSale(ctx, sale.VoucherID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, domain.ErrVoucherNotFound
			}
			return nil, err
		}
		voucher := voucherFromLockRow(lock)
		if err := voucher.ValidateUsable(time.Now()); err != nil {
			return nil, err
		}
		if err := voucher.ValidateSubtotal(sale.Subtotal); err != nil {
			return nil, err
		}
		usedRows, err := qtx.MarkVoucherUsed(ctx, db.MarkVoucherUsedParams{
			UpdatedBy: int8FromID(actorID),
			ID:        sale.VoucherID,
		})
		if err != nil {
			return nil, err
		}
		if usedRows == 0 {
			return nil, domain.ErrVoucherUsed
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

	actor := int8FromID(actorID)

	var dueDate pgtype.Date
	if sale.DueDate != nil && !sale.DueDate.IsZero() {
		dueDate = pgtype.Date{Time: *sale.DueDate, Valid: true}
	}

	var memberID pgtype.Int8
	if sale.MemberID > 0 {
		memberID = pgtype.Int8{Int64: sale.MemberID, Valid: true}
	}

	var voucherID pgtype.Int8
	if sale.VoucherID > 0 {
		voucherID = pgtype.Int8{Int64: sale.VoucherID, Valid: true}
	}

	header, err := qtx.CreateSale(ctx, db.CreateSaleParams{
		SaleNumber:      saleNumber,
		SaleDate:        pgtype.Date{Time: sale.SaleDate, Valid: true},
		MemberID:        memberID,
		TaxType:         sale.TaxType,
		Subtotal:        numericFromMoney(sale.Subtotal),
		ItemDiscount:    numericFromMoney(sale.ItemDiscount),
		DiscountType:    sale.DiscountType,
		DiscountValue:   numericFromDecimal(sale.DiscountValue),
		DiscountAmount:  numericFromMoney(sale.DiscountAmount),
		DppAmount:       numericFromMoney(sale.DppAmount),
		TaxAmount:       numericFromMoney(sale.TaxAmount),
		TotalAmount:     numericFromMoney(sale.TotalAmount),
		PaidAmount:      numericFromMoney(sale.PaidAmount),
		ChangeAmount:    numericFromMoney(sale.ChangeAmount),
		CostTotal:       numericFromMoney(sale.CostTotal),
		PointsEarned:    sale.PointsEarned,
		RedeemPoints:    sale.RedeemPoints,
		RedeemAmount:    numericFromMoney(sale.RedeemAmount),
		VoucherID:       voucherID,
		VoucherCode:     sale.VoucherCode,
		VoucherDiscount: numericFromMoney(sale.VoucherDiscount),
		PaymentStatus:   sale.PaymentStatus,
		Status:          domain.SaleStatusPosted,
		DueDate:         dueDate,
		Notes:           sale.Notes,
		CreatedBy:       actor,
		UpdatedBy:       actor,
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

	// Pengurangan stok seluruh baris dalam satu UPDATE ... FROM. Penjaga
	// current_stock >= qty tetap berlaku di SQL; selisih jumlah baris berarti
	// stok salah satu produk tidak cukup.
	productIDs := make([]int64, 0, len(sale.Details))
	qtys := make([]int32, 0, len(sale.Details))
	need := make(map[int64]int, len(sale.Details))
	for _, d := range sale.Details {
		productIDs = append(productIDs, d.ProductID)
		qtys = append(qtys, int32(d.Qty))
		need[d.ProductID] += d.Qty
	}
	applied, err := qtx.DecrementProductStocksForSale(ctx, db.DecrementProductStocksForSaleParams{
		ProductIds: productIDs,
		Qtys:       qtys,
		UpdatedBy:  actor,
	})
	if err != nil {
		return nil, err
	}
	if applied.ExpectedCount != applied.UpdatedCount {
		// Sebut produk pertama yang stoknya kurang agar pesan tetap ramah.
		for id, qty := range need {
			p := locked[id]
			if p.CurrentStock < qty {
				return nil, fmt.Errorf("%w: %s (tersedia %d)", domain.ErrSaleStokTidakCukup, p.Name, p.CurrentStock)
			}
		}
		return nil, fmt.Errorf("%w: stok produk berubah saat transaksi", domain.ErrSaleStokTidakCukup)
	}

	for _, d := range sale.Details {
		if err := qtx.CreateSaleDetail(ctx, db.CreateSaleDetailParams{
			SaleID:         header.ID,
			ProductID:      d.ProductID,
			Sku:            d.SKU,
			ProductName:    d.ProductName,
			Qty:            int32(d.Qty),
			UnitPrice:      numericFromMoney(d.UnitPrice),
			DiscountType:   d.DiscountType,
			DiscountValue:  numericFromDecimal(d.DiscountValue),
			DiscountAmount: numericFromMoney(d.DiscountAmount),
			GrossAmount:    numericFromMoney(d.GrossAmount),
			LineTotal:      numericFromMoney(d.LineTotal),
			DppAmount:      numericFromMoney(d.DppAmount),
			TaxAmount:      numericFromMoney(d.TaxAmount),
			CostUnit:       numericFromDecimal(d.CostUnit),
			CostTotal:      numericFromMoney(d.CostTotal),
		}); err != nil {
			return nil, err
		}
	}

	for _, p := range sale.Payments {
		if err := qtx.CreateSalePayment(ctx, db.CreateSalePaymentParams{
			SaleID:          header.ID,
			PaymentMethodID: p.PaymentMethodID,
			Amount:          numericFromMoney(p.Amount),
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
	defer func() { _ = tx.Rollback(ctx) }()

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

	// Kembalikan stok pada HPP asli penjualan. Rata-rata tertimbang dipakai
	// karena avg_cost produk bisa sudah berubah sejak nota dibuat.
	type restock struct {
		qty       int32
		costTotal domain.Money
	}
	restocks := make(map[int64]restock, len(details))
	productIDs := make([]int64, 0, len(details))
	for _, d := range details {
		r := restocks[d.ProductID]
		if r.qty == 0 {
			productIDs = append(productIDs, d.ProductID)
		}
		r.qty += d.Qty
		r.costTotal = r.costTotal.Add(moneyFromNumeric(d.CostTotal))
		restocks[d.ProductID] = r
	}
	productLocks, err := qtx.LockProductsForVoid(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	if len(productLocks) != len(productIDs) {
		return nil, domain.ErrSaleProductNotFound
	}
	actor := int8FromID(actorID)
	restoreIDs := make([]int64, 0, len(productLocks))
	restoreQtys := make([]int32, 0, len(productLocks))
	restoreCosts := make([]pgtype.Numeric, 0, len(productLocks))
	for _, p := range productLocks {
		r := restocks[p.ID]
		restoreIDs = append(restoreIDs, p.ID)
		restoreQtys = append(restoreQtys, r.qty)
		restoreCosts = append(restoreCosts, numericFromMoney(r.costTotal))
	}
	restored, err := qtx.RestoreProductStocksWithValuation(ctx, db.RestoreProductStocksWithValuationParams{
		ProductIds: restoreIDs,
		Qtys:       restoreQtys,
		CostTotals: restoreCosts,
		UpdatedBy:  actor,
	})
	if err != nil {
		return nil, err
	}
	if restored != int64(len(restoreIDs)) {
		return nil, domain.ErrSaleProductNotFound
	}

	// Batalkan poin yang terlibat pada nota ini (earn ditarik, redeem dikembalikan).
	if lock.MemberID.Valid && (lock.PointsEarned > 0 || lock.RedeemPoints > 0) {
		if err := reverseMemberPointsForSale(ctx, qtx, lock.MemberID.Int64, id, lock.PointsEarned, lock.RedeemPoints, actorID); err != nil {
			return nil, err
		}
	}

	// Voucher sekali pakai dikembalikan agar dapat digunakan lagi setelah nota
	// dibatalkan.
	if lock.VoucherID.Valid {
		if err := qtx.ReleaseVoucher(ctx, db.ReleaseVoucherParams{
			UpdatedBy: actor,
			ID:        lock.VoucherID.Int64,
		}); err != nil {
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

// voucherFromLockRow memetakan baris voucher terkunci menjadi domain.Voucher.
func voucherFromLockRow(v db.LockVoucherForSaleRow) *domain.Voucher {
	return &domain.Voucher{
		ID:            v.ID,
		Code:          v.Code,
		DiscountType:  v.DiscountType,
		DiscountValue: decimalFromNumeric(v.DiscountValue),
		MinPurchase:   moneyFromNumeric(v.MinPurchase),
		MaxDiscount:   moneyFromNumeric(v.MaxDiscount),
		ValidFrom:     dateOnlyFromPg(v.ValidFrom),
		ValidUntil:    dateOnlyFromPg(v.ValidUntil),
		IsActive:      v.IsActive,
		UsedAt:        convertPgTimeZ(v.UsedAt),
	}
}

// saleProductSnapshotChanged mendeteksi perubahan nilai yang memengaruhi total
// nota atau HPP antara snapshot pra-transaksi dan baris produk yang terkunci.
func saleProductSnapshotChanged(d domain.SaleDetail, p domain.SaleProductInfo) bool {
	if d.UnitPrice.Cmp(p.SellingPrice) != 0 {
		return true
	}
	if d.DiscountType != p.DiscountType || d.DiscountValue.Cmp(p.DiscountValue) != 0 {
		return true
	}
	if d.CostUnit.Cmp(p.AvgCost) != 0 {
		return true
	}
	return d.IsTaxable != p.IsTaxable
}

func mapSaleDetail(row db.GetSaleDetailsRow) domain.SaleDetail {
	return domain.SaleDetail{
		ID:             row.ID,
		SaleID:         row.SaleID,
		ProductID:      row.ProductID,
		SKU:            row.Sku,
		ProductName:    row.ProductName,
		Qty:            int(row.Qty),
		UnitPrice:      moneyFromNumeric(row.UnitPrice),
		DiscountType:   row.DiscountType,
		DiscountValue:  decimalFromNumeric(row.DiscountValue),
		DiscountAmount: moneyFromNumeric(row.DiscountAmount),
		GrossAmount:    moneyFromNumeric(row.GrossAmount),
		LineTotal:      moneyFromNumeric(row.LineTotal),
		DppAmount:      moneyFromNumeric(row.DppAmount),
		TaxAmount:      moneyFromNumeric(row.TaxAmount),
		CostUnit:       decimalFromNumeric(row.CostUnit),
		CostTotal:      moneyFromNumeric(row.CostTotal),
	}
}

func mapSalePayment(row db.GetSalePaymentsRow) domain.SalePayment {
	return domain.SalePayment{
		ID:                row.ID,
		SaleID:            row.SaleID,
		PaymentMethodID:   row.PaymentMethodID,
		PaymentMethodName: row.PaymentMethodName,
		Channel:           row.Channel,
		Amount:            moneyFromNumeric(row.Amount),
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
		ID:              row.ID,
		SaleNumber:      row.SaleNumber,
		SaleDate:        row.SaleDate.Time,
		MemberName:      row.MemberName,
		TaxType:         row.TaxType,
		Subtotal:        moneyFromNumeric(row.Subtotal),
		ItemDiscount:    moneyFromNumeric(row.ItemDiscount),
		DiscountType:    row.DiscountType,
		DiscountValue:   decimalFromNumeric(row.DiscountValue),
		DiscountAmount:  moneyFromNumeric(row.DiscountAmount),
		DppAmount:       moneyFromNumeric(row.DppAmount),
		TaxAmount:       moneyFromNumeric(row.TaxAmount),
		TotalAmount:     moneyFromNumeric(row.TotalAmount),
		PaidAmount:      moneyFromNumeric(row.PaidAmount),
		ChangeAmount:    moneyFromNumeric(row.ChangeAmount),
		CostTotal:       moneyFromNumeric(row.CostTotal),
		PointsEarned:    row.PointsEarned,
		RedeemPoints:    row.RedeemPoints,
		RedeemAmount:    moneyFromNumeric(row.RedeemAmount),
		VoucherCode:     row.VoucherCode,
		VoucherDiscount: moneyFromNumeric(row.VoucherDiscount),
		PaymentStatus:   row.PaymentStatus,
		Status:          row.Status,
		Notes:           row.Notes,
		CreatedAt:       row.CreatedAt.Time,
	}
	if row.MemberID.Valid {
		sale.MemberID = row.MemberID.Int64
	}
	if row.VoucherID.Valid {
		sale.VoucherID = row.VoucherID.Int64
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

func mapSaleListRow(id int64, saleNumber string, saleDate pgtype.Date, memberID pgtype.Int8, memberName string, total, paid pgtype.Numeric, voucherCode, paymentStatus, status, taxType string, createdAt pgtype.Timestamptz) domain.Sale {
	sale := domain.Sale{
		ID:            id,
		SaleNumber:    saleNumber,
		SaleDate:      saleDate.Time,
		MemberName:    memberName,
		TotalAmount:   moneyFromNumeric(total),
		PaidAmount:    moneyFromNumeric(paid),
		VoucherCode:   voucherCode,
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
			results = append(results, mapSaleListRow(v.ID, v.SaleNumber, v.SaleDate, v.MemberID, v.MemberName, v.TotalAmount, v.PaidAmount, v.VoucherCode, v.PaymentStatus, v.Status, v.TaxType, v.CreatedAt))
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
		results = append(results, mapSaleListRow(v.ID, v.SaleNumber, v.SaleDate, v.MemberID, v.MemberName, v.TotalAmount, v.PaidAmount, v.VoucherCode, v.PaymentStatus, v.Status, v.TaxType, v.CreatedAt))
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
			SellingPrice:  moneyFromNumeric(v.SellingPrice),
			CurrentStock:  int(v.CurrentStock),
			CategoryName:  v.CategoryName,
			SellingUnit:   v.SellingUnit,
			DiscountType:  v.DiscountType,
			DiscountValue: decimalFromNumeric(v.DiscountValue),
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
