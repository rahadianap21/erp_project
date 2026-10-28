package postgres

import (
	"context"
	"errors"
	"fmt"
	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	// 🌟 SAMAKAN DENGAN MASTER: Menggunakan pgxpool murni
)

type sqlPurchaseOrderRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewSQLPurchaseOrderRepository menginisialisasi PO berbasis pgxpool
func NewSQLPurchaseOrderRepository(pool *pgxpool.Pool) domain.PurchaseOrderRepository {
	return &sqlPurchaseOrderRepository{pool: pool, q: db.New(pool)}
}

func (r *sqlPurchaseOrderRepository) Create(ctx context.Context, po *domain.PurchaseOrder) error {
	// Membuka transaksi standard native pgxpool
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Injeksi scope transaksi pgx murni ke Queries SQLC
	qtx := r.q.WithTx(tx)

	// 1. Eksekusi Header PO via Kueri SQLC
	dppNumeric := numericFromMoney(po.DppAmount)
	taxNumeric := numericFromMoney(po.TaxAmount)
	totalNumeric := numericFromMoney(po.TotalAmount)

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
		uPrice := numericFromMoney(item.UnitPrice)
		sTotal := numericFromMoney(item.SubTotal)

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
		totalAmt := moneyFromNumeric(v.TotalAmount)
		dppAmt := moneyFromNumeric(v.DppAmount)
		taxAmt := moneyFromNumeric(v.TaxAmount)

		results = append(results, domain.PurchaseOrder{
			ID:       v.ID,
			PoNumber: v.PoNumber,
			PoDate:   v.PoDate.Time,

			SupplierID: v.PoSupplierID, // 🌟 KUNCI: Wajib disalurkan agar frontend edit mode bisa mengenali supplier

			// 🌟 PASTIKAN NILAI BARU INI DISALURKAN KE OBJECT DOMAIN
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

	dppAmt := moneyFromNumeric(row.DppAmount)
	taxAmt := moneyFromNumeric(row.TaxAmount)
	totalAmt := moneyFromNumeric(row.TotalAmount)

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
		uPrice := moneyFromNumeric(item.UnitPrice)
		sTotal := moneyFromNumeric(item.SubTotal)

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
// tanpa ini, dua request bersamaan bisa sama-sama membaca status lama lalu
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
// lengkap, bukan diff. Tabel detail GRN memang mereferensikan baris PO dengan
// ON DELETE RESTRICT, tetapi transaksi ini menolak update bila ada baris dengan
// qty_received > 0, sehingga baris yang sudah punya riwayat penerimaan tidak
// pernah dihapus.
//
// Pemeriksaan status dan qty_received di usecase saja TIDAK cukup: approve atau
// GRN bisa menang balapan setelah usecase membaca PENDING. Karena itu transaksi
// ini mengunci header lebih dulu, lalu MEMBACA ULANG status dan qty_received
// setelah lock didapat. Approve, CreateGRN, dan FinishReceiving juga mengunci
// header yang sama, sehingga urutan yang mungkin hanya:
//   - koreksi selesai lebih dulu, lalu approve; atau
//   - approve/GRN selesai lebih dulu, lalu koreksi ditolak.
func (r *sqlPurchaseOrderRepository) Update(ctx context.Context, po *domain.PurchaseOrder) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)

	// 1. Kunci header dan validasi ulang kondisi yang menjadi syarat koreksi.
	lock, err := qtx.LockPurchaseOrderForUpdate(ctx, po.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.NewNotFoundError("dokumen po tidak ditemukan")
		}
		return err
	}
	if lock.Status != "PENDING" {
		return domain.NewBusinessRuleError("dokumen berstatus %s sudah terkunci dan tidak dapat dikoreksi lagi", lock.Status)
	}
	received, err := qtx.CountPurchaseOrderReceivedQty(ctx, po.ID)
	if err != nil {
		return err
	}
	if received > 0 {
		return domain.NewBusinessRuleError("koreksi ditolak: %d unit barang pada dokumen ini sudah diterima di gudang", received)
	}

	// 2. Koreksi header PO. Kueri memakai RETURNING sehingga baris yang terpengaruh
	// bisa dideteksi: kalau nol baris berarti PO sudah terhapus / tidak ada.
	header, err := qtx.UpdatePurchaseOrderHeader(ctx, db.UpdatePurchaseOrderHeaderParams{
		TaxType:     po.TaxType,
		DppAmount:   numericFromMoney(po.DppAmount),
		TaxAmount:   numericFromMoney(po.TaxAmount),
		TotalAmount: numericFromMoney(po.TotalAmount),
		Notes:       pgtype.Text{String: po.Notes, Valid: po.Notes != ""},
		UpdatedBy:   nullableInt8(po.UpdatedBy),
		ID:          po.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.NewBusinessRuleError("dokumen tidak dapat dikoreksi karena statusnya sudah berubah")
		}
		return err
	}

	// 3. Hapus baris detail lama (qty_received sudah diverifikasi nol di atas).
	if err := qtx.DeletePurchaseOrderDetails(ctx, po.ID); err != nil {
		return err
	}

	// 4. Tulis ulang detail dengan harga & satuan dari master produk.
	for _, item := range po.Items {
		err = qtx.CreatePurchaseOrderDetail(ctx, db.CreatePurchaseOrderDetailParams{
			PurchaseOrderID: header.ID,
			ProductID:       item.ProductID,
			QtyOrdered:      item.QtyOrdered,
			PurchaseUnitID:  item.PurchaseUnitID,
			UnitPrice:       numericFromMoney(item.UnitPrice),
			SubTotal:        numericFromMoney(item.SubTotal),
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
//
// Seperti Update, header dikunci dan kondisi PENDING/qty_received dibaca ulang
// di dalam transaksi. Tanpa ini, approve yang menang balapan setelah pengecekan
// usecase bisa diikuti soft delete, sehingga dokumen APPROVED lenyap dari daftar
// aktif padahal sudah menjadi rujukan penerimaan barang.
func (r *sqlPurchaseOrderRepository) SoftDelete(ctx context.Context, id int64, deletedBy int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)

	lock, err := qtx.LockPurchaseOrderForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.NewNotFoundError("dokumen po tidak ditemukan")
		}
		return err
	}
	if lock.Status != "PENDING" {
		return domain.NewBusinessRuleError("dokumen berstatus %s sudah terkunci dan tidak dapat dicabut", lock.Status)
	}
	received, err := qtx.CountPurchaseOrderReceivedQty(ctx, id)
	if err != nil {
		return err
	}
	if received > 0 {
		return domain.NewBusinessRuleError("pencabutan ditolak: %d unit barang pada dokumen ini sudah diterima di gudang", received)
	}

	var updatedByPg pgtype.Int8
	if deletedBy > 0 {
		updatedByPg = pgtype.Int8{Int64: deletedBy, Valid: true}
	}

	rows, err := qtx.SoftDeletePurchaseOrder(ctx, db.SoftDeletePurchaseOrderParams{
		UpdatedBy: updatedByPg,
		ID:        id,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewBusinessRuleError("dokumen tidak dapat dicabut karena statusnya sudah berubah")
	}

	return tx.Commit(ctx)
}
