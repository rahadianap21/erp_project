package usecase

import (
	"context"
	"errors"
	"math"
	"testing"

	"go_backend/internal/domain"
)

// fakeGRNRepo mencatat argumen CreateGRN dan mengembalikan hasil yangysetel.
type fakeGRNRepo struct {
	createdInput domain.CreateGRNInput
	createCalls  int
	createResult *domain.GoodsReceiptNote
	createErr    error

	byPO      []domain.GoodsReceiptNote
	byPOErr   error
	byPOCalls int

	byID map[int64]*domain.GoodsReceiptNote

	finishCalls         int
	finishPOID          int64
	finishUserID        int64
	finishNetCostFactor float64
	finishErr           error
}

func (f *fakeGRNRepo) FinishReceiving(ctx context.Context, poID int64, updatedBy int64, netCostFactor float64) error {
	f.finishCalls++
	f.finishPOID = poID
	f.finishUserID = updatedBy
	f.finishNetCostFactor = netCostFactor
	return f.finishErr
}

func (f *fakeGRNRepo) CreateGRN(ctx context.Context, in domain.CreateGRNInput) (*domain.GoodsReceiptNote, error) {
	f.createCalls++
	f.createdInput = in
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResult != nil {
		return f.createResult, nil
	}
	return &domain.GoodsReceiptNote{ID: 1, PurchaseOrderID: in.PurchaseOrderID, TotalQty: 1}, nil
}

func (f *fakeGRNRepo) GetListCursor(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]domain.GoodsReceiptNote, int64, error) {
	return nil, 0, nil
}

func (f *fakeGRNRepo) GetByID(ctx context.Context, id int64) (*domain.GoodsReceiptNote, error) {
	if f.byID != nil {
		return f.byID[id], nil
	}
	return nil, nil
}

func (f *fakeGRNRepo) GetByPOID(ctx context.Context, poID int64) ([]domain.GoodsReceiptNote, error) {
	f.byPOCalls++
	if f.byPOErr != nil {
		return nil, f.byPOErr
	}
	return f.byPO, nil
}

// fakeReceivePORepo hanya venerable untuk jalur FinishReceiving.
type fakeReceivePORepo struct {
	po             *domain.PurchaseOrder
	getErr         error
	transition     int
	transitionRows int64
	transitionErr  error
	lastAllowFrom  []string
	lastForbidFrom []string
	lastStatus     string
	lastUpdatedBy  int64
}

func (f *fakeReceivePORepo) Create(ctx context.Context, po *domain.PurchaseOrder) error { return nil }
func (f *fakeReceivePORepo) GetListCursor(ctx context.Context, param domain.PurchaseOrderParam) ([]domain.PurchaseOrder, error) {
	return nil, nil
}
func (f *fakeReceivePORepo) CountTotal(ctx context.Context, param domain.PurchaseOrderParam) (int64, error) {
	return 0, nil
}
func (f *fakeReceivePORepo) GetByID(ctx context.Context, id int64) (*domain.PurchaseOrder, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.po, nil
}
func (f *fakeReceivePORepo) TransitionStatus(ctx context.Context, id int64, allowFrom, forbidFrom []string, status string, updatedBy int64, reason string) (int64, error) {
	f.transition++
	f.lastAllowFrom = allowFrom
	f.lastForbidFrom = forbidFrom
	f.lastStatus = status
	f.lastUpdatedBy = updatedBy
	if f.transitionErr != nil {
		return 0, f.transitionErr
	}
	return f.transitionRows, nil
}
func (f *fakeReceivePORepo) NextPurchaseOrderSeq(ctx context.Context, prefix string) (int64, error) {
	return 1, nil
}
func (f *fakeReceivePORepo) Update(ctx context.Context, po *domain.PurchaseOrder) error { return nil }
func (f *fakeReceivePORepo) CountReceivedQty(ctx context.Context, id int64) (int64, error) {
	return 0, nil
}
func (f *fakeReceivePORepo) SoftDelete(ctx context.Context, id int64, deletedBy int64) error {
	return nil
}
func (f *fakeReceivePORepo) CreatePO(ctx context.Context, po *domain.PurchaseOrder) error { return nil }

func approvedPO(items ...domain.PurchaseOrderDetail) *domain.PurchaseOrder {
	return &domain.PurchaseOrder{
		ID:     7,
		Status: "APPROVED",
		Items:  items,
	}
}

func detail(id int64, ordered, received int32) domain.PurchaseOrderDetail {
	return domain.PurchaseOrderDetail{ID: id, QtyOrdered: ordered, QtyReceived: received}
}

