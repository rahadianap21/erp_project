package postgres

import (
	"context"
	"fmt"
	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	// 🌟 SAMAKAN DENGAN MASTER: Menggunakan pgxpool murni
)

type sqlPurchaseOrderRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewSQLPurchaseOrderRepository menginisialisasi implementasi PO berbasis pgxpool
func NewSQLPurchaseOrderRepository(pool *pgxpool.Pool) domain.PurchaseOrderRepository {
	return &sqlPurchaseOrderRepository{pool: pool, q: db.New(pool)}
}

func (r *sqlPurchaseOrderRepository) Create(ctx context.Context, po *domain.PurchaseOrder) error {
	// Membuka transaksi standard native pgxpool
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Injeksi scope transaksi pgx murni ke Queries SQLC
	qtx := r.q.WithTx(tx)

	// 1. Eksekusi Header PO via Kueri SQLC
	dppNumeric := numericFromFloat(po.DppAmount)
	taxNumeric := numericFromFloat(po.TaxAmount)
	totalNumeric := numericFromFloat(po.TotalAmount)

	header, err := qtx.CreatePurchaseOrderHeader(ctx, db.CreatePurchaseOrderHeaderParams{
		PoNumber:    po.PoNumber,
		SupplierID:  po.SupplierID,
		PoDate:      pgtype.Date{Time: po.PoDate, Valid: true},
		TaxType:     po.TaxType,
		DppAmount:   dppNumeric,
		TaxAmount:   taxNumeric,
		TotalAmount: totalNumeric,
		Status:      po.Status,
		Notes:       pgtype.Text{String: po.Notes, Valid: po.Notes != ""},
		CreatedBy:   nullableInt8(po.CreatedBy),
		UpdatedBy:   nullableInt8(po.UpdatedBy),
	})
	if err != nil {
		return mapNomorPOBentrok(err)
	}
	po.ID = header.ID
	po.CreatedAt = header.CreatedAt.Time

	// 2. Simpan Detail PO menggunakan Biner SQLC di dalam Loop Transaksi yang sama
	for _, item := range po.Items {
		uPrice := numericFromFloat(item.UnitPrice)
		sTotal := numericFromFloat(item.SubTotal)

		// 🌟 PERBAIKAN TARGET: Ubah menjadi CreatePurchaseOrderDetail agar sesuai biner SQLC baru
		err = qtx.CreatePurchaseOrderDetail(ctx, db.CreatePurchaseOrderDetailParams{
			PurchaseOrderID: header.ID,
			ProductID:       item.ProductID,
			QtyOrdered:      item.QtyOrdered,
			PurchaseUnitID:  item.PurchaseUnitID,
			UnitPrice:       uPrice,
			SubTotal:        sTotal,
		})
		if err != nil {
			return err
		}
	}

	// Komit transaksi native dengan parameter context bawaan pgx
	return tx.Commit(ctx)
}

