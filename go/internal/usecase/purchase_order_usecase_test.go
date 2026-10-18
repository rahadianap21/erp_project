package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"go_backend/internal/domain"
)

// ---------------------------------------------------------------------------
// Fake repository untukPurchaseOrderRepository
// ---------------------------------------------------------------------------

type fakePORepo struct {
	existing        *domain.PurchaseOrder
	existingAfter   *domain.PurchaseOrder
	getByIDCalls    int
	getByIDErr      error
	getByIDErrAfter error
	receivedQty     int64
	receivedErr     error
	updateErr       error

	// penangkap argumen Update
	updateCalls  int
	updatedPO    *domain.PurchaseOrder
	receivedCall int

	// penangkap argumen SoftDelete
	softDeleteCalls int
	softDeleteErr   error
	softDeletedID   int64
	softDeletedBy   int64

	// penangkap argumen & hasil NextPurchaseOrderSeq + Create
	seqCalls              int
	seqFixed              int64
	seqErr                error
	createCalls           int
	createNumbers         []string
	createErr             error
	createBentrokSebanyak []error

	// penangkap argumen & hasil TransitionStatus
	transitionCalls      int
	transitionID         int64
	transitionAllowFrom  []string
	transitionForbidFrom []string
	transitionStatus     string
	transitionRows       int64
	transitionErr        error
}

// NextPurchaseOrderSeq meniru penghitung: mengembalikan angka berikutnya setiap
// dipanggil, atau seqErr kalau sudah batas.
func (f *fakePORepo) NextPurchaseOrderSeq(ctx context.Context, prefix string) (int64, error) {
	f.seqCalls++
	if f.seqErr != nil {
		return 0, f.seqErr
	}
	if f.seqFixed > 0 {
		return f.seqFixed, nil
	}
	return int64(f.seqCalls), nil
}

// Create meniru simpan: bisa disuruh gagal dengan ErrNomorPOBentrok untuk
// percobaan-percobaan pertama, meniru penghitung yang tertinggal di belakang data.
func (f *fakePORepo) Create(ctx context.Context, po *domain.PurchaseOrder) error {
	f.createCalls++
	f.createNumbers = append(f.createNumbers, po.PoNumber)
	if f.createErr != nil {
		return f.createErr
	}
	if len(f.createBentrokSebanyak) > 0 {
		f.createBentrokSebanyak = f.createBentrokSebanyak[1:]
		return fmt.Errorf("%w: t_purchase_orders_po_number_key", domain.ErrNomorPOBentrok)
	}
	return nil
}

func (f *fakePORepo) GetListCursor(ctx context.Context, param domain.PurchaseOrderParam) ([]domain.PurchaseOrder, error) {
	return nil, nil
}

func (f *fakePORepo) CountTotal(ctx context.Context, param domain.PurchaseOrderParam) (int64, error) {
	return 0, nil
}

func (f *fakePORepo) GetByID(ctx context.Context, id int64) (*domain.PurchaseOrder, error) {
	f.getByIDCalls++
	// Panggilan kedua dan seterusnya meniru pembacaan ulang setelah konflik:
	// statusnya sudah berubah karena permintaan lain yang menang balapan.
	if f.getByIDCalls > 1 {
		if f.getByIDErrAfter != nil {
			return nil, f.getByIDErrAfter
		}
		if f.existingAfter != nil {
			return f.existingAfter, nil
		}
	}
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	return f.existing, nil
}

// TransitionStatus meniru compare-and-swap. transitionRows dipakai untuk
// menyimulasikan konflik: 0 berarti permintaan lain menang balapan.
func (f *fakePORepo) TransitionStatus(ctx context.Context, id int64, allowFrom, forbidFrom []string, status string, updatedBy int64, reason string) (int64, error) {
	f.transitionCalls++
	f.transitionID = id
	f.transitionAllowFrom = allowFrom
	f.transitionForbidFrom = forbidFrom
	f.transitionStatus = status
	if f.transitionErr != nil {
		return 0, f.transitionErr
	}
	return f.transitionRows, nil
}

func (f *fakePORepo) Update(ctx context.Context, po *domain.PurchaseOrder) error {
	f.updateCalls++
	f.updatedPO = po
	return f.updateErr
}

func (f *fakePORepo) CountReceivedQty(ctx context.Context, id int64) (int64, error) {
	f.receivedCall++
	return f.receivedQty, f.receivedErr
}

func (f *fakePORepo) SoftDelete(ctx context.Context, id int64, deletedBy int64) error {
	f.softDeleteCalls++
	f.softDeletedID = id
	f.softDeletedBy = deletedBy
	return f.softDeleteErr
}

// ---------------------------------------------------------------------------
// Fake repository untuk ProductRepository
// ---------------------------------------------------------------------------

type fakeProductRepo struct {
	products map[int64]*domain.Product
	getErr   error
}

func (f *fakeProductRepo) Create(ctx context.Context, prod *domain.Product) error { return nil }

func (f *fakeProductRepo) GetProductsCursor(ctx context.Context, search string, categoryFilter int32, cursorID int64, limit int, direction string) ([]domain.Product, int64, error) {
	return nil, 0, nil
}

func (f *fakeProductRepo) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	p, ok := f.products[id]
	if !ok {
		return nil, errors.New("barang tidak ditemukan")
	}
	return p, nil
}

func (f *fakeProductRepo) Update(ctx context.Context, prod *domain.Product) error { return nil }

