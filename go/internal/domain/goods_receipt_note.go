package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// DateOnly membaca tanggal dari string "2006-01-02" (nilai mentah <input
// type="date">) maupun RFC3339. Tanpa ini, *time.Time menolak "2026-10-04" dan
// seluruh body dianggap tidak valid.
type DateOnly struct {
	time.Time
}

func (d *DateOnly) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		return nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			d.Time = t
			return nil
		}
	}
	return fmt.Errorf("format tanggal tidak dikenali: %q", s)
}

var (
	// ErrGRNBarisTidakValid dikembalikan saat baris penerimaan tidak valid.
	ErrGRNBarisTidakValid = errors.New("penerimaan ditolak: baris penerimaan tidak valid")
	// ErrGRNPenerimaanKosong dikembalikan saat tidak ada qty yang diterima.
	ErrGRNPenerimaanKosong = errors.New("penerimaan ditolak: tidak ada barang yang diterima")
	// ErrGRNPOSalahStatus dikembalikan saat PO tidak dalam status APPROVED.
	ErrGRNPOSalahStatus = errors.New("penerimaan ditolak: po tidak dalam status APPROVED")
	// ErrGRNOverReceipt dikembalikan saat qty melebihi sisa yang bisa diterima.
	ErrGRNOverReceipt = errors.New("penerimaan ditolak: kuantitas melebihi sisa penerimaan")
	// ErrGRNPONotFound dikembalikan saat PO acuan tidak ada.
	ErrGRNPONotFound = errors.New("penerimaan ditolak: dokumen PO tidak ditemukan atau sudah dihapus")
	// ErrGRNSatuanBermasalah dikembalikan saat satuan baris PO tidak punya
	// faktor konversi yang bisa dipercaya terhadap master produk.
	ErrGRNSatuanBermasalah = errors.New("penerimaan ditolak: satuan barang pada baris PO tidak sesuai dengan satuan produk")
	// ErrGRNBarisPONotFound dikembalikan saat detail PO acuan tidak ada.
	ErrGRNBarisPONotFound = errors.New("penerimaan ditolak: baris detail PO tidak ditemukan")
)

// GoodsReceiptNote merepresentasikan dokumen penerimaan barang (GRN).
type GoodsReceiptNote struct {
	ID              int64                    `json:"id"`
	GRNNumber       string                   `json:"grn_number"`
	PurchaseOrderID int64                    `json:"purchase_order_id"`
	PONumber        string                   `json:"po_number,omitempty"`
	ReceivedDate    time.Time                `json:"received_date"`
	TotalQty        int32                    `json:"total_qty"`
	Notes           string                   `json:"notes,omitempty"`
	CreatedAt       time.Time                `json:"created_at"`
	CreatedBy       *int64                   `json:"created_by,omitempty"`
	Items           []GoodsReceiptNoteDetail `json:"items,omitempty"`
}

// GoodsReceiptNoteDetail merepresentasikan baris detail penerimaan barang.
type GoodsReceiptNoteDetail struct {
	ID                    int64     `json:"id"`
	GRNID                 int64     `json:"grn_id"`
	PurchaseOrderDetailID int64     `json:"purchase_order_detail_id"`
	ProductID             int64     `json:"product_id"`
	SKU                   string    `json:"sku,omitempty"`
	ProductName           string    `json:"product_name,omitempty"`
	PurchaseUnitKode      string    `json:"purchase_unit_kode,omitempty"`
	QtyReceived           int32     `json:"qty_received"`
	ConversionRate        int32     `json:"conversion_rate"`
	CreatedAt             time.Time `json:"created_at"`
}

// CreateGRNItem adalah input untuk satu baris penerimaan.
type CreateGRNItem struct {
	PurchaseOrderDetailID int64 `json:"purchase_order_detail_id"`
	QtyReceived           int32 `json:"qty_received"`
}

// CreateGRNInput adalah input untuk membuat GRN.
type CreateGRNInput struct {
	PurchaseOrderID int64           `json:"purchase_order_id"`
	ReceivedDate    *DateOnly       `json:"received_date,omitempty"`
	Notes           string          `json:"notes,omitempty"`
	Items           []CreateGRNItem `json:"items"`
	CreatedBy       int64           `json:"-"`
}

// GoodsReceiptNoteRepository mendefinisikan kontrak ke penyimpanan GRN.
type GoodsReceiptNoteRepository interface {
	CreateGRN(ctx context.Context, in CreateGRNInput) (*GoodsReceiptNote, error)
	// FinishReceiving menutup penerimaan sebuah PO secara transaksional: mengunci
	// header, menambah stok dari seluruh GRN PO tersebut, lalu mengubah status PO
	// menjadi RECEIVED. Stok sengaja baru ditambahkan di sini, bukan saat GRN
	// dibuat, supaya barang yang diterima belum masuk stok sampai penerimaan
	// ditutup.
	//
	// netCostFactor mengubah harga terima bruto menjadi harga pokok perolehan:
	// 1 bila harga tidak memuat PPN masukan, dan 1/(1+PPN) bila harga sudah
	// termasuk PPN yang bukan biaya perolehan. Nilai ini dipakai saat menghitung
	// ulang harga pokok rata-rata produk; stok dan harga pokok selalu berubah
	// bersama di dalam transaksi ini.
	FinishReceiving(ctx context.Context, poID int64, updatedBy int64, netCostFactor float64) error
	GetListCursor(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]GoodsReceiptNote, int64, error)
	GetByID(ctx context.Context, id int64) (*GoodsReceiptNote, error)
	GetByPOID(ctx context.Context, poID int64) ([]GoodsReceiptNote, error)
}

// GRNJournalPoster menerbitkan jurnal otomatis dari satu GRN (auto-posting).
// Penerapan wajib idempoten: GRN yang sudah punya jurnal POSTED tidak boleh
// menghasilkan jurnal kedua, supaya penutupan penerimaan yang diulang tidak pernah
// mencatat persediaan dua kali.
type GRNJournalPoster interface {
	PostGRNJournal(ctx context.Context, grn *GoodsReceiptNote, po *PurchaseOrder, actorID int64) error
}

// GoodsReceiptNoteUsecase mendefinisikan aturan bisnis GRN.
type GoodsReceiptNoteUsecase interface {
	CreateGRN(ctx context.Context, in CreateGRNInput) (*GoodsReceiptNote, error)
	GetAll(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]GoodsReceiptNote, int64, error)
	GetByID(ctx context.Context, id int64) (*GoodsReceiptNote, error)
	GetByPOID(ctx context.Context, poID int64) ([]GoodsReceiptNote, error)
	FinishReceiving(ctx context.Context, poID int64, allowPartial bool, updatedBy int64) error
}