func (r *sqlPurchaseOrderRepository) GetListCursor(ctx context.Context, param domain.PurchaseOrderParam) ([]domain.PurchaseOrder, error) {
	var rows []db.GetPurchaseOrdersCursorNextRow
	var err error

	if param.Direction == "prev" {
		prevRows, err := r.q.GetPurchaseOrdersCursorPrev(ctx, db.GetPurchaseOrdersCursorPrevParams{
			SearchQuery:  param.Search,
			StatusFilter: param.StatusFilter,
			CursorID:     param.CursorID,
			ReqLimit:     param.Limit,
		})
		if err != nil {
			return nil, err
		}
		for _, v := range prevRows {
			rows = append(rows, db.GetPurchaseOrdersCursorNextRow(v))
		}
		reverseRows(rows)
	} else {
		rows, err = r.q.GetPurchaseOrdersCursorNext(ctx, db.GetPurchaseOrdersCursorNextParams{
			SearchQuery:  param.Search,
			StatusFilter: param.StatusFilter,
			CursorID:     param.CursorID,
			ReqLimit:     param.Limit,
		})
		if err != nil {
			return nil, err
		}
	}

	var results []domain.PurchaseOrder
	for _, v := range rows {
		var totalAmt float64
		if v.TotalAmount.Valid {
			floatVal, _ := v.TotalAmount.Float64Value()
			totalAmt = floatVal.Float64
		}

		// 🌟 EXTRAK DPP DARI LIST ROWS
		var dppAmt float64
		if v.DppAmount.Valid {
			floatVal, _ := v.DppAmount.Float64Value()
			dppAmt = floatVal.Float64
		}

		// 🌟 EXTRAK PPN DARI LIST ROWS
		var taxAmt float64
		if v.TaxAmount.Valid {
			floatVal, _ := v.TaxAmount.Float64Value()
			taxAmt = floatVal.Float64
		}

		results = append(results, domain.PurchaseOrder{
			ID:       v.ID,
			PoNumber: v.PoNumber,
			PoDate:   v.PoDate.Time,

			SupplierID: v.PoSupplierID, // 🌟 KUNCI: Wajib disalurkan agar frontend edit mode bisa mengenali supplier

			// 🌟 PASTIKAN VARIABEL BARU INI DISALURKAN KE OBJECT DOMAIN
			TaxType:   v.TaxType,
			DppAmount: dppAmt,
			TaxAmount: taxAmt,

			TotalAmount:   totalAmt,
			Status:        v.Status,
			Notes:         v.Notes.String,
			ApprovedBy:    convertPgInt64(v.ApprovedBy),
			ApprovedAt:    convertPgTime(v.ApprovedAt),
			RejectedBy:    convertPgInt64(v.RejectedBy),
			RejectedAt:    convertPgTime(v.RejectedAt),
			RejectReason:  v.RejectReason.String,
			SupplierName:  v.SupplierName.String,
			SupplierPic:   v.SupplierPic.String,
			SupplierPhone: v.SupplierPhone.String,
			CreatedAt:     v.CreatedAt.Time,
		})
	}
	return results, nil
}

func (r *sqlPurchaseOrderRepository) CountTotal(ctx context.Context, param domain.PurchaseOrderParam) (int64, error) {
	return r.q.CountPurchaseOrders(ctx, db.CountPurchaseOrdersParams{
		SearchQuery:  param.Search,
		StatusFilter: param.StatusFilter,
	})
}