func (f *fakeProductRepo) Delete(ctx context.Context, id int64, deletedBy int64) error { return nil }

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

// supplierID adalah rekanan dokumen PO pada seluruh skenario uji.
const supplierID int64 = 1

func newUsecase(poStatus string, received int64) (*purchaseOrderUsecase, *fakePORepo, *fakeProductRepo) {
	poRepo := &fakePORepo{
		existing: &domain.PurchaseOrder{
			ID:         77,
			PoNumber:   "PO/20260101/0001",
			SupplierID: supplierID,
			Status:     poStatus,
		},
		receivedQty: received,
		// Default-nya CAS dianggap berhasil; test konflik menetapkannya ke 0.
		transitionRows: 1,
	}
	prodRepo := &fakeProductRepo{products: map[int64]*domain.Product{
		4: {ID: 4, Name: "KURSI KERJA", SupplierID: supplierID, PurchasePrice: 450000, PurchaseUnitID: 7},
		5: {ID: 5, Name: "KERTAS HVS", SupplierID: supplierID, PurchasePrice: 55000, PurchaseUnitID: 8},
		9: {ID: 9, Name: "BARANG VENDOR LAIN", SupplierID: 2, PurchasePrice: 100, PurchaseUnitID: 9},
	}}
	return &purchaseOrderUsecase{poRepo: poRepo, productRepo: prodRepo}, poRepo, prodRepo
}

func item(productID int64, qty int32) domain.PurchaseOrderItemInput {
	return domain.PurchaseOrderItemInput{ProductID: productID, QtyOrdered: qty}
}

// ---------------------------------------------------------------------------
// Uji alur berhasil
// ---------------------------------------------------------------------------

func TestUpdatePO_HappyPath_HargaSelaluDariMaster(t *testing.T) {
	u, poRepo, _ := newUsecase("PENDING", 0)

	err := u.UpdatePO(context.Background(), 77, 9, domain.PurchaseOrderUpdateInput{
		TaxType: "TAX_EXCLUSIVE",
		Notes:   "Kirim box tertutup",
		Items:   []domain.PurchaseOrderItemInput{item(4, 10), item(5, 2)},
	})
	if err != nil {
		t.Fatalf("harus berhasil, dapat error: %v", err)
	}
	if poRepo.updateCalls != 1 {
		t.Fatalf("repo.Update harus dipanggil tepat 1x, dapat %d", poRepo.updateCalls)
	}

	got := poRepo.updatedPO
	if len(got.Items) != 2 {
		t.Fatalf("harus ada 2 baris detail, dapat %d", len(got.Items))
	}

	// Harga harus berasal dari master (450000 dan 55000), bukan dari klien.
	if got.Items[0].UnitPrice != 450000 {
		t.Errorf("harga baris 1 harus 450000 dari master, dapat %v", got.Items[0].UnitPrice)
	}
	if got.Items[0].SubTotal != 10*450000 {
		t.Errorf("subtotal baris 1 harus %v, dapat %v", 10*450000, got.Items[0].SubTotal)
	}
	if got.Items[1].UnitPrice != 55000 {
		t.Errorf("harga baris 2 harus 55000 dari master, dapat %v", got.Items[1].UnitPrice)
	}
	if got.Items[1].SubTotal != 2*55000 {
		t.Errorf("subtotal baris 2 harus %v, dapat %v", 2*55000, got.Items[1].SubTotal)
	}

	// Satuan beli juga wajib dari master.
	if got.Items[0].PurchaseUnitID != 7 || got.Items[1].PurchaseUnitID != 8 {
		t.Errorf("purchase_unit_id harus dari master, dapat %d dan %d",
			got.Items[0].PurchaseUnitID, got.Items[1].PurchaseUnitID)
	}

	// Kalk pajak TAX_EXCLUSIVE: gross 10*450000 + 2*55000 = 4.610.000; PPN 12%
	const gross = 10*450000 + 2*55000
	wantTax := 553200.0 // 4610000 * 0.12
	if got.DppAmount != gross {
		t.Errorf("DPP harus %v, dapat %v", gross, got.DppAmount)
	}
	if got.TaxAmount != wantTax {
		t.Errorf("PPN harus %v, dapat %v", wantTax, got.TaxAmount)
	}
	if got.TotalAmount != gross+wantTax {
		t.Errorf("total harus %v, dapat %v", gross+wantTax, got.TotalAmount)
	}

	if got.Notes != "Kirim box tertutup" {
		t.Errorf("memo harus tersimpan, dapat %q", got.Notes)
	}
	if got.UpdatedBy == nil || *got.UpdatedBy != 9 {
		t.Errorf("updated_by harus 9, dapat %v", got.UpdatedBy)
	}
	if got.ID != 77 {
		t.Errorf("ID harus tetap 77, dapat %d", got.ID)
	}
}

// TestUpdatePO_TidakBisaDialihkanSupplier memastikan kolom supplier tidak pernah
// diubah walau klien mengirim supplier_id lain (nilainya tidak ada di payload).
func TestUpdatePO_RekananTerkunci(t *testing.T) {
	u, _, _ := newUsecase("PENDING", 0)
	err := u.UpdatePO(context.Background(), 77, 9, domain.PurchaseOrderUpdateInput{
		TaxType: "NON_PPN",
		Items:   []domain.PurchaseOrderItemInput{item(4, 1)},
	})
	if err != nil {
		t.Fatalf("harus berhasil, dapat error: %v", err)
	}
	// Update tidak menerima supplier sama sekali; struct yang dikirim ke repo tidak
	// memuat SupplierID sehingga nilainya 0 dan tidak bisa menimpa rekanan lama.
	if u.poRepo.(*fakePORepo).updatedPO.SupplierID != 0 {
		t.Errorf("supplier_id tidak boleh ikut terkirim ke repo update")
	}
}

