package usecase

import (
	"context"
	"errors"
	"fmt"
	"go_backend/internal/domain"
	"math"
	"time"
)

type purchaseOrderUsecase struct {
	poRepo      domain.PurchaseOrderRepository
	productRepo domain.ProductRepository
}

func NewPurchaseOrderUsecase(po domain.PurchaseOrderRepository, prod domain.ProductRepository) domain.PurchaseOrderUsecase {
	return &purchaseOrderUsecase{poRepo: po, productRepo: prod}
}

// taxRatePPN adalah regulasi PPN aktif yang berlaku untuk dokumen PO.
const taxRatePPN float64 = 0.12

// applyTaxMatrix mengisi DppAmount / TaxAmount / TotalAmount sesuai tipe pajak
// dokumen. Dipakai bersama oleh alur terbit (CreatePO) dan alur koreksi
// (UpdatePO) supaya keduanya tidak pernah menghasilkan perhitungan berbeda.
func applyTaxMatrix(po *domain.PurchaseOrder, totalGrossItems float64) {
	switch po.TaxType {
	case "TAX_EXCLUSIVE":
		// Skenario: Harga belum termasuk PPN (PPN ditambahkan di akhir)
		po.DppAmount = totalGrossItems
		po.TaxAmount = math.Round(po.DppAmount * taxRatePPN)
		po.TotalAmount = po.DppAmount + po.TaxAmount

	case "TAX_INCLUSIVE":
		// Skenario: Harga sudah termasuk PPN (PPN diurai ke dalam)
		po.TotalAmount = totalGrossItems
		po.DppAmount = math.Round(po.TotalAmount / (1.0 + taxRatePPN))
		po.TaxAmount = po.TotalAmount - po.DppAmount

	default:
		// Skenario: NON_PPN (Tanpa pajak pertambahan nilai)
		po.DppAmount = totalGrossItems
		po.TaxAmount = 0
		po.TotalAmount = totalGrossItems
	}
}

// CreatePO mengotomatiskan penerbitan nomor urut dokumen dan kalkulasi finansial kaku

func (u *purchaseOrderUsecase) CreatePO(ctx context.Context, po *domain.PurchaseOrder) error {
	if po.SupplierID <= 0 {
		return errors.New("pesanan ditolak: wajib menentukan rekanan vendor / supplier tujuan")
	}
	if len(po.Items) == 0 {
		return errors.New("pesanan ditolak: tidak ada baris komoditas barang di dalam detail PO")
	}

	// Validasi tipe pajak komersial yang didukung sistem
	if po.TaxType != "NON_PPN" && po.TaxType != "TAX_EXCLUSIVE" && po.TaxType != "TAX_INCLUSIVE" {
		po.TaxType = "NON_PPN"
	}

	// 1. Tanggal dokumen dan VALIDASI HARGA BELI + KALKULASI PAJAK DI SISI SERVER
	//
	// Harga beli selalu diambil ulang dari master produk; nilai apa pun yang
	// dikirim klien diabaikan supaya harga PO tidak bisa dimanipulasi.
	now := time.Now()
	datePrefix := now.Format("20060102")
	po.PoDate = now
	po.Status = "PENDING"

	var totalGrossItems float64 = 0
	for i, item := range po.Items {
		productObj, err := u.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			return fmt.Errorf("barang pada baris ke-%d tidak valid: %v", i+1, err)
		}

		po.Items[i].UnitPrice = productObj.PurchasePrice
		po.Items[i].PurchaseUnitID = productObj.PurchaseUnitID
		po.Items[i].SubTotal = float64(item.QtyOrdered) * productObj.PurchasePrice
		totalGrossItems += po.Items[i].SubTotal
	}

	// 🌟 ALGORITMA CORE: Matriks Perhitungan DPP & PPN 12% Standar ERP
	applyTaxMatrix(po, totalGrossItems)

	po.UpdatedBy = po.CreatedBy

	// 2. TERBITKAN NOMOR DOKUMEN (Format: PO/YYYYMMDD/0001) LALU SIMPAN
	//
	// Nomor diambil dari penghitung di database, bukan dari COUNT+1. Versi
	// COUNT+1 punya tiga cacat:
	//   - dua permintaan bersamaan menghitung angka yang sama lalu bentrok;
	//   - baris soft-deleted tidak ikut dihitung padahal nomornya masih dipakai,
	//     sehingga nomor yang sama diminta lagi dan CREATE gagal;
	//   - penghitungnya memakai query pencarian yang juga mencocokkan nama
	//     supplier, jadi hitungannya bisa salah.
	//
	// Penghitung di database menyelesaikan ketiganya sekaligus: penambahannya
	// atomik, tidak bergantung pada baris PO, dan tidak pernah mundur.
	//
	// Penghitung dan data nyata tetap bisa tidak sinkron, misalnya ada nomor lama
	// yang diisikan manual tanpa lewat penghitung. Karena itu percobaan dibatasi:
	// kalau nomor ternyata sudah dipakai, ambil nomor berikutnya.
	const maksPercobaanNomor = 5
	for attempt := 1; attempt <= maksPercobaanNomor; attempt++ {
		nextSequence, err := u.poRepo.NextPurchaseOrderSeq(ctx, datePrefix)
		if err != nil {
			return err
		}
		po.PoNumber = fmt.Sprintf("PO/%s/%04d", datePrefix, nextSequence)

		err = u.poRepo.Create(ctx, po)
		if err == nil {
			return nil
		}
		if !errors.Is(err, domain.ErrNomorPOBentrok) {
			return err
		}
		if attempt == maksPercobaanNomor {
			return errors.New(
				"pesanan ditolak: nomor dokumen bentrok berulang, mohon hubungi administrator",
			)
		}
	}
	return nil
}

