package usecase

import (
	"context"

	"go_backend/internal/domain"
)

type grnUsecase struct {
	grnRepo domain.GoodsReceiptNoteRepository
	poRepo  domain.PurchaseOrderRepository
	// poster menerbitkan jurnal otomatis penerimaan barang. Boleh nil hanya untuk
	// pemakaian tanpa modul akuntansi (misal uji unit); produksi selalu mengisi ini.
	poster domain.GRNJournalPoster
}

func NewGoodsReceiptNoteUsecase(grnRepo domain.GoodsReceiptNoteRepository, poRepo domain.PurchaseOrderRepository, poster domain.GRNJournalPoster) domain.GoodsReceiptNoteUsecase {
	return &grnUsecase{grnRepo: grnRepo, poRepo: poRepo, poster: poster}
}

func (u *grnUsecase) CreateGRN(ctx context.Context, in domain.CreateGRNInput) (*domain.GoodsReceiptNote, error) {
	if len(in.Items) == 0 {
		return nil, domain.ErrGRNPenerimaanKosong
	}
	grn, err := u.grnRepo.CreateGRN(ctx, in)
	if err != nil {
		return nil, err
	}
	return grn, nil
}

func (u *grnUsecase) GetAll(ctx context.Context, search string, cursorID int64, limit int, direction string) ([]domain.GoodsReceiptNote, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return u.grnRepo.GetListCursor(ctx, search, cursorID, limit, direction)
}

func (u *grnUsecase) GetByID(ctx context.Context, id int64) (*domain.GoodsReceiptNote, error) {
	if id <= 0 {
		return nil, domain.NewBusinessRuleError("id grn tidak valid")
	}
	return u.grnRepo.GetByID(ctx, id)
}

func (u *grnUsecase) GetByPOID(ctx context.Context, poID int64) ([]domain.GoodsReceiptNote, error) {
	if poID <= 0 {
		return nil, domain.NewBusinessRuleError("id po tidak valid")
	}
	return u.grnRepo.GetByPOID(ctx, poID)
}

func (u *grnUsecase) FinishReceiving(ctx context.Context, poID int64, allowPartial bool, updatedBy int64) error {
	if poID <= 0 {
		return domain.NewBusinessRuleError("id po tidak valid")
	}
	po, err := u.poRepo.GetByID(ctx, poID)
	if err != nil {
		return err
	}
	if po == nil {
		return domain.NewBusinessRuleError("dokumen po tidak ditemukan")
	}
	if po.Status != "APPROVED" {
		return domain.NewBusinessRuleError("penerimaan ditolak: po tidak dalam status APPROVED")
	}

	// Minimal harus ada satu GRN. Tanpa ini PO bisaditandai RECEIVED walau tidak ada barang masuk.
	grns, err := u.grnRepo.GetByPOID(ctx, poID)
	if err != nil {
		return err
	}
	if len(grns) == 0 {
		return domain.ErrGRNPenerimaanKosong
	}

	// hitung apakah semua baris penuh
	allFull := true
	for i := range po.Items {
		if po.Items[i].QtyReceived < po.Items[i].QtyOrdered {
			allFull = false
			break
		}
	}
	if !allFull && !allowPartial {
		return domain.NewBusinessRuleError("penerimaan ditolak: masih ada sisa barang yang belum diterima")
	}

	// Penerbitan jurnal otomatis didahulukan, bukan diakhiri. Bila penerbitan
	// gagal (periode fiskal ditutup, pemetaan akun belum diisi, dsb.) PO masih
	// berstatus APPROVED sehingga pengguna cukup mengulang sampai berhasil — dan
	// penerbitan per GRN idempoten sehingga pengulangan aman. Sebaliknya bila
	// stok dan status PO dulu yang berubah, PO sudah tampak selesai padahal buku
	// besar belum terisi, dan kegagalan seperti itu justru jarang diulang.
	if u.poster != nil {
		postable := make([]*domain.GoodsReceiptNote, 0, len(grns))
		for i := range grns {
			g := &grns[i]
			// GetByPOID sudah membawa detail seluruh GRN dalam satu query. Baris
			// tanpa detail berarti datanya menghilang di antara pembacaan header
			// dan detail; jangan lanjut karena stok akan bertambah tanpa jurnal.
			if len(g.Items) == 0 {
				return domain.NewBusinessRuleError("penerimaan ditolak: detail grn %d tidak ditemukan saat penerbitan jurnal", g.ID)
			}
			postable = append(postable, g)
		}
		// Satu panggilan untuk seluruh GRN: pemetaan akun, validasi akun, dan
		// pengecekan periode dibaca sekali, tidak lagi per GRN.
		if err := u.poster.PostGRNJournals(ctx, postable, po, updatedBy); err != nil {
			return err
		}
	}

	// Finalisasi transaksional: stok baru ditambahkan di sini lalu status PO
	// berpindah ke RECEIVED. Stok tidak lagi bertambah saat GRN dibuat. Harga
	// pokok rata-rata ikut dihitung ulang di transaksi yang sama.
	//
	// expectedGRNCount dikirim supaya repository dapat menolak finalisasi bila
	// ada GRN baru yang dibuat setelah daftar di atas dibaca. GRN itu belum
	// menerbitkan jurnal; menambahkan stoknya sekarang akan membuat persediaan
	// dan buku besar tidak sinkron. Pengguna cukup mengulang, dan pengulangan
	// aman karena penerbitan jurnal per GRN idempoten.
	if err := u.grnRepo.FinishReceiving(ctx, poID, updatedBy, netCostFactorFor(po.TaxType), len(grns)); err != nil {
		return err
	}
	return nil
}

// netCostFactorFor mengubah tipe pajak PO menjadi pengali harga terima untuk
// memperoleh harga pokok perolehan (DPP).
//
// PPN masukan bukan biaya perolehan barang — ia tagihan yang diterima kembali
// dari negara — sehingga tidak boleh menambah nilai persediaan. Pada NON_PPN
// dan TAX_EXCLUSIVE harga terima sudah di luar PPN (faktor 1), sedangkan pada
// TAX_INCLUSIVE harga terima memuat PPN sehingga harus dibagi (1+PPN).
func netCostFactorFor(taxType string) domain.Decimal {
	if taxType == "TAX_INCLUSIVE" {
		return domain.DecInt(1).Div(domain.DecInt(1).Add(taxRatePPN))
	}
	return domain.DecInt(1)
}