func TestUpdatePO_MatriksPajak(t *testing.T) {
	tests := []struct {
		taxType  string
		gross    float64
		wantDPP  float64
		wantTax  float64
		wantTot  float64
		wantNote string
	}{
		{"NON_PPN", 4610000, 4610000, 0, 4610000, ""},
		{"TAX_EXCLUSIVE", 4610000, 4610000, 553200, 5163200, ""},
		{"TAX_INCLUSIVE", 4610000, 4116071, 493929, 4610000, ""},
		{"NGARANG", 4610000, 4610000, 0, 4610000, "tipe tak dikenal -> NON_PPN"},
	}
	for _, tc := range tests {
		u, poRepo, _ := newUsecase("PENDING", 0)
		err := u.UpdatePO(context.Background(), 77, 9, domain.PurchaseOrderUpdateInput{
			TaxType: tc.taxType,
			Items:   []domain.PurchaseOrderItemInput{item(4, 10), item(5, 2)},
		})
		if err != nil {
			t.Fatalf("[%s] harus berhasil, dapat error: %v", tc.taxType, err)
		}
		g := poRepo.updatedPO
		if g.DppAmount != tc.wantDPP {
			t.Errorf("[%s] DPP harus %v, dapat %v", tc.taxType, tc.wantDPP, g.DppAmount)
		}
		if g.TaxAmount != tc.wantTax {
			t.Errorf("[%s] PPN harus %v, dapat %v", tc.taxType, tc.wantTax, g.TaxAmount)
		}
		if g.TotalAmount != tc.wantTot {
			t.Errorf("[%s] total harus %v, dapat %v", tc.taxType, tc.wantTot, g.TotalAmount)
		}
	}
}

// ---------------------------------------------------------------------------
// Uji penolakan:TIAP KASUS WAJIB MEMASTIKAN repo.Update TIDAK PERNAH DIPANGGIL
// ---------------------------------------------------------------------------

func TestUpdatePO_Ditolak(t *testing.T) {
	tests := []struct {
		name        string
		poStatus    string
		received    int64
		id          int64
		items       []domain.PurchaseOrderItemInput
		wantErrPart string
	}{
		{
			name:        "dokumen sudah APPROVED",
			poStatus:    "APPROVED",
			id:          77,
			items:       []domain.PurchaseOrderItemInput{item(4, 1)},
			wantErrPart: "sudah terkunci",
		},
		{
			name:        "dokumen sudah RECEIVED",
			poStatus:    "RECEIVED",
			id:          77,
			items:       []domain.PurchaseOrderItemInput{item(4, 1)},
			wantErrPart: "sudah terkunci",
		},
		{
			name:        "dokumen sudah REJECTED",
			poStatus:    "REJECTED",
			id:          77,
			items:       []domain.PurchaseOrderItemInput{item(4, 1)},
			wantErrPart: "sudah terkunci",
		},
		{
			name:        "sudah ada barang diterima di gudang",
			poStatus:    "PENDING",
			received:    12,
			id:          77,
			items:       []domain.PurchaseOrderItemInput{item(4, 1)},
			wantErrPart: "sudah diterima",
		},
		{
			name:        "ID tidak valid",
			poStatus:    "PENDING",
			id:          0,
			items:       []domain.PurchaseOrderItemInput{item(4, 1)},
			wantErrPart: "tidak valid",
		},
		{
			name:        "detail kosong",
			poStatus:    "PENDING",
			id:          77,
			items:       nil,
			wantErrPart: "tidak ada baris",
		},
		{
			name:        "barang belum dipilih",
			poStatus:    "PENDING",
			id:          77,
			items:       []domain.PurchaseOrderItemInput{item(0, 5)},
			wantErrPart: "belum dipilih",
		},
		{
			name:        "kuantitas nol",
			poStatus:    "PENDING",
			id:          77,
			items:       []domain.PurchaseOrderItemInput{item(4, 0)},
			wantErrPart: "lebih besar dari nol",
		},
		{
			name:        "kuantitas negatif",
			poStatus:    "PENDING",
			id:          77,
			items:       []domain.PurchaseOrderItemInput{item(4, -3)},
			wantErrPart: "lebih besar dari nol",
		},
		{
			name:        "barang sama dobel",
			poStatus:    "PENDING",
			id:          77,
			items:       []domain.PurchaseOrderItemInput{item(4, 1), item(4, 2)},
			wantErrPart: "sudah terisi di baris lain",
		},
		{
			name:        "barang milik vendor lain",
			poStatus:    "PENDING",
			id:          77,
			items:       []domain.PurchaseOrderItemInput{item(9, 1)},
			wantErrPart: "bukan milik rekanan supplier",
		},
		{
			name:        "barang tidak ada di master",
			poStatus:    "PENDING",
			id:          77,
			items:       []domain.PurchaseOrderItemInput{item(999, 1)},
			wantErrPart: "tidak valid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, poRepo, _ := newUsecase(tc.poStatus, tc.received)
			err := u.UpdatePO(context.Background(), tc.id, 9, domain.PurchaseOrderUpdateInput{
				TaxType: "NON_PPN",
				Items:   tc.items,
			})

			if err == nil {
				t.Fatalf("harus ditolak, tapi tidak ada error")
			}
			if !strings.Contains(err.Error(), tc.wantErrPart) {
				t.Errorf("pesan error harus memuat %q, dapat %q", tc.wantErrPart, err.Error())
			}
			// Pengaman terpenting: penolakan TIDAK BOLEH menyentuh database.
			if poRepo.updateCalls != 0 {
				t.Errorf("repo.Update tidak boleh dipanggil pada jalur penolakan, dipanggil %d kali", poRepo.updateCalls)
			}
		})
	}
}