func (r *sqlPurchaseOrderRepository) GetByID(ctx context.Context, id int64) (*domain.PurchaseOrder, error) {
	row, err := r.q.GetPurchaseOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var dppAmt float64
	if row.DppAmount.Valid {
		dppVal, _ := row.DppAmount.Float64Value()
		dppAmt = dppVal.Float64
	}

	var taxAmt float64
	if row.TaxAmount.Valid {
		taxVal, _ := row.TaxAmount.Float64Value()
		taxAmt = taxVal.Float64
	}

	var totalAmt float64
	if row.TotalAmount.Valid {
		// 🌟 PERBAIKAN MUTLAK: Tangkap nilai error ke-2 menggunakan blank identifier (_)
		floatVal, _ := row.TotalAmount.Float64Value()
		totalAmt = floatVal.Float64
	}

	po := &domain.PurchaseOrder{
		ID:            row.ID,
		PoNumber:      row.PoNumber,
		SupplierID:    row.SupplierID,
		SupplierName:  row.SupplierName.String,
		SupplierPic:   row.SupplierPic.String,
		SupplierPhone: row.SupplierPhone.String,
		SupplierEmail: row.SupplierEmail.String,
		PoDate:        row.PoDate.Time,

		// 🌟 IKAT PROPERTI PAJAK KE LAYER DOMAIN BISNIS
		TaxType:     row.TaxType,
		DppAmount:   dppAmt,
		TaxAmount:   taxAmt,
		TotalAmount: totalAmt,

		Status:       row.Status,
		Notes:        row.Notes.String,
		ApprovedBy:   convertPgInt64(row.ApprovedBy),
		ApprovedAt:   convertPgTime(row.ApprovedAt),
		RejectedBy:   convertPgInt64(row.RejectedBy),
		RejectedAt:   convertPgTime(row.RejectedAt),
		RejectReason: row.RejectReason.String,
		CreatedAt:    row.CreatedAt.Time,
	}

	items, err := r.q.GetPurchaseOrderDetailItems(ctx, id)
	if err != nil {
		return nil, err
	}

	var detailItems []domain.PurchaseOrderDetail
	for _, item := range items {
		var uPrice float64
		if item.UnitPrice.Valid {
			// 🌟 PERBAIKAN MUTLAK: Tangkap nilai error ke-2 menggunakan blank identifier (_)
			floatVal, _ := item.UnitPrice.Float64Value()
			uPrice = floatVal.Float64
		}

		var sTotal float64
		if item.SubTotal.Valid {
			// 🌟 PERBAIKAN MUTLAK: Tangkap nilai error ke-2 menggunakan blank identifier (_)
			floatVal, _ := item.SubTotal.Float64Value()
			sTotal = floatVal.Float64
		}

		detailItems = append(detailItems, domain.PurchaseOrderDetail{
			ID:              item.ID,
			PurchaseOrderID: item.PurchaseOrderID,
			ProductID:       item.ProductID,
			ProductName:     item.ProductName, // ✨ Hilangkan .String jika di db.go tipenya sudah string murni, atau biarkan mengikuti tipe asli hasil generate SQLC
			ProductSku:      item.ProductSku,
			QtyOrdered:      item.QtyOrdered,
			QtyReceived:     item.QtyReceived,
			PurchaseUnitID:  item.PurchaseUnitID,
			UnitKode:        item.UnitKode,
			ConversionRate:  item.ConversionRate,
			SellingUnitID:   item.SellingUnitID,
			SellingUnitKode: item.SellingUnitKode.String,
			UnitPrice:       uPrice,
			SubTotal:        sTotal,
		})
	}

	po.Items = detailItems
	return po, nil
}

// TransitionStatus memindahkan status hanya bila kondisi masih terpenuhi.
//
// allowFrom dan forbidFrom disatukan sebagai syarat di dalam SQL, lalu
// RowsAffected dipakai sebagai sinyal konflik. Inilah penjaga race approve/cancel:
// tanpa ini, dua request paralel bisa sama-sama membaca status lama lalu
// sama-sama menulis status baru, dan dokumen hanya berakhir pada status dari
// penulisan terakhir tanpa ada yang menyadari.
func (r *sqlPurchaseOrderRepository) TransitionStatus(ctx context.Context, id int64, allowFrom, forbidFrom []string, status string, updatedBy int64, reason string) (int64, error) {
	var updatedByPg pgtype.Int8
	if updatedBy > 0 {
		updatedByPg = pgtype.Int8{Int64: updatedBy, Valid: true}
	} else {
		updatedByPg = pgtype.Int8{Valid: false}
	}

	// Kosongkan slice agar pgx tidak mengirim NULL ke klausa = ANY().
	if allowFrom == nil {
		allowFrom = []string{}
	}
	if forbidFrom == nil {
		forbidFrom = []string{}
	}

	var rejectReasonPg pgtype.Text
	if status == "REJECTED" && reason != "" {
		rejectReasonPg = pgtype.Text{String: reason, Valid: true}
	} else if status == "REJECTED" {
		rejectReasonPg = pgtype.Text{String: reason, Valid: reason != ""}
	} else {
		rejectReasonPg = pgtype.Text{Valid: false}
	}
	return r.q.TransitionPurchaseOrderStatus(ctx, db.TransitionPurchaseOrderStatusParams{
		NewStatus:    status,
		UpdatedBy:    updatedByPg,
		ID:           id,
		AllowFrom:    allowFrom,
		ForbidFrom:   forbidFrom,
		RejectReason: rejectReasonPg,
	})
}

// NextPurchaseOrderSeq meminta nomor urut berikutnya ke penghitung di database.
func (r *sqlPurchaseOrderRepository) NextPurchaseOrderSeq(ctx context.Context, prefix string) (int64, error) {
	return r.q.NextPurchaseOrderSeq(ctx, prefix)
}

