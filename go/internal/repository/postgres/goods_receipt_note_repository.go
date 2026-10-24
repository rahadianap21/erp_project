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

type sqlGRNRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewSQLGoodsReceiptNoteRepository(pool *pgxpool.Pool) domain.GoodsReceiptNoteRepository {
	return &sqlGRNRepository{pool: pool, q: db.New(pool)}
}

// poLine merepresentasikan baris PO yang terkunci beserta data satuan produk
// yang dibutuhkan untuk menentukan faktor konversi saat penerimaan.
type poLine struct {
	ProductID             int64
	QtyOrdered            int32
	QtyReceived           int32
	PurchaseUnitID        int32
	ProductPurchaseUnitID int32
	ProductSellingUnitID  int32
	ConversionRate        int32
}

// sellingRate menentukan berapa satuan jualequivalent dari satu satuan beli
// yang dipakai baris PO tersebut.
//
// undertakenBaris PO menyimpan purchase_unit_id milik produk. Kalau baris memakai
// satuan jual produk, konversinya 1. Kalau memakai satuan beli produk, konversinya
// mengikuti conversion_rate. Pair lain ditolak karena tidak ada faktor konversi
// yang bisa dittrusted.
func sellingRate(l poLine) (int32, error) {
	switch l.PurchaseUnitID {
	case l.ProductSellingUnitID:
		return 1, nil
	case l.ProductPurchaseUnitID:
		if l.ConversionRate < 1 {
			return 1, nil
		}
		return l.ConversionRate, nil
	default:
		return 0, domain.ErrGRNSatuanBermasalah
	}
}