// TestUpdatePO_StatusPENDINGBoleh proving PENDING adalah satu-satunya status yang
// boleh dikoreksi.
//
// Test ini sebelumnya memakai DRAFT. Status itu sudah dihapus karena tidak ada
// alur yang pernah menghasilkannya: CreatePO selalu menetapkan PENDING. Jadi
// pembuktian "draft boleh diedit" yang dulu ada di sini sebenarnya tidak pernah
// menyangkut apa pun tentang alur yang benar-benar berjalan.
func TestUpdatePO_StatusPENDINGBoleh(t *testing.T) {
	u, poRepo, _ := newUsecase("PENDING", 0)
	if err := u.UpdatePO(context.Background(), 77, 9, domain.PurchaseOrderUpdateInput{
		TaxType: "NON_PPN",
		Items:   []domain.PurchaseOrderItemInput{item(4, 1)},
	}); err != nil {
		t.Fatalf("status PENDING harus boleh dikoreksi, dapat error: %v", err)
	}
	if poRepo.updateCalls != 1 {
		t.Errorf("repo.Update harus dipanggil 1x, dapat %d", poRepo.updateCalls)
	}
}

// TestUpdatePO_ErrorRepoDiteruskan memastikan kegagalan DB tidak ditelan diam-diam.
func TestUpdatePO_ErrorRepoDiteruskan(t *testing.T) {
	t.Run("GetByID gagal", func(t *testing.T) {
		u, poRepo, _ := newUsecase("PENDING", 0)
		poRepo.getByIDErr = errors.New("koneksi putus")
		err := u.UpdatePO(context.Background(), 77, 9, domain.PurchaseOrderUpdateInput{
			TaxType: "NON_PPN", Items: []domain.PurchaseOrderItemInput{item(4, 1)},
		})
		if err == nil || !strings.Contains(err.Error(), "koneksi putus") {
			t.Fatalf("error repo harus diteruskan, dapat %v", err)
		}
	})

	t.Run("CountReceivedQty gagal", func(t *testing.T) {
		u, poRepo, _ := newUsecase("PENDING", 0)
		poRepo.receivedErr = errors.New("lock timeout")
		err := u.UpdatePO(context.Background(), 77, 9, domain.PurchaseOrderUpdateInput{
			TaxType: "NON_PPN", Items: []domain.PurchaseOrderItemInput{item(4, 1)},
		})
		if err == nil || !strings.Contains(err.Error(), "lock timeout") {
			t.Fatalf("error repo harus diteruskan, dapat %v", err)
		}
		if poRepo.updateCalls != 0 {
			t.Errorf("repo.Update tidak boleh dipanggil, dipanggil %d kali", poRepo.updateCalls)
		}
	})

	t.Run("Update gagal", func(t *testing.T) {
		u, poRepo, _ := newUsecase("PENDING", 0)
		poRepo.updateErr = errors.New("deadlock terdeteksi")
		err := u.UpdatePO(context.Background(), 77, 9, domain.PurchaseOrderUpdateInput{
			TaxType: "NON_PPN", Items: []domain.PurchaseOrderItemInput{item(4, 1)},
		})
		if err == nil || !strings.Contains(err.Error(), "deadlock terdeteksi") {
			t.Fatalf("error repo harus diteruskan, dapat %v", err)
		}
	})
}

// TestCreatePO_Dan_UpdatePO_GunakanMatriksPajakYangSama memastikan kalkulasi
// tidak melenceng antara alur terbit dan alur koreksi.
func TestCreatePO_Dan_UpdatePO_GunakanMatriksPajakYangSama(t *testing.T) {
	po := &domain.PurchaseOrder{TaxType: "TAX_INCLUSIVE"}
	applyTaxMatrix(po, 4610000)
	if po.DppAmount != 4116071 || po.TaxAmount != 493929 || po.TotalAmount != 4610000 {
		t.Errorf("matriks pajak tidak konsisten: dpp=%v tax=%v total=%v",
			po.DppAmount, po.TaxAmount, po.TotalAmount)
	}
}

// ---------------------------------------------------------------------------
// Uji DeletePO (soft delete)
// ---------------------------------------------------------------------------

func TestDeletePO_HappyPath(t *testing.T) {
	u, poRepo, _ := newUsecase("PENDING", 0)
	if err := u.DeletePO(context.Background(), 77, 9); err != nil {
		t.Fatalf("harus berhasil, dapat error: %v", err)
	}
	if poRepo.softDeleteCalls != 1 {
		t.Fatalf("SoftDelete harus dipanggil tepat 1x, dapat %d", poRepo.softDeleteCalls)
	}
	if poRepo.softDeletedID != 77 {
		t.Errorf("ID yang dicabut harus 77, dapat %d", poRepo.softDeletedID)
	}
	if poRepo.softDeletedBy != 9 {
		t.Errorf("deleted_by harus 9, dapat %d", poRepo.softDeletedBy)
	}
}