// TestFinishReceivingTanGRNMustahir memastikan PO tidak bisa ditutup RECEIVED
// tanpa satu pun GRN. Tanpa guard ini, PO yang tidak pernah menerima barang
// bisa ditandai selesai hanya dengan satu klik.
func TestFinishReceivingTanGRNMustahir(t *testing.T) {
	grnRepo := &fakeGRNRepo{byPO: nil}
	poRepo := &fakeReceivePORepo{po: approvedPO(detail(1, 10, 10))}
	uc := NewGoodsReceiptNoteUsecase(grnRepo, poRepo, nil)

	err := uc.FinishReceiving(context.Background(), 7, true, 99)
	if !errors.Is(err, domain.ErrGRNPenerimaanKosong) {
		t.Fatalf("harus menolak tanpa GRN, dapat %v", err)
	}
	if grnRepo.finishCalls != 0 {
		t.Fatalf("finalisasi tidak boleh dipanggil, dapat %d kali", grnRepo.finishCalls)
	}
}

// TestFinishReceivingPenuhMenutupPO memastikan PO yang sudah penuh ditutup
// tanpa perlu allow_partial.
func TestFinishReceivingPenuhMenutupPO(t *testing.T) {
	grnRepo := &fakeGRNRepo{byPO: []domain.GoodsReceiptNote{{ID: 1, GRNNumber: "GRN/20261004/0001"}}}
	poRepo := &fakeReceivePORepo{po: approvedPO(detail(1, 10, 10), detail(2, 5, 5)), transitionRows: 1}
	uc := NewGoodsReceiptNoteUsecase(grnRepo, poRepo, nil)

	if err := uc.FinishReceiving(context.Background(), 7, false, 99); err != nil {
		t.Fatalf("po penuh harus bisa ditutup: %v", err)
	}
	if grnRepo.finishCalls != 1 {
		t.Fatalf("finalisasi harus dipanggil sekali, dapat %d", grnRepo.finishCalls)
	}
	if grnRepo.finishPOID != 7 {
		t.Fatalf("poID harus 7, dapat %d", grnRepo.finishPOID)
	}
	if grnRepo.finishUserID != 99 {
		t.Fatalf("updatedBy harus diteruskan, dapat %d", grnRepo.finishUserID)
	}
}

// TestFinishReceivingBelumPenuhTanAllowPartial ditolak karena masih ada sisa.
func TestFinishReceivingBelumPenuhTanAllowPartial(t *testing.T) {
	grnRepo := &fakeGRNRepo{byPO: []domain.GoodsReceiptNote{{ID: 1}}}
	poRepo := &fakeReceivePORepo{po: approvedPO(detail(1, 10, 4))}
	uc := NewGoodsReceiptNoteUsecase(grnRepo, poRepo, nil)

	err := uc.FinishReceiving(context.Background(), 7, false, 99)
	if err == nil {
		t.Fatal("short receipt tanpa allow_partial harus ditolak")
	}
	if grnRepo.finishCalls != 0 {
		t.Fatalf("status tidak boleh berubah, finalisasi %d kali", grnRepo.finishCalls)
	}
}

// TestFinishReceivingShortDenganAllowPartial ditutup meski masih ada sisa.
func TestFinishReceivingShortDenganAllowPartial(t *testing.T) {
	grnRepo := &fakeGRNRepo{byPO: []domain.GoodsReceiptNote{{ID: 1}}}
	poRepo := &fakeReceivePORepo{po: approvedPO(detail(1, 10, 4)), transitionRows: 1}
	uc := NewGoodsReceiptNoteUsecase(grnRepo, poRepo, nil)

	if err := uc.FinishReceiving(context.Background(), 7, true, 99); err != nil {
		t.Fatalf("short receipt dengan allow_partial harus ditutup: %v", err)
	}
	if grnRepo.finishCalls != 1 {
		t.Fatalf("finalisasi harus dipanggil sekali, dapat %d", grnRepo.finishCalls)
	}
}

// TestFinishReceivingStatusSalahDitolak menjaga PO non-APPROVED tidak bisa ditutup.
func TestFinishReceivingStatusSalahDitolak(t *testing.T) {
	grnRepo := &fakeGRNRepo{}
	poRepo := &fakeReceivePORepo{po: &domain.PurchaseOrder{ID: 7, Status: "RECEIVED"}}
	uc := NewGoodsReceiptNoteUsecase(grnRepo, poRepo, nil)

	err := uc.FinishReceiving(context.Background(), 7, true, 99)
	if err == nil {
		t.Fatal("PO RECEIVED tidak boleh ditutup lagi")
	}
	if grnRepo.finishCalls != 0 {
		t.Fatalf("tidak boleh ada finalisasi, dapat %d", grnRepo.finishCalls)
	}
}