// mapNomorPOBentrok menerjemahkan pelanggaran unique constraint nomor PO
// menjadi domain.ErrNomorPOBentrok. Lapisan usecase memakai errors.Is terhadap
// sentinel ini untuk mengulang penomoran, jadi pesan PostgreSQL mentah tidak
// boleh diteruskan apa adanya.
func mapNomorPOBentrok(err error) error {
	if pgErr, ok := asUniqueViolation(err); ok {
		return fmt.Errorf("%w: %s", domain.ErrNomorPOBentrok, pgErr.ConstraintName)
	}
	return err
}

// CountReceivedQty melaporkan total kuantitas yang sudah diterima pada satu PO.
// Dipanggil usecase sebagai pengaman sebelum koreksi.
func (r *sqlPurchaseOrderRepository) CountReceivedQty(ctx context.Context, id int64) (int64, error) {
	return r.q.CountPurchaseOrderReceivedQty(ctx, id)
}

// Update mengoreksi header lalu mengganti SELURUH baris detail dalam satu transaksi.
//
// Strategi "delete all + insert all" dipilih karena form edit mengirim daftar item
// lengkap, bukan diff. Aman karena tidak ada tabel lain yang meng referensi
// t_purchase_order_details, dan pemanggil sudah menolak update bila ada baris
// dengan qty_received > 0 sehingga riwayat penerimaan tidak pernah hilang.
func (r *sqlPurchaseOrderRepository) Update(ctx context.Context, po *domain.PurchaseOrder) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	// 1. Koreksi header PO. Kueri memakai RETURNING sehingga baris yang terpengaruh
	// bisa dideteksi: kalau nol baris berarti PO sudah terhapus / tidak ada.
	header, err := qtx.UpdatePurchaseOrderHeader(ctx, db.UpdatePurchaseOrderHeaderParams{
		TaxType:     po.TaxType,
		DppAmount:   numericFromFloat(po.DppAmount),
		TaxAmount:   numericFromFloat(po.TaxAmount),
		TotalAmount: numericFromFloat(po.TotalAmount),
		Notes:       pgtype.Text{String: po.Notes, Valid: po.Notes != ""},
		UpdatedBy:   nullableInt8(po.UpdatedBy),
		ID:          po.ID,
	})
	if err != nil {
		return err
	}

	// 2. Hapus baris detail lama (qty_received dijamin nol oleh usecase).
	if err := qtx.DeletePurchaseOrderDetails(ctx, po.ID); err != nil {
		return err
	}

	// 3. Tulis ulang detail dengan harga & satuan dari master produk.
	for _, item := range po.Items {
		err = qtx.CreatePurchaseOrderDetail(ctx, db.CreatePurchaseOrderDetailParams{
			PurchaseOrderID: header.ID,
			ProductID:       item.ProductID,
			QtyOrdered:      item.QtyOrdered,
			PurchaseUnitID:  item.PurchaseUnitID,
			UnitPrice:       numericFromFloat(item.UnitPrice),
			SubTotal:        numericFromFloat(item.SubTotal),
		})
		if err != nil {
			return err
		}
	}

	po.PoNumber = header.PoNumber
	po.Status = header.Status
	if header.UpdatedAt.Valid {
		po.UpdatedAt = header.UpdatedAt.Time
	}

	return tx.Commit(ctx)
}

// SoftDelete mencabut dokumen PO dengan mengisi deleted_at. Tidak menghapus
// fisik: dokumen hilang dari daftar karena semua query pembacaan memfilter
// deleted_at IS NULL, sementara riwayat keuangan & detail tetap utuh untuk audit.
func (r *sqlPurchaseOrderRepository) SoftDelete(ctx context.Context, id int64, deletedBy int64) error {
	var updatedByPg pgtype.Int8
	if deletedBy > 0 {
		updatedByPg = pgtype.Int8{Int64: deletedBy, Valid: true}
	}

	return r.q.SoftDeletePurchaseOrder(ctx, db.SoftDeletePurchaseOrderParams{
		UpdatedBy: updatedByPg,
		ID:        id,
	})
}