func TestDeletePO_Ditolak(t *testing.T) {
	tests := []struct {
		name        string
		poStatus    string
		received    int64
		id          int64
		wantErrPart string
	}{
		{"dokumen sudah APPROVED", "APPROVED", 0, 77, "sudah terkunci"},
		{"dokumen sudah RECEIVED", "RECEIVED", 0, 77, "sudah terkunci"},
		{"dokumen sudah REJECTED", "REJECTED", 0, 77, "sudah terkunci"},
		{"sudah ada barang diterima", "PENDING", 5, 77, "sudah diterima"},
		{"ID tidak valid", "PENDING", 0, 0, "tidak valid"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, poRepo, _ := newUsecase(tc.poStatus, tc.received)
			err := u.DeletePO(context.Background(), tc.id, 9)

			if err == nil {
				t.Fatalf("harus ditolak, tapi tidak ada error")
			}
			if !strings.Contains(err.Error(), tc.wantErrPart) {
				t.Errorf("pesan error harus memuat %q, dapat %q", tc.wantErrPart, err.Error())
			}
			// Penolakan tidak boleh menyentuh database.
			if poRepo.softDeleteCalls != 0 {
				t.Errorf("SoftDelete tidak boleh dipanggil pada jalur penolakan, dipanggil %d kali", poRepo.softDeleteCalls)
			}
			// Update juga tidak boleh terpicu.
			if poRepo.updateCalls != 0 {
				t.Errorf("Update tidak boleh dipanggil pada jalur penolakan hapus, dipanggil %d kali", poRepo.updateCalls)
			}
		})
	}
}

// TestDeletePO_StatusPENDINGBoleh proving dokumen yang masih PENDING boleh
// dicabut. Sama seperti di atas, test ini sebelumnya memakai DRAFT.
func TestDeletePO_StatusPENDINGBoleh(t *testing.T) {
	u, poRepo, _ := newUsecase("PENDING", 0)
	if err := u.DeletePO(context.Background(), 77, 9); err != nil {
		t.Fatalf("status PENDING harus boleh dicabut, dapat error: %v", err)
	}
	if poRepo.softDeleteCalls != 1 {
		t.Errorf("SoftDelete harus dipanggil 1x, dapat %d", poRepo.softDeleteCalls)
	}
}

// ---------------------------------------------------------------------------
// Uji compare-and-swap status (penutup celah TOCTOU approve/cancel)
//
// Celah yang ditutup: versi lama melakukan SELECT status lalu UPDATE tanpa
// syarat status. Dua permintaan yang datang bersamaan bisa sama-sama membaca
// status lama, sama-sama lolos pengecekan, lalu sama-sama menulis status baru.
// Akibatnya dokumen hanya berakhir pada status dari penulisan terakhir, dan
// kasus terburuknya approve+cancel saling menimpa tanpa ada yang gagal.
// ---------------------------------------------------------------------------

func TestApprovePO_HappyPath(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)

	if err := u.ApprovePO(context.Background(), 77, 9); err != nil {
		t.Fatalf("harus berhasil, dapat error: %v", err)
	}

	if poRepo.transitionCalls != 1 {
		t.Fatalf("TransitionStatus harus dipanggil tepat 1x, dapat %d", poRepo.transitionCalls)
	}
	if poRepo.transitionID != 77 {
		t.Errorf("ID yang di-transition harus 77, dapat %d", poRepo.transitionID)
	}
	if poRepo.transitionStatus != poStatusApproved {
		t.Errorf("status tujuan harus %s, dapat %s", poStatusApproved, poRepo.transitionStatus)
	}
	// Syarat wajib dibawa ke SQL, bukan hanya dicek di aplikasi.
	if len(poRepo.transitionAllowFrom) != 1 || poRepo.transitionAllowFrom[0] != poStatusPending {
		t.Errorf("allowFrom harus [PENDING], dapat %v", poRepo.transitionAllowFrom)
	}
	if len(poRepo.transitionForbidFrom) != 0 {
		t.Errorf("approve tidak boleh punya forbidFrom, dapat %v", poRepo.transitionForbidFrom)
	}
}

func TestApprovePO_KalahBalapanDenganPermintaanLain(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)
	// GetByID masih melaporkan PENDING (meniru request paralel yang menang
	// selang beberapa milidetik sebelum UPDATE dieksekusi).
	poRepo.transitionRows = 0

	err := u.ApprovePO(context.Background(), 77, 9)
	if err == nil {
		t.Fatal("CAS dengan 0 baris harus gagal, bukan dianggap berhasil")
	}

	var conflict *domain.TransitionStatusConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("error harus TransitionStatusConflictError, dapat %T: %v", err, err)
	}
	if conflict.To != poStatusApproved {
		t.Errorf("sasaran konflik harus %s, dapat %s", poStatusApproved, conflict.To)
	}
	if !strings.Contains(err.Error(), "silakan muat ulang") {
		t.Errorf("pesan harus menyuruh pengguna memuat ulang, dapat: %s", err.Error())
	}
}