// TestFinishReceivingGagalAmbilGRNMembatalkan memastikan error baca GRN tidak
// ditelan lalu diproses sebagai PO kosong.
func TestFinishReceivingGagalAmbilGRNMembatalkan(t *testing.T) {
	grnRepo := &fakeGRNRepo{byPOErr: errors.New("db down")}
	poRepo := &fakeReceivePORepo{po: approvedPO(detail(1, 10, 10))}
	uc := NewGoodsReceiptNoteUsecase(grnRepo, poRepo, nil)

	err := uc.FinishReceiving(context.Background(), 7, true, 99)
	if err == nil {
		t.Fatal("error repository harus diteruskan")
	}
	if grnRepo.finishCalls != 0 {
		t.Fatalf("tidak boleh ada finalisasi saat GRN gagal dibaca, dapat %d", grnRepo.finishCalls)
	}
}

// TestCreateGRNMenyerahkanInput memastikan use case tidak mengubah isi input
// sebelum repository yang LocksPO dan memvalidasi over-receipt.
func TestCreateGRNMenyerahkanInput(t *testing.T) {
	in := domain.CreateGRNInput{
		PurchaseOrderID: 12,
		CreatedBy:       5,
		Notes:           "catatan",
		Items:           []domain.CreateGRNItem{{PurchaseOrderDetailID: 3, QtyReceived: 4}},
	}
	grnRepo := &fakeGRNRepo{}
	poRepo := &fakeReceivePORepo{}
	uc := NewGoodsReceiptNoteUsecase(grnRepo, poRepo, nil)

	if _, err := uc.CreateGRN(context.Background(), in); err != nil {
		t.Fatalf("create harus diteruskan: %v", err)
	}
	if grnRepo.createCalls != 1 {
		t.Fatalf("repository harus dipanggil sekali, dapat %d", grnRepo.createCalls)
	}
	if grnRepo.createdInput.PurchaseOrderID != 12 {
		t.Fatalf("purchase_order_id tidak tersalin, dapat %d", grnRepo.createdInput.PurchaseOrderID)
	}
	if len(grnRepo.createdInput.Items) != 1 || grnRepo.createdInput.Items[0].QtyReceived != 4 {
		t.Fatalf("baris barang tidak tersalin: %+v", grnRepo.createdInput.Items)
	}
}

// TestCreateGRNErrorRepositoryDiteruskan memastikan error transaksi tidak ditimpa.
func TestCreateGRNErrorRepositoryDiteruskan(t *testing.T) {
	want := domain.ErrGRNOverReceipt
	grnRepo := &fakeGRNRepo{createErr: want}
	uc := NewGoodsReceiptNoteUsecase(grnRepo, &fakeReceivePORepo{}, nil)

	_, err := uc.CreateGRN(context.Background(), domain.CreateGRNInput{
		PurchaseOrderID: 1,
		Items:           []domain.CreateGRNItem{{PurchaseOrderDetailID: 1, QtyReceived: 1}},
	})
	if !errors.Is(err, want) {
		t.Fatalf("harus meneruskan %v, dapat %v", want, err)
	}
}

// TestCreateGRNKosongDitolak menjaga GRN tanpa barang tidak pernah sampai ke database.
func TestCreateGRNKosongDitolak(t *testing.T) {
	grnRepo := &fakeGRNRepo{}
	uc := NewGoodsReceiptNoteUsecase(grnRepo, &fakeReceivePORepo{}, nil)

	_, err := uc.CreateGRN(context.Background(), domain.CreateGRNInput{PurchaseOrderID: 1})
	if !errors.Is(err, domain.ErrGRNPenerimaanKosong) {
		t.Fatalf("harus menolak GRN kosong, dapat %v", err)
	}
	if grnRepo.createCalls != 0 {
		t.Fatalf("repository tidak boleh dipanggil, dapat %d", grnRepo.createCalls)
	}
}

// TestGetByPOIDValidasiID menolak id tidak valid sebelum menyentuh database.
func TestGetByPOIDValidasiID(t *testing.T) {
	grnRepo := &fakeGRNRepo{}
	uc := NewGoodsReceiptNoteUsecase(grnRepo, &fakeReceivePORepo{}, nil)

	if _, err := uc.GetByPOID(context.Background(), 0); err == nil {
		t.Fatal("id 0 harus ditolak")
	}
	if grnRepo.byPOCalls != 0 {
		t.Fatalf("repository tidak boleh dipanggil untuk id invalid, dapat %d", grnRepo.byPOCalls)
	}
}

// fakeGRNPoster mencatat GRN yang dimintakan jurnal otomatis.
type fakeGRNPoster struct {
	calls []int64
	err   error
}

func (f *fakeGRNPoster) PostGRNJournal(ctx context.Context, grn *domain.GoodsReceiptNote, po *domain.PurchaseOrder, actorID int64) error {
	f.calls = append(f.calls, grn.ID)
	return f.err
}