func (r *sqlGRNRepository) CreateGRN(ctx context.Context, in domain.CreateGRNInput) (*domain.GoodsReceiptNote, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)

	// Kunci header PO dulu supaya dua penerimaan bersamaan untuk PO yang sama
	// tidak bisa membaca sisa qty yang sama lalu sama-sama incremented.
	lock, err := qtx.LockPurchaseOrderForReceiving(ctx, in.PurchaseOrderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrGRNPONotFound
		}
		return nil, err
	}
	if lock.Status != "APPROVED" {
		return nil, domain.ErrGRNPOSalahStatus
	}

	if len(in.Items) == 0 {
		return nil, domain.ErrGRNPenerimaanKosong
	}

	detailRows, err := qtx.GetPurchaseOrderDetailLines(ctx, in.PurchaseOrderID)
	if err != nil {
		return nil, err
	}
	lines := make(map[int64]poLine, len(detailRows))
	for _, d := range detailRows {
		lines[d.ID] = poLine{
			ProductID:             d.ProductID,
			QtyOrdered:            d.QtyOrdered,
			QtyReceived:           d.QtyReceived,
			PurchaseUnitID:        d.PurchaseUnitID,
			ProductPurchaseUnitID: d.ProductPurchaseUnitID,
			ProductSellingUnitID:  d.ProductSellingUnitID,
			ConversionRate:        d.ConversionRate,
		}
	}

	// Validasi seluruh baris sebelum menulis apa pun. Stok TIDAK disentuh di sini;
	// stok baru ditambahkan saat penerimaan ditutup (lihat FinishReceiving).
	totalQty := int32(0)
	rates := make(map[int64]int32, len(in.Items))
	for _, it := range in.Items {
		if it.QtyReceived <= 0 {
			return nil, domain.ErrGRNBarisTidakValid
		}
		l, ok := lines[it.PurchaseOrderDetailID]
		if !ok {
			return nil, domain.ErrGRNBarisPONotFound
		}
		// Sisa dibaca dari snapshot yang baru dikunci, bukan dari nilai kiriman.
		if it.QtyReceived > l.QtyOrdered-l.QtyReceived {
			return nil, domain.ErrGRNOverReceipt
		}
		rate, err := sellingRate(l)
		if err != nil {
			return nil, err
		}
		rates[it.PurchaseOrderDetailID] = rate
		totalQty += it.QtyReceived
	}
	if totalQty <= 0 {
		return nil, domain.ErrGRNPenerimaanKosong
	}

	now := time.Now()
	datePrefix := now.Format("20060102")
	nextSeq, err := qtx.NextGRNSeq(ctx, datePrefix)
	if err != nil {
		return nil, err
	}
	grnNumber := fmt.Sprintf("GRN/%s/%04d", datePrefix, nextSeq)

	receivedDate := now
	if in.ReceivedDate != nil && !in.ReceivedDate.IsZero() {
		receivedDate = in.ReceivedDate.Time
	}
	var notesPg pgtype.Text
	if in.Notes != "" {
		notesPg = pgtype.Text{String: in.Notes, Valid: true}
	}
	var userPg pgtype.Int8
	if in.CreatedBy > 0 {
		userPg = pgtype.Int8{Int64: in.CreatedBy, Valid: true}
	}

	header, err := qtx.CreateGRN(ctx, db.CreateGRNParams{
		GrnNumber:       grnNumber,
		PurchaseOrderID: in.PurchaseOrderID,
		ReceivedDate:    pgtype.Date{Time: receivedDate, Valid: true},
		TotalQty:        totalQty,
		Notes:           notesPg,
		CreatedBy:       userPg,
		UpdatedBy:       userPg,
	})
	if err != nil {
		return nil, err
	}

	for _, it := range in.Items {
		l := lines[it.PurchaseOrderDetailID]
		rate := rates[it.PurchaseOrderDetailID]
		if err := qtx.CreateGRNDetail(ctx, db.CreateGRNDetailParams{
			GrnID:                 header.ID,
			PurchaseOrderDetailID: it.PurchaseOrderDetailID,
			ProductID:             l.ProductID,
			QtyReceived:           it.QtyReceived,
			ConversionRate:        rate,
		}); err != nil {
			return nil, err
		}
		// Penjaga di SQL (qty_received + qty <= qty_ordered) yang menjadi penentu
		// terakhir; error di sini berarti bentrok dengan penerimaan lain.
		if _, err := qtx.IncrementDetailQtyReceived(ctx, db.IncrementDetailQtyReceivedParams{
			ID:       it.PurchaseOrderDetailID,
			QtyToAdd: it.QtyReceived,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, domain.ErrGRNOverReceipt
			}
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &domain.GoodsReceiptNote{
		ID:              header.ID,
		GRNNumber:       header.GrnNumber,
		PurchaseOrderID: header.PurchaseOrderID,
		ReceivedDate:    header.ReceivedDate.Time,
		TotalQty:        header.TotalQty,
		CreatedAt:       header.CreatedAt.Time,
	}, nil
}

// FinishReceiving menutup penerimaan sebuah PO dalam SATU transaksi: kunci
// header PO, tambahkan stok dari seluruh GRN-nya, lalu ubah status ke RECEIVED.
//
// Stok baru ditambahkan di sini (bukan saat CreateGRN) supaya barang yang sudah
// diterima belum masuk current_stock sampai penerimaan benar-benar ditutup.
// Karena transisi status APPROVED->RECEIVED hanya bisa menang sekali, penambahan
// stok pun otomatis tidak pernah dobel.
//
// expectedGRNCount adalah daftar GRN yang jurnalnya sudah diterbitkan usecase.
// Setelah header dikunci, jumlah GRN di database harus sama. Kalau tidak,
// berarti ada CreateGRN yang menang balapan di antara pembacaan usecase dan lock
// ini; GRN tersebut belum berjurnal, jadi penutupan dibatalkan. CreateGRN yang
// datang setelah lock akan menunggu, lalu ditolak karena status sudah RECEIVED.
//
// netCostFactor mengubah nilai terima bruto menjadi harga pokok perolehan per
// produk. Faktor ini diteruskan sebagai NUMERIC 6 desimal supaya pembagian
// TAX_INCLUSIVE (1/1.12) tidak kehilangan presisi sebelum sampai ke query.
func (r *sqlGRNRepository) FinishReceiving(ctx context.Context, poID int64, updatedBy int64, netCostFactor float64, expectedGRNCount int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)

	lock, err := qtx.LockPurchaseOrderForReceiving(ctx, poID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrGRNPONotFound
		}
		return err
	}
	if lock.Status != "APPROVED" {
		return domain.ErrGRNPOSalahStatus
	}

	currentGRNCount, err := qtx.CountGRNsByPOID(ctx, poID)
	if err != nil {
		return err
	}
	if currentGRNCount != int64(expectedGRNCount) {
		return domain.ErrGRNReceiptChanged
	}

	var updatedByPg pgtype.Int8
	if updatedBy > 0 {
		updatedByPg = pgtype.Int8{Int64: updatedBy, Valid: true}
	}

	var netCostFactorPg pgtype.Numeric
	if err := netCostFactorPg.Scan(fmt.Sprintf("%.6f", netCostFactor)); err != nil {
		return err
	}

	deltas, err := qtx.GetPOReceivedDeltas(ctx, db.GetPOReceivedDeltasParams{
		PurchaseOrderID: poID,
		NetCostFactor:   netCostFactorPg,
	})
	if err != nil {
		return err
	}
	for _, d := range deltas {
		if d.DeltaSelling <= 0 {
			continue
		}
		if _, err := qtx.IncrementProductStock(ctx, db.IncrementProductStockParams{
			ID:           d.ProductID,
			DeltaSelling: d.DeltaSelling,
			AddedValue:   d.NetValue,
			UpdatedBy:    updatedByPg,
		}); err != nil {
			return err
		}
	}

	rows, err := qtx.TransitionPurchaseOrderStatus(ctx, db.TransitionPurchaseOrderStatusParams{
		NewStatus:  "RECEIVED",
		UpdatedBy:  updatedByPg,
		ID:         poID,
		AllowFrom:  []string{"APPROVED"},
		ForbidFrom: []string{"RECEIVED", "REJECTED"},
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrGRNPOSalahStatus
	}

	return tx.Commit(ctx)
}