func TestApprovePO_StatusLainDitolakSebelumQuery(t *testing.T) {
	for _, status := range []string{poStatusApproved, poStatusReceived, poStatusCancelled} {
		t.Run(status, func(t *testing.T) {
			u, poRepo, _ := newUsecase(status, 0)

			err := u.ApprovePO(context.Background(), 77, 9)
			if err == nil {
				t.Fatalf("status %s tidak boleh disetujui", status)
			}
			if !strings.Contains(err.Error(), "status saat ini sudah "+status) {
				t.Errorf("pesan harus menyebut status saat ini, dapat: %s", err.Error())
			}
			// Pengecekan di atas sudah cukup untuk kasus ini; tidak
			// perlu menyentuh database kalau statusnya memang sudah salah.
			if poRepo.transitionCalls != 0 {
				t.Errorf("TransitionStatus tidak boleh dipanggil, dapat %d", poRepo.transitionCalls)
			}
		})
	}
}

func TestCancelPO_HappyPath(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)

	if err := u.CancelPO(context.Background(), 77, 9); err != nil {
		t.Fatalf("harus berhasil, dapat error: %v", err)
	}

	if poRepo.transitionStatus != poStatusCancelled {
		t.Errorf("status tujuan harus %s, dapat %s", poStatusCancelled, poRepo.transitionStatus)
	}
	// APPROVED dan RECEIVED harus dilarang di dalam SQL supaya pembatalan tidak
	// bisa menimpa dokumen yang baru saja disetujui oleh request paralel.
	if len(poRepo.transitionForbidFrom) != 2 ||
		poRepo.transitionForbidFrom[0] != poStatusApproved ||
		poRepo.transitionForbidFrom[1] != poStatusReceived {
		t.Errorf("forbidFrom harus [APPROVED RECEIVED], dapat %v", poRepo.transitionForbidFrom)
	}
	if len(poRepo.transitionAllowFrom) != 0 {
		t.Errorf("cancel tidak boleh punya allowFrom, dapat %v", poRepo.transitionAllowFrom)
	}
}

func TestCancelPO_KalahBalapanDenganApprove(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)
	// Approve menang balapan dan menulis APPROVED sebelum UPDATE cancel jalan.
	poRepo.transitionRows = 0

	err := u.CancelPO(context.Background(), 77, 9)
	if err == nil {
		t.Fatal("pembatalan yang kalah balapan harus gagal, bukan menimpa dokumen")
	}

	var conflict *domain.TransitionStatusConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("error harus TransitionStatusConflictError, dapat %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "dibatalkan") {
		t.Errorf("pesan harus menyebut aksi pembatalan, dapat: %s", err.Error())
	}
}

func TestCancelPO_StatusTerkunciDitolakSebelumQuery(t *testing.T) {
	for _, status := range []string{poStatusApproved, poStatusReceived} {
		t.Run(status, func(t *testing.T) {
			u, poRepo, _ := newUsecase(status, 0)

			err := u.CancelPO(context.Background(), 77, 9)
			if err == nil {
				t.Fatalf("status %s tidak boleh dibatalkan", status)
			}
			if !strings.Contains(err.Error(), "tidak bisa dibatalkan sepihak") {
				t.Errorf("pesan harus menyebut penguncian, dapat: %s", err.Error())
			}
			if poRepo.transitionCalls != 0 {
				t.Errorf("TransitionStatus tidak boleh dipanggil, dapat %d", poRepo.transitionCalls)
			}
		})
	}
}

// TestApproveDanCancelParalel_HanyaSatuYangMenang memodelkan dua request
// yang benar-benar bersamaan: keduanya membaca PENDING, lalu tepat satu yang
// boleh menulis. Dengan CAS, urutan sebagai apa pun tidak mungkin menghasilkan
// dua penulisan.
func TestApproveDanCancelParalel_HanyaSatuYangMenang(t *testing.T) {
	t.Run("approve menang lebih dulu", func(t *testing.T) {
		uApprove, approveRepo, _ := newUsecase(poStatusPending, 0)
		uCancel, cancelRepo, _ := newUsecase(poStatusPending, 0)
		approveRepo.transitionRows = 1 // approve menang
		cancelRepo.transitionRows = 0  // cancel datang menyusul, tidak cocok lagi

		if err := uApprove.ApprovePO(context.Background(), 77, 9); err != nil {
			t.Fatalf("approve harus berhasil: %v", err)
		}
		if err := uCancel.CancelPO(context.Background(), 77, 9); err == nil {
			t.Fatal("cancel setelah approve harus ditolak")
		}
	})

	t.Run("cancel menang lebih dulu", func(t *testing.T) {
		uApprove, approveRepo, _ := newUsecase(poStatusPending, 0)
		uCancel, cancelRepo, _ := newUsecase(poStatusPending, 0)
		cancelRepo.transitionRows = 1
		approveRepo.transitionRows = 0

		if err := uCancel.CancelPO(context.Background(), 77, 9); err != nil {
			t.Fatalf("cancel harus berhasil: %v", err)
		}
		if err := uApprove.ApprovePO(context.Background(), 77, 9); err == nil {
			t.Fatal("approve setelah cancel harus ditolak")
		}
	})
}