func (u *purchaseOrderUsecase) FindAllPO(ctx context.Context, param domain.PurchaseOrderParam) ([]domain.PurchaseOrder, int64, error) {
	if param.Limit <= 0 {
		param.Limit = 10
	}
	if param.StatusFilter == "" {
		param.StatusFilter = "ALL"
	}

	list, err := u.poRepo.GetListCursor(ctx, param)
	if err != nil {
		return nil, 0, err
	}

	total, err := u.poRepo.CountTotal(ctx, param)
	if err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (u *purchaseOrderUsecase) FindPOByID(ctx context.Context, id int64) (*domain.PurchaseOrder, error) {
	if id <= 0 {
		return nil, errors.New("ID dokumen PO tidak valid")
	}
	return u.poRepo.GetByID(ctx, id)
}

// Status dokumen Purchase Order. Nilai-nilai ini juga dipaksakan oleh CHECK
// constraint di database (migrasi 000003) sehingga tidak ada string asing yang
// bisa masuk lewat jalur lain.
const (
	poStatusPending   = "PENDING"
	poStatusApproved  = "APPROVED"
	poStatusReceived  = "RECEIVED"
	poStatusCancelled = "REJECTED"
)

// nonCancellableStatuses adalah status yang tidak boleh dibatalkan sepihak:
// setelah dokumen disetujui, ia menjadi rujukan penerimaan barang di gudang.
var nonCancellableStatuses = []string{poStatusApproved, poStatusReceived}

func (u *purchaseOrderUsecase) ApprovePO(ctx context.Context, id int64, userID int64) error {
	po, err := u.poRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if po.Status != poStatusPending {
		return fmt.Errorf("dokumen tidak dapat disetujui, status saat ini sudah %s", po.Status)
	}

	// Pengecekan di atas hanya untuk pesan yang ramah. Penegakan sebenarnya ada
	// pada TransitionStatus: syarat status='PENDING' dievaluasi di dalam UPDATE
	// yang sama, sehingga dua permintaan approve yang datang bersamaan tidak
	// bisa dua-duanya berhasil.
	rows, err := u.poRepo.TransitionStatus(ctx, id, []string{poStatusPending}, nil, poStatusApproved, userID, "")
	if err != nil {
		return err
	}
	if rows == 0 {
		return &domain.TransitionStatusConflictError{
			From:   u.statusTerbaruSetelahKonflik(ctx, id, po.Status),
			To:     poStatusApproved,
			Action: "disetujui",
		}
	}
	return nil
}

func (u *purchaseOrderUsecase) CancelPO(ctx context.Context, id int64, userID int64) error {
	po, err := u.poRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if po.Status == poStatusApproved || po.Status == poStatusReceived {
		return fmt.Errorf("dokumen yang sudah berstatus %s tidak bisa dibatalkan sepihak", po.Status)
	}

	// Sama seperti approve: syarat "belum APPROVED dan belum RECEIVED" ikut
	// ditulis di dalam UPDATE. Jika approve menang balapan, baris tidak lagi
	// cocok dan pembatalan ditolak alih-alih menimpa dokumen yang baru disetujui.
	rows, err := u.poRepo.TransitionStatus(ctx, id, nil, nonCancellableStatuses, poStatusCancelled, userID, "")
	if err != nil {
		return err
	}
	if rows == 0 {
		return &domain.TransitionStatusConflictError{
			From:   u.statusTerbaruSetelahKonflik(ctx, id, po.Status),
			To:     poStatusCancelled,
			Action: "dibatalkan",
		}
	}
	return nil
}

// editablePOStatuses adalah status dokumen yang masih aman untuk dikoreksi.
// Setelah approve, dokumen menjadi rujukan penerimaan barang di gudang sehingga
// perubahan header/detail berpotensi merusak konsistensi stok.
var editablePOStatuses = map[string]bool{
	"PENDING": true,
}

// UpdatePO mengoreksi dokumen PO yang masih dapat diedit.
//
// Aturan utama yang ditegakkan di sini:
//  1. Hanya dokumen berstatus PENDING yang boleh dikoreksi.
//  2. Bila sudah ada barang yang diterima (qty_received > 0), detail tidak boleh
//     dihapus karena akan menghapus riwayat penerimaan.
//  3. Rekanan supplier dikunci; kolom supplier di form edit sengaja disabled.
//  4. Harga beli dan satuan beli SELALU diambil ulang dari master produk. Nilai
//     apa pun yang dikirim klien untuk harga diabaikan, sehingga harga PO tidak
//     bisa dimanipulasi dari browser.
func (u *purchaseOrderUsecase) UpdatePO(ctx context.Context, id int64, userID int64, req domain.PurchaseOrderUpdateInput) error {
	if id <= 0 {
		return errors.New("ID dokumen PO tidak valid")
	}
	if len(req.Items) == 0 {
		return errors.New("koreksi ditolak: tidak ada baris komoditas barang di dalam detail PO")
	}

	existing, err := u.poRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !editablePOStatuses[existing.Status] {
		return fmt.Errorf("dokumen berstatus %s sudah terkunci dan tidak dapat dikoreksi lagi", existing.Status)
	}

	received, err := u.poRepo.CountReceivedQty(ctx, id)
	if err != nil {
		return err
	}
	if received > 0 {
		return fmt.Errorf("koreksi ditolak: %d unit barang pada dokumen ini sudah diterima di gudang", received)
	}

	taxType := req.TaxType
	if taxType != "NON_PPN" && taxType != "TAX_EXCLUSIVE" && taxType != "TAX_INCLUSIVE" {
		taxType = "NON_PPN"
	}

	// Susun detail dari master produk. Rekanan supplier diambil dari dokumen
	// yang ada, bukan dari payload, karena tidak boleh dialihkan.
	details := make([]domain.PurchaseOrderDetail, 0, len(req.Items))
	var totalGrossItems float64
	seenProducts := make(map[int64]bool, len(req.Items))

	for i, it := range req.Items {
		if it.ProductID <= 0 {
			return fmt.Errorf("baris ke-%d tidak valid: barang belum dipilih", i+1)
		}
		if it.QtyOrdered <= 0 {
			return fmt.Errorf("baris ke-%d tidak valid: kuantitas harus lebih besar dari nol", i+1)
		}
		if seenProducts[it.ProductID] {
			return fmt.Errorf("baris ke-%d tidak valid: barang yang sama sudah terisi di baris lain", i+1)
		}
		seenProducts[it.ProductID] = true

		productObj, err := u.productRepo.GetByID(ctx, it.ProductID)
		if err != nil {
			return fmt.Errorf("barang pada baris ke-%d tidak valid: %v", i+1, err)
		}

		// Pengaman integritas: barang harus milik rekanan PO. Tanpa ini klien
		// bisa menyuntikkan barang vendor lain beserta harga belinya.
		if productObj.SupplierID != existing.SupplierID {
			return fmt.Errorf("barang pada baris ke-%d bukan milik rekanan supplier dokumen ini", i+1)
		}

		unitPrice := productObj.PurchasePrice
		details = append(details, domain.PurchaseOrderDetail{
			ProductID:      productObj.ID,
			QtyOrdered:     it.QtyOrdered,
			PurchaseUnitID: productObj.PurchaseUnitID,
			UnitPrice:      unitPrice,
			SubTotal:       float64(it.QtyOrdered) * unitPrice,
		})
		totalGrossItems += float64(it.QtyOrdered) * unitPrice
	}

	po := &domain.PurchaseOrder{
		ID:        id,
		TaxType:   taxType,
		Notes:     req.Notes,
		UpdatedBy: &userID,
		Items:     details,
	}
	applyTaxMatrix(po, totalGrossItems)

	return u.poRepo.Update(ctx, po)
}

// DeletePO mencabut dokumen PO dengan soft delete.
//
// Pengaman yang sama seperti UpdatePO berlaku: hanya PENDING yang boleh
// dicabut, dan dokumen yang sudah menerima barang di gudang tidak boleh dihapus
// karena catatan penerimaannya akan ikut hilang dari jejak audit.
func (u *purchaseOrderUsecase) DeletePO(ctx context.Context, id int64, userID int64) error {
	if id <= 0 {
		return errors.New("ID dokumen PO tidak valid")
	}

	existing, err := u.poRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !editablePOStatuses[existing.Status] {
		return fmt.Errorf("dokumen berstatus %s sudah terkunci dan tidak dapat dicabut", existing.Status)
	}

	received, err := u.poRepo.CountReceivedQty(ctx, id)
	if err != nil {
		return err
	}
	if received > 0 {
		return fmt.Errorf("pencabutan ditolak: %d unit barang pada dokumen ini sudah diterima di gudang", received)
	}

	return u.poRepo.SoftDelete(ctx, id, userID)
}

// statusTerbaruSetelahKonflik membaca ulang status supaya pesan error melaporkan
// nilai yang benar.
//
// Nilai yang dibaca sebelum transisi (po.Status) adalah status LAMA, jadi
// memakainya di pesan akan memberi tahu pengguna bahwa statusnya "berubah
// menjadi PENDING" padahal justru itulah nilai sebelumnya. Untuk pesan yang
// menambah instruksi "silakan muat ulang dokumen", pengguna harus diberi nilai
// yang benar.
func (u *purchaseOrderUsecase) statusTerbaruSetelahKonflik(ctx context.Context, id int64, fallback string) string {
	po, err := u.poRepo.GetByID(ctx, id)
	if err != nil || po == nil || po.Status == "" {
		return fallback
	}
	return po.Status
}
func (u *purchaseOrderUsecase) RejectPO(ctx context.Context, id int64, userID int64, reason string) error {
	po, err := u.poRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if po.Status != poStatusPending {
		return fmt.Errorf("dokumen tidak dapat ditolak, status saat ini sudah %s", po.Status)
	}

	rows, err := u.poRepo.TransitionStatus(ctx, id, []string{poStatusPending}, nil, poStatusCancelled, userID, reason)
	if err != nil {
		return err
	}
	if rows == 0 {
		return &domain.TransitionStatusConflictError{
			From:   u.statusTerbaruSetelahKonflik(ctx, id, po.Status),
			To:     poStatusCancelled,
			Action: "ditolak",
		}
	}
	return nil
}