// mapGRNRow memetakan baris query GRN yang sudah dilengkapi nomor PO ke domain.
func mapGRNRow(id int64, grnNumber string, poID int64, receivedDate pgtype.Date, totalQty int32, notes pgtype.Text, createdAt pgtype.Timestamp, createdBy pgtype.Int8, poNumber string) domain.GoodsReceiptNote {
	var createdByPtr *int64
	if createdBy.Valid {
		v := createdBy.Int64
		createdByPtr = &v
	}
	return domain.GoodsReceiptNote{
		ID:              id,
		GRNNumber:       grnNumber,
		PurchaseOrderID: poID,
		PONumber:        poNumber,
		ReceivedDate:    receivedDate.Time,
		TotalQty:        totalQty,
		Notes:           notes.String,
		CreatedAt:       createdAt.Time,
		CreatedBy:       createdByPtr,
	}
}

func (r *sqlGRNRepository) GetListCursor(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]domain.GoodsReceiptNote, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	total, err := r.q.CountGRNsTotal(ctx, search)
	if err != nil {
		return nil, 0, err
	}

	if direction == "prev" {
		prevRows, err := r.q.GetGRNsCursorPrev(ctx, db.GetGRNsCursorPrevParams{
			SearchQuery: search,
			CursorID:    cursorID,
			ReqLimit:    int32(limit),
		})
		if err != nil {
			return nil, 0, err
		}
		results := make([]domain.GoodsReceiptNote, 0, len(prevRows))
		for _, v := range prevRows {
			results = append(results, mapGRNRow(v.ID, v.GrnNumber, v.PurchaseOrderID, v.ReceivedDate, v.TotalQty, v.Notes, v.CreatedAt, v.CreatedBy, v.PoNumber))
		}
		// Query prev berjalan menaik agar dapat memakai indeks, lalu dibalik
		// supaya urutan halaman tetap sama dengan query next.
		for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
			results[i], results[j] = results[j], results[i]
		}
		return results, total, nil
	}

	nextRows, err := r.q.GetGRNsCursorNext(ctx, db.GetGRNsCursorNextParams{
		SearchQuery: search,
		CursorID:    cursorID,
		ReqLimit:    int32(limit),
	})
	if err != nil {
		return nil, 0, err
	}
	results := make([]domain.GoodsReceiptNote, 0, len(nextRows))
	for _, v := range nextRows {
		results = append(results, mapGRNRow(v.ID, v.GrnNumber, v.PurchaseOrderID, v.ReceivedDate, v.TotalQty, v.Notes, v.CreatedAt, v.CreatedBy, v.PoNumber))
	}
	return results, total, nil
}