func TestApprovePO_ErrorRepoDiteruskan(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)
	poRepo.transitionErr = errors.New("deadlock terdeteksi")

	err := u.ApprovePO(context.Background(), 77, 9)
	if err == nil {
		t.Fatal("error database harus diteruskan, bukan ditelan")
	}
	var conflict *domain.TransitionStatusConflictError
	if errors.As(err, &conflict) {
		t.Fatalf("error database tidak boleh dilabeli konflik, dapat: %v", err)
	}
	if !strings.Contains(err.Error(), "deadlock terdeteksi") {
		t.Errorf("pesan asli harus diteruskan, dapat: %s", err.Error())
	}
}

// TestKonflikStatusMelaporkanStatusTerbaru menguji akurasi pesan.
//
// Bug yang diuji: pesan konflik dulunya diisi dengan status yang dibaca SEBELUM
// transisi, sehingga user diberi tahu statusnya "berubah menjadi PENDING"
// padahal itu justru nilai sebelumnya. Untuk pesan yang menyuruh user memuat
// ulang dokumen, nilai yang salah lebih buruk daripada tidak disebut.
func TestKonflikStatusMelaporkanStatusTerbaru(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)
	poRepo.transitionRows = 0
	poRepo.existingAfter = &domain.PurchaseOrder{ID: 77, Status: poStatusApproved}

	err := u.ApprovePO(context.Background(), 77, 9)
	if err == nil {
		t.Fatal("harus ada error konflik")
	}

	var conflict *domain.TransitionStatusConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("harus TransitionStatusConflictError, dapat %T", err)
	}
	if conflict.From != poStatusApproved {
		t.Errorf("From harus status terbaru %s, dapat %s", poStatusApproved, conflict.From)
	}
	if strings.Contains(err.Error(), `"PENDING"`) {
		t.Errorf("pesan tidak boleh menyebut status lama sebagai status baru: %s", err.Error())
	}
	if !strings.Contains(err.Error(), `"APPROVED"`) {
		t.Errorf("pesan harus menyebut status terbaru: %s", err.Error())
	}
}

// TestKonflikStatus_PembacaanUlangGagalTetapAman memastikan kegagalan membaca
// ulang tidak membuat alur gagal total; pesan tetap terbit dengan nilai lama
// sebagai cadangan.
func TestKonflikStatus_PembacaanUlangGagalTetapAman(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)
	poRepo.transitionRows = 0
	poRepo.getByIDErrAfter = errors.New("koneksi putus saat membaca ulang")

	err := u.ApprovePO(context.Background(), 77, 9)
	if err == nil {
		t.Fatal("harus ada error konflik")
	}
	var conflict *domain.TransitionStatusConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("harus tetap TransitionStatusConflictError, dapat %T: %v", err, err)
	}
	if conflict.From != poStatusPending {
		t.Errorf("nilai cadangan harus status lama %s, dapat %s", poStatusPending, conflict.From)
	}
}

// ---------------------------------------------------------------------------
// Uji penerbitan nomor dokumen
//
// Nomor PO dulu dihitung dengan COUNT+1 terhadap baris yang masih aktif. Tiga
// cacat muncul dari sana, dan ketiganya ditutup oleh penghitung di database:
//   1. dua permintaan bersamaan mendapat nomor yang sama lalu CREATE gagal;
//   2. baris soft-deleted tidak ikut dihitung, sehingga nomor yang sudah pernah
//      terbit diminta lagi dan CREATE gagal;
//   3. query penghitungnya sekaligus mencocokkan nama supplier.
// ---------------------------------------------------------------------------

func TestCreatePO_NomorDiambilDariPenghitung(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)

	po := &domain.PurchaseOrder{
		SupplierID: 1,
		TaxType:    "NON_PPN",
		Items:      []domain.PurchaseOrderDetail{{ProductID: 4, QtyOrdered: 1}},
	}
	if err := u.CreatePO(context.Background(), po); err != nil {
		t.Fatalf("harus berhasil, dapat error: %v", err)
	}

	if poRepo.seqCalls != 1 {
		t.Fatalf("penghitung harus dipanggil 1x, dapat %d", poRepo.seqCalls)
	}
	prefiks := time.Now().Format("20060102")
	want := fmt.Sprintf("PO/%s/0001", prefiks)
	if po.PoNumber != want {
		t.Fatalf("nomor harus %s, dapat %s", want, po.PoNumber)
	}
	if po.Status != "PENDING" || po.PoDate.IsZero() {
		t.Errorf("status/tanggal dokumen harus terisi, dapat %s / %v", po.Status, po.PoDate)
	}
}

func TestCreatePO_HargaSelaluDariMaster(t *testing.T) {
	u, _, _ := newUsecase(poStatusPending, 0)

	po := &domain.PurchaseOrder{
		SupplierID: 1,
		TaxType:    "NON_PPN",
		// Qty 2, tapi harga dan satuan sengaja diisi ngawur: harus diabaikan.
		Items: []domain.PurchaseOrderDetail{
			{ProductID: 4, QtyOrdered: 2, UnitPrice: 1, PurchaseUnitID: 999},
		},
	}
	if err := u.CreatePO(context.Background(), po); err != nil {
		t.Fatalf("harus berhasil, dapat error: %v", err)
	}

	if po.Items[0].UnitPrice != 450000 {
		t.Errorf("harga harus dari master (450000), dapat %v", po.Items[0].UnitPrice)
	}
	if po.Items[0].PurchaseUnitID != 7 {
		t.Errorf("satuan harus dari master (7), dapat %d", po.Items[0].PurchaseUnitID)
	}
	if po.Items[0].SubTotal != 900000 {
		t.Errorf("subtotal harus 2x450000, dapat %v", po.Items[0].SubTotal)
	}
}