// TestFinishReceivingMenerbitkanJurnal memastikan penutupan penerimaan menerbitkan
// jurnal otomatis per GRN.
func TestFinishReceivingMenerbitkanJurnal(t *testing.T) {
	grn := domain.GoodsReceiptNote{ID: 1, GRNNumber: "GRN/20261014/0001"}
	grnRepo := &fakeGRNRepo{
		byPO: []domain.GoodsReceiptNote{grn},
		byID: map[int64]*domain.GoodsReceiptNote{1: &grn},
	}
	poRepo := &fakeReceivePORepo{po: approvedPO(detail(1, 10, 10)), transitionRows: 1}
	poster := &fakeGRNPoster{}
	uc := NewGoodsReceiptNoteUsecase(grnRepo, poRepo, poster)

	if err := uc.FinishReceiving(context.Background(), 7, false, 99); err != nil {
		t.Fatalf("penutupan harus berhasil: %v", err)
	}
	if len(poster.calls) != 1 || poster.calls[0] != 1 {
		t.Fatalf("jurnal harus diterbitkan untuk GRN 1, dapat %v", poster.calls)
	}
	if grnRepo.finishCalls != 1 {
		t.Fatalf("finalisasi stok harus tetap berjalan, dapat %d", grnRepo.finishCalls)
	}
}

// TestFinishReceivingBatalBilaJurnalGagal menjaga agar stok dan status PO TIDAK
// berubah ketika jurnal otomatis gagal: pengguna mengulang dan pada akhirnya
// penerbitan per GRN yang idempoten menyelaraskan buku besar dengan stok.
func TestFinishReceivingBatalBilaJurnalGagal(t *testing.T) {
	grn := domain.GoodsReceiptNote{ID: 1, GRNNumber: "GRN/20261014/0001"}
	grnRepo := &fakeGRNRepo{
		byPO: []domain.GoodsReceiptNote{grn},
		byID: map[int64]*domain.GoodsReceiptNote{1: &grn},
	}
	poRepo := &fakeReceivePORepo{po: approvedPO(detail(1, 10, 10)), transitionRows: 1}
	poster := &fakeGRNPoster{err: errors.New("periode fiskal ditutup")}
	uc := NewGoodsReceiptNoteUsecase(grnRepo, poRepo, poster)

	if err := uc.FinishReceiving(context.Background(), 7, false, 99); err == nil {
		t.Fatal("kegagalan jurnal harus diteruskan")
	}
	if grnRepo.finishCalls != 0 {
		t.Fatalf("stok tidak boleh berubah saat jurnal gagal, finalisasi %d kali", grnRepo.finishCalls)
	}
}

// TestNetCostFactorFor memastikan hanya TAX_INCLUSIVE yang membagi harga terima
// dengan (1+PPN), karena hanya pada tipe itu harga memuat PPN masukan yang harus
// dikeluarkan dari nilai persediaan.
func TestNetCostFactorFor(t *testing.T) {
	cases := []struct {
		taxType string
		want    float64
	}{
		{"NON_PPN", 1.0},
		{"TAX_EXCLUSIVE", 1.0},
		{"TAX_INCLUSIVE", 1.0 / 1.12},
		{"", 1.0},
		{"TIDAK_DIKENAL", 1.0},
	}
	for _, c := range cases {
		if got := netCostFactorFor(c.taxType); math.Abs(got-c.want) > 1e-12 {
			t.Fatalf("netCostFactorFor(%q) = %v, ingin %v", c.taxType, got, c.want)
		}
	}
}

// TestFinishReceivingMeneruskanFaktorHargaPokok membuktikan tipe pajak PO tidak
// hilang di jalan: faktor yang dihitung usecase harus benar-benar sampai ke
// lapisan penyimpanan, sebab hanya di sanalah harga pokok rata-rata dihitung.
func TestFinishReceivingMeneruskanFaktorHargaPokok(t *testing.T) {
	cases := []struct {
		taxType string
		want    float64
	}{
		{"NON_PPN", 1.0},
		{"TAX_INCLUSIVE", 1.0 / 1.12},
	}
	for _, c := range cases {
		grnRepo := &fakeGRNRepo{byPO: []domain.GoodsReceiptNote{{ID: 1, GRNNumber: "GRN/20261014/0001"}}}
		po := approvedPO(detail(1, 10, 10))
		po.TaxType = c.taxType
		poRepo := &fakeReceivePORepo{po: po, transitionRows: 1}
		uc := NewGoodsReceiptNoteUsecase(grnRepo, poRepo, nil)

		if err := uc.FinishReceiving(context.Background(), 7, false, 99); err != nil {
			t.Fatalf("penutupan untuk %s gagal: %v", c.taxType, err)
		}
		if math.Abs(grnRepo.finishNetCostFactor-c.want) > 1e-12 {
			t.Fatalf("faktor untuk %s = %v, ingin %v", c.taxType, grnRepo.finishNetCostFactor, c.want)
		}
	}
}