func (r *sqlGRNRepository) GetByID(ctx context.Context, id int64) (*domain.GoodsReceiptNote, error) {
	if id <= 0 {
		return nil, errors.New("id grn tidak valid")
	}
	row, err := r.q.GetGRNByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	res := mapGRNRow(row.ID, row.GrnNumber, row.PurchaseOrderID, row.ReceivedDate, row.TotalQty, row.Notes, row.CreatedAt, row.CreatedBy, row.PoNumber)

	detailRows, err := r.q.GetGRNDetails(ctx, id)
	if err != nil {
		return nil, err
	}
	res.Items = make([]domain.GoodsReceiptNoteDetail, 0, len(detailRows))
	for _, d := range detailRows {
		res.Items = append(res.Items, domain.GoodsReceiptNoteDetail{
			ID:                    d.ID,
			GRNID:                 d.GrnID,
			PurchaseOrderDetailID: d.PurchaseOrderDetailID,
			ProductID:             d.ProductID,
			SKU:                   d.Sku,
			ProductName:           d.ProductName,
			PurchaseUnitKode:      d.PurchaseUnitKode,
			QtyReceived:           d.QtyReceived,
			ConversionRate:        d.ConversionRate,
		})
	}
	return &res, nil
}

func (r *sqlGRNRepository) GetByPOID(ctx context.Context, poID int64) ([]domain.GoodsReceiptNote, error) {
	if poID <= 0 {
		return nil, errors.New("id po tidak valid")
	}
	rows, err := r.q.GetGRNsByPOID(ctx, poID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []domain.GoodsReceiptNote{}, nil
	}

	// Detail seluruh GRN dibaca sekali untuk PO ini. Versi sebelumnya memanggil
	// GetByID per GRN, sehingga biaya query tumbuh linear terhadap jumlah GRN.
	detailRows, err := r.q.GetGRNDetailsByPOID(ctx, poID)
	if err != nil {
		return nil, err
	}
	detailsByGRN := make(map[int64][]domain.GoodsReceiptNoteDetail, len(rows))
	for _, d := range detailRows {
		detailsByGRN[d.GrnID] = append(detailsByGRN[d.GrnID], domain.GoodsReceiptNoteDetail{
			ID:                    d.ID,
			GRNID:                 d.GrnID,
			PurchaseOrderDetailID: d.PurchaseOrderDetailID,
			ProductID:             d.ProductID,
			SKU:                   d.Sku,
			ProductName:           d.ProductName,
			PurchaseUnitKode:      d.PurchaseUnitKode,
			QtyReceived:           d.QtyReceived,
			ConversionRate:        d.ConversionRate,
		})
	}

	results := make([]domain.GoodsReceiptNote, 0, len(rows))
	for _, v := range rows {
		grn := mapGRNRow(v.ID, v.GrnNumber, v.PurchaseOrderID, v.ReceivedDate, v.TotalQty, v.Notes, v.CreatedAt, v.CreatedBy, "")
		grn.Items = detailsByGRN[grn.ID]
		results = append(results, grn)
	}
	return results, nil
}