func TestCreatePO_MencobaLagiSaatNomorBentrok(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)
	// Dua percobaan pertama ditolak unique constraint, meniru penghitung yang
	// tertinggal di belakang data nyata. Percobaan ketiga harus berhasil.
	poRepo.createBentrokSebanyak = []error{
		fmt.Errorf("%w: x", domain.ErrNomorPOBentrok),
		fmt.Errorf("%w: x", domain.ErrNomorPOBentrok),
	}

	po := &domain.PurchaseOrder{
		SupplierID: 1,
		TaxType:    "NON_PPN",
		Items:      []domain.PurchaseOrderDetail{{ProductID: 4, QtyOrdered: 1}},
	}
	if err := u.CreatePO(context.Background(), po); err != nil {
		t.Fatalf("percobaan ulang harus berhasil, dapat error: %v", err)
	}

	if poRepo.createCalls != 3 {
		t.Fatalf("harus ada 3 percobaan simpan, dapat %d", poRepo.createCalls)
	}
	// Setiap percobaan harus memakai nomor yang berbeda.
	want := []string{"0001", "0002", "0003"}
	for i, got := range poRepo.createNumbers {
		suffix := got[len(got)-4:]
		if suffix != want[i] {
			t.Errorf("percobaan %d harus memakai nomor %s, dapat %s", i+1, want[i], suffix)
		}
	}
}

func TestCreatePO_BerlimaPercobaanGagalTetapBersih(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)
	// Selalu bentrok: percobaan harus dihentikan dan pesannya tidak boleh
	// membocorkan pesan SQL mentah.
	for i := 0; i < 10; i++ {
		poRepo.createBentrokSebanyak = append(poRepo.createBentrokSebanyak, nil)
	}

	po := &domain.PurchaseOrder{
		SupplierID: 1,
		TaxType:    "NON_PPN",
		Items:      []domain.PurchaseOrderDetail{{ProductID: 4, QtyOrdered: 1}},
	}
	err := u.CreatePO(context.Background(), po)
	if err == nil {
		t.Fatal("harus gagal setelah batas percobaan habis")
	}
	if poRepo.createCalls != 5 {
		t.Fatalf("percobaan harus dibatasi 5x, dapat %d", poRepo.createCalls)
	}
	if !strings.Contains(err.Error(), "ditolak") {
		t.Errorf("pesan harus berupa penolakan bisnis yang rapi, dapat: %s", err.Error())
	}
	if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "SQLSTATE") {
		t.Errorf("pesan tidak boleh membocorkan detail SQL: %s", err.Error())
	}
}

func TestCreatePO_ErrorLainTidakDipercobaUlang(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)
	poRepo.createErr = errors.New("koneksi database terputus")

	po := &domain.PurchaseOrder{
		SupplierID: 1,
		TaxType:    "NON_PPN",
		Items:      []domain.PurchaseOrderDetail{{ProductID: 4, QtyOrdered: 1}},
	}
	err := u.CreatePO(context.Background(), po)
	if err == nil {
		t.Fatal("error database harus diteruskan")
	}
	if poRepo.createCalls != 1 {
		t.Fatalf("error non-bentrok tidak boleh diulang, percobaan=%d", poRepo.createCalls)
	}
	if !strings.Contains(err.Error(), "koneksi database terputus") {
		t.Errorf("pesan asli harus diteruskan, dapat: %s", err.Error())
	}
}

func TestCreatePO_ErrorPenghitungDiteruskan(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)
	poRepo.seqErr = errors.New("penghitung tidak merespon")

	po := &domain.PurchaseOrder{
		SupplierID: 1,
		TaxType:    "NON_PPN",
		Items:      []domain.PurchaseOrderDetail{{ProductID: 4, QtyOrdered: 1}},
	}
	err := u.CreatePO(context.Background(), po)
	if err == nil || !strings.Contains(err.Error(), "penghitung tidak merespon") {
		t.Fatalf("error penghitung harus diteruskan, dapat: %v", err)
	}
	if poRepo.createCalls != 0 {
		t.Errorf("tidak boleh mencoba simpan tanpa nomor, percobaan=%d", poRepo.createCalls)
	}
}

func TestCreatePO_ValidasiTetapBerjalanSebelumNomorDiterbitkan(t *testing.T) {
	u, poRepo, _ := newUsecase(poStatusPending, 0)

	// Supplier tidak sah
	po := &domain.PurchaseOrder{Items: []domain.PurchaseOrderDetail{{ProductID: 4, QtyOrdered: 1}}}
	if err := u.CreatePO(context.Background(), po); err == nil {
		t.Error("supplier kosong harus ditolak")
	}
	// Barang tidak ada
	po = &domain.PurchaseOrder{SupplierID: 1}
	if err := u.CreatePO(context.Background(), po); err == nil {
		t.Error("PO tanpa barang harus ditolak")
	}

	if poRepo.seqCalls != 0 {
		t.Errorf("nomor tidak boleh diterbitkan untuk PO yang sudah ditolak, seqCalls=%d", poRepo.seqCalls)
	}
	if poRepo.createCalls != 0 {
		t.Errorf("tidak boleh ada percobaan simpan, dapat %d", poRepo.createCalls)
	}
}
