package domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrNomorPOBentrok menandai bahwa nomor dokumen yang baru dihitung sudah dipakai
// dokumen lain. Error ini muncul kalau penghitung dan data nyata tidak sinkron,
// misalnya ada nomor yang diisikan manual. Lapisan atas boleh mencoba ulang dengan
// nomor berikutnya alih-alih langsung gagal.
var ErrNomorPOBentrok = errors.New("nomor dokumen sudah dipakai dokumen lain")

// PurchaseOrder mewakili dokumen header pesanan pembelian utama
type PurchaseOrder struct {
	ID            int64                 `json:"id"`
	PoNumber      string                `json:"po_number"`
	SupplierID    int64                 `json:"supplier_id"`
	SupplierName  string                `json:"supplier_name,omitempty"`
	SupplierPic   string                `json:"supplier_pic,omitempty"`
	SupplierPhone string                `json:"supplier_phone,omitempty"`
	SupplierEmail string                `json:"supplier_email,omitempty"`
	PoDate        time.Time             `json:"po_date"`
	TaxType       string                `json:"tax_type"`   // 'NON_PPN', 'TAX_EXCLUSIVE', 'TAX_INCLUSIVE'
	DppAmount     Money                 `json:"dpp_amount"` // Nilai Dasar Pengenaan Pajak
	TaxAmount     Money                 `json:"tax_amount"`
	TotalAmount   Money                 `json:"total_amount"`
	Status        string                `json:"status"` // PENDING, APPROVED, RECEIVED, REJECTED
	Notes         string                `json:"notes,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
	CreatedBy     *int64                `json:"created_by,omitempty"`
	UpdatedBy     *int64                `json:"updated_by,omitempty"`
	ApprovedBy    *int64                `json:"approved_by,omitempty"`
	ApprovedAt    *time.Time            `json:"approved_at,omitempty"`
	RejectedBy    *int64                `json:"rejected_by,omitempty"`
	RejectedAt    *time.Time            `json:"rejected_at,omitempty"`
	RejectReason  string                `json:"reject_reason,omitempty"`
	Items         []PurchaseOrderDetail `json:"items,omitempty"` // Komoditas barang bersarang
}

// PurchaseOrderDetail mewakili baris item barang yang dipesan ke vendor
type PurchaseOrderDetail struct {
	ID              int64  `json:"id"`
	PurchaseOrderID int64  `json:"purchase_order_id"`
	ProductID       int64  `json:"product_id"`
	ProductName     string `json:"product_name,omitempty"`
	ProductSku      string `json:"product_sku,omitempty"`
	QtyOrdered      int32  `json:"qty_ordered"`
	QtyReceived     int32  `json:"qty_received"`
	PurchaseUnitID  int32  `json:"purchase_unit_id"`
	UnitKode        string `json:"unit_kode,omitempty"`
	// ConversionRate memengaruhi berapa satuan jual yang bertambah saat barang
	// diterima, jadi UI perlu menyodorkannya sebelum GRN dibuat.
	ConversionRate  int32  `json:"conversion_rate"`
	SellingUnitID   int32  `json:"selling_unit_id"`
	SellingUnitKode string `json:"selling_unit_kode,omitempty"`
	UnitPrice       Money  `json:"unit_price"`
	SubTotal        Money  `json:"sub_total"`
}

// PurchaseOrderParam penampung filter kueri cursor pagination dari Next.js frontend
type PurchaseOrderParam struct {
	Search       string `json:"search"`
	StatusFilter string `json:"status_filter"`
	CursorID     int64  `json:"cursor_id"`
	Limit        int32  `json:"limit"`
	Direction    string `json:"direction"` // next atau prev
}

// PurchaseOrderRepository menetapkan kontrak interaksi kaku ke driver PostgreSQL
type PurchaseOrderRepository interface {
	Create(ctx context.Context, po *PurchaseOrder) error
	GetListCursor(ctx context.Context, param PurchaseOrderParam) ([]PurchaseOrder, error)
	CountTotal(ctx context.Context, param PurchaseOrderParam) (int64, error)
	GetByID(ctx context.Context, id int64) (*PurchaseOrder, error)
	// TransitionStatus memindahkan status secara atomik dengan syarat status
	// saat itu masih memenuhi allowFrom (dan tidak ada di forbidFrom).
	//
	// Syarat WAJIB dievaluasi di dalam statement SQL yang sama dengan penulisan
	// status baru. Kalau syarat hanya dicek di aplikasi, dua request bersamaan bisa
	// sama-sama membaca status lama lalu sama-sama menuliskannya, dan dokumen
	// berakhir pada status hasil penulisan terakhir tanpa ada yang menyadarikan.
	//
	// Mengembalikan jumlah baris yang benar-benar berubah: 0 berarti ada
	// request lain yang lebih dulu mengubah status, sehingga pemanggil wajib
	// membatalkan aksinya.
	TransitionStatus(ctx context.Context, id int64, allowFrom, forbidFrom []string, status string, updatedBy int64, reason string) (int64, error)
	// NextPurchaseOrderSeq mengembalikan nomor urut berikutnya untuk satu prefiks
	// tanggal, misalnya "20261003".
	//
	// WAJIB diimplementasikan sebagai satu operasi atomik di database. Kalau dihitung
	// di aplikasi (misal dengan COUNT+1), dua permintaan bersamaan akan
	// mendapat nomor yang sama dan salah satunya gagal.
	NextPurchaseOrderSeq(ctx context.Context, prefix string) (int64, error)
	// Update mengoreksi header + mengganti SELURUH baris detail dalam satu transaksi.
	//
	// Pemeriksaan status & qty_received di usecase hanya untuk pesan yang ramah.
	// Proses ini WAJIB mengunci header, lalu membaca ulang kedua syarat itu di
	// dalam transaksi yang sama sebelum menulis. Tanpa itu, approve atau GRN yang
	// datang bersamaan bisa lolos di antara pengecekan usecase dan penulisan.
	Update(ctx context.Context, po *PurchaseOrder) error
	// CountReceivedQty melaporkan total kuantitas yang sudah diterima pada satu PO,
	// dipakai sebagai pengaman agar koreksi tidak menghapus riwayat penerimaan.
	CountReceivedQty(ctx context.Context, id int64) (int64, error)
	// SoftDelete mencabut dokumen dengan mengisi deleted_at, bukan menghapus fisik.
	// Sama seperti Update, proses ini wajib mengunci header dan memverifikasi
	// status PENDING serta qty_received = 0 di dalam transaksi.
	SoftDelete(ctx context.Context, id int64, deletedBy int64) error
}

// PurchaseOrderUsecase menetapkan orkestrasi aturan bisnis (Business Logic)
type PurchaseOrderUsecase interface {
	CreatePO(ctx context.Context, po *PurchaseOrder) error
	FindAllPO(ctx context.Context, param PurchaseOrderParam) ([]PurchaseOrder, int64, error)
	FindPOByID(ctx context.Context, id int64) (*PurchaseOrder, error)
	ApprovePO(ctx context.Context, id int64, userID int64) error
	CancelPO(ctx context.Context, id int64, userID int64) error
	RejectPO(ctx context.Context, id int64, userID int64, reason string) error
	// UpdatePO mengoreksi dokumen PO yang belum melewati approve. Rekanan
	// supplier dikunci (tidak bisa dialihkan) dan harga beli selalu diambil ulang
	// dari master produk, bukan dari payload klien.
	UpdatePO(ctx context.Context, id int64, userID int64, req PurchaseOrderUpdateInput) error
	// DeletePO mencabut dokumen PO yang belum dikunci. Dokumen yang sudah approve
	// atau sudah menerima barang tidak boleh dihapus.
	DeletePO(ctx context.Context, id int64, userID int64) error
}

// PurchaseOrderUpdateInput memuat field yang boleh dikirim klien saat koreksi PO.
// Sengaja TIDAK memuat unit_price / purchase_price: harga dan satuan beli
// diturunkan server dari master produk agar tidak bisa dimanipulasi dari browser.
type PurchaseOrderUpdateInput struct {
	TaxType string
	Notes   string
	Items   []PurchaseOrderItemInput
}

// PurchaseOrderItemInput adalah satu baris barang hasil koreksi dari klien.
type PurchaseOrderItemInput struct {
	ProductID  int64
	QtyOrdered int32
}

// TransitionStatusConflictError menandai bahwa kondisi status berubah di antara
// pengecekan dan penulisan, sehingga aksi tidak jadi dilakukan.
type TransitionStatusConflictError struct {
	From   string
	To     string
	Action string
}

func (e *TransitionStatusConflictError) Error() string {
	from := e.From
	if from == "" {
		from = "tidak diketahui"
	}
	action := e.Action
	if action == "" {
		action = "diubah"
	}
	return fmt.Sprintf(
		"dokumen tidak dapat %s karena statusnya sudah berubah menjadi %q oleh permintaan lain (sasaran: %s), silakan muat ulang dokumen",
		action, from, e.To,
	)
}
