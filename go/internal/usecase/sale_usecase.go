package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go_backend/internal/domain"
)

type saleUsecase struct {
	saleRepo          domain.SaleRepository
	memberRepo        domain.MemberRepository
	paymentMethodRepo domain.PaymentMethodRepository
	voucherRepo       domain.VoucherRepository
	poster            domain.SaleJournalPoster
	settings          domain.TaxConfigProvider
}

// NewSaleUsecase membuat usecase transaksi penjualan. poster boleh nil hanya
// untuk pengujian tanpa modul akuntansi; produksi selalu mengisinya.
// settings boleh nil (memakai tarif/mode pajak bawaan) untuk pengujian.
// voucherRepo boleh nil selama tidak ada voucher pada transaksi.
func NewSaleUsecase(saleRepo domain.SaleRepository, memberRepo domain.MemberRepository, paymentMethodRepo domain.PaymentMethodRepository, voucherRepo domain.VoucherRepository, poster domain.SaleJournalPoster, settings domain.TaxConfigProvider) domain.SaleUsecase {
	return &saleUsecase{
		saleRepo:          saleRepo,
		memberRepo:        memberRepo,
		paymentMethodRepo: paymentMethodRepo,
		voucherRepo:       voucherRepo,
		poster:            poster,
		settings:          settings,
	}
}

// NewPOSUsecase memakai objek yang sama untuk melayani kebutuhan layar kasir.
func NewPOSUsecase(saleRepo domain.SaleRepository, memberRepo domain.MemberRepository, paymentMethodRepo domain.PaymentMethodRepository, settings domain.TaxConfigProvider) domain.POSUsecase {
	return &saleUsecase{
		saleRepo:          saleRepo,
		memberRepo:        memberRepo,
		paymentMethodRepo: paymentMethodRepo,
		settings:          settings,
	}
}

// resolveTaxConfig membaca konfigurasi pajak dari pengaturan; bila provider
// tidak ada, memakai nilai bawaan.
func (u *saleUsecase) resolveTaxConfig(ctx context.Context) (domain.TaxConfig, error) {
	if u.settings == nil {
		return domain.TaxConfig{Rate: domain.DefaultPPNRate, Mode: domain.DefaultPPNPriceMode}, nil
	}
	return u.settings.TaxConfig(ctx)
}

func (u *saleUsecase) CreateSale(ctx context.Context, in domain.CreateSaleInput) (*domain.Sale, error) {
	if len(in.Items) == 0 {
		return nil, domain.ErrSaleItemsRequired
	}
	if in.CreatedBy <= 0 {
		return nil, domain.NewBusinessRuleError("tidak terautentikasi")
	}

	ids := make([]int64, 0, len(in.Items))
	for _, it := range in.Items {
		if it.ProductID <= 0 {
			return nil, domain.ErrSaleLineInvalid
		}
		ids = append(ids, it.ProductID)
	}

	// Member opsional. Bila dipakai, ambil namanya, tempo bawaan, dan batasi
	// jumlah poin yang boleh ditukar dengan saldo yang dimiliki.
	var member *domain.Member
	var err error
	if in.MemberID > 0 {
		member, err = u.memberRepo.GetByID(ctx, in.MemberID)
		if err != nil {
			return nil, err
		}
		if member == nil {
			return nil, domain.NewBusinessRuleError("member tidak ditemukan")
		}
		if in.RedeemPoints > 0 {
			if in.RedeemPoints > member.Points {
				return nil, domain.ErrSaleRedeemMelebihiSaldo
			}
			if in.RedeemPoints < 0 {
				in.RedeemPoints = 0
			}
		}
	} else {
		in.MemberID = 0
		in.RedeemPoints = 0
	}

	// Voucher opsional, dipilih dari kode barcode. Validasi status (aktif,
	// belum terpakai, masa berlaku) dilakukan di sini; penguncian final dan
	// penandaan used_at dilakukan repository di dalam transaksi nota.
	var voucher *domain.Voucher
	in.VoucherCode = domain.NormalizeVoucherCode(in.VoucherCode)
	if in.VoucherCode != "" {
		if u.voucherRepo == nil {
			return nil, domain.NewBusinessRuleError("modul voucher tidak tersedia")
		}
		voucher, err = u.voucherRepo.GetByCode(ctx, in.VoucherCode)
		if err != nil {
			return nil, err
		}
		if voucher == nil {
			return nil, domain.ErrVoucherNotFound
		}
		if err := voucher.ValidateUsable(time.Now()); err != nil {
			return nil, err
		}
	}

	taxCfg, err := u.resolveTaxConfig(ctx)
	if err != nil {
		return nil, err
	}

	// Snapshot produk dibaca sebelum transaksi. Repository membandingkannya
	// dengan baris yang sudah dikunci; bila harga/diskon/HPP berubah, transaksi
	// diulang dengan snapshot baru agar COGS dan total tidak memakai data basi.
	const maxSnapshotAttempts = 3
	for attempt := 1; attempt <= maxSnapshotAttempts; attempt++ {
		products, err := u.saleRepo.GetProductsForSale(ctx, ids)
		if err != nil {
			return nil, err
		}

		sale, err := domain.CalculateSaleWithVoucher(in, products, taxCfg, voucher)
		if err != nil {
			return nil, err
		}

		// Pemeriksaan stok awal untuk pesan yang ramah. Penjaga sesungguhnya
		// ada di dalam transaksi sehingga tetap aman dari balapan.
		for _, d := range sale.Details {
			if p, ok := products[d.ProductID]; ok && p.CurrentStock < d.Qty {
				return nil, fmt.Errorf("%w: %s (tersedia %d)", domain.ErrSaleStokTidakCukup, d.ProductName, p.CurrentStock)
			}
		}

		now := time.Now()
		sale.SaleDate = now
		createdBy := in.CreatedBy
		sale.CreatedBy = &createdBy

		if member != nil {
			sale.MemberName = member.Name
		}
		sale.DueDate = in.DueDate
		if sale.DueDate == nil && member != nil && member.DefaultTermDays > 0 {
			d := now.AddDate(0, 0, member.DefaultTermDays)
			sale.DueDate = &d
		}

		paymentAccounts, err := u.resolvePaymentAccounts(ctx, sale.Payments)
		if err != nil {
			return nil, err
		}

		var plan *domain.SaleJournalPlan
		if u.poster != nil {
			plan, err = u.poster.PlanSaleJournal(ctx, sale, paymentAccounts, in.CreatedBy)
			if err != nil {
				return nil, err
			}
		}

		created, err := u.saleRepo.CreateSale(ctx, sale, plan)
		if errors.Is(err, domain.ErrSaleSnapshotChanged) && attempt < maxSnapshotAttempts {
			continue
		}
		return created, err
	}
	return nil, domain.ErrSaleSnapshotChanged
}

// resolvePaymentAccounts memetakan setiap metode pembayaran yang dipakai ke
// akun CoA tujuannya. Metode ber-channel CREDIT ditolak karena sisa tagihan
// dicatat sebagai piutang, bukan sebagai penerimaan.
func (u *saleUsecase) resolvePaymentAccounts(ctx context.Context, payments []domain.SalePayment) (map[int64]int64, error) {
	// Kumpulkan metode unik lalu baca semuanya dalam satu query. Sebelumnya
	// setiap baris pembayaran memicu GetByID sendiri (N+1).
	uniqueIDs := make([]int64, 0, len(payments))
	seen := make(map[int64]bool, len(payments))
	for _, p := range payments {
		if seen[p.PaymentMethodID] {
			continue
		}
		seen[p.PaymentMethodID] = true
		uniqueIDs = append(uniqueIDs, p.PaymentMethodID)
	}

	methods, err := u.paymentMethodRepo.GetByIDs(ctx, uniqueIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]domain.PaymentMethod, len(methods))
	for _, pm := range methods {
		byID[pm.ID] = pm
	}

	accounts := make(map[int64]int64, len(payments))
	for _, p := range payments {
		pm, ok := byID[p.PaymentMethodID]
		if !ok {
			return nil, domain.NewBusinessRuleError("metode pembayaran id %d tidak ditemukan", p.PaymentMethodID)
		}
		if pm.Channel == domain.PaymentChannelCredit {
			return nil, domain.NewBusinessRuleError("metode kredit tidak boleh dipakai sebagai pembayaran tunai")
		}
		if pm.AccountID == nil || *pm.AccountID <= 0 {
			return nil, domain.NewBusinessRuleError("metode pembayaran %s belum memiliki akun kas/bank", pm.Name)
		}
		accounts[p.PaymentMethodID] = *pm.AccountID
	}
	return accounts, nil
}

func (u *saleUsecase) VoidSale(ctx context.Context, id, actorID int64, reason string) error {
	if id <= 0 {
		return domain.NewBusinessRuleError("id penjualan tidak valid")
	}
	sale, err := u.saleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if sale == nil {
		return domain.ErrSaleNotFound
	}
	if sale.Status == domain.SaleStatusVoid {
		return domain.ErrSaleAlreadyVoid
	}
	if sale.Status != domain.SaleStatusPosted {
		return domain.ErrSaleNotPosted
	}

	var reversal *domain.SaleJournalPlan
	if u.poster != nil {
		reversal, err = u.poster.PlanSaleReversal(ctx, sale, strings.TrimSpace(reason), actorID)
		if err != nil {
			return err
		}
	}

	_, err = u.saleRepo.VoidSale(ctx, id, actorID, strings.TrimSpace(reason), reversal)
	return err
}

func (u *saleUsecase) GetByID(ctx context.Context, id int64) (*domain.Sale, error) {
	if id <= 0 {
		return nil, domain.NewBusinessRuleError("id penjualan tidak valid")
	}
	return u.saleRepo.GetByID(ctx, id)
}

func (u *saleUsecase) GetAll(ctx context.Context, param domain.SaleListParam) ([]domain.Sale, int64, error) {
	param.Search = strings.TrimSpace(param.Search)
	if param.Direction != "prev" {
		param.Direction = "next"
	}
	return u.saleRepo.GetListCursor(ctx, param)
}

// --- POS ---

func (u *saleUsecase) Bootstrap(ctx context.Context) (*domain.POSBootstrap, error) {
	methods, err := u.paymentMethodRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	cats, err := u.saleRepo.GetActiveCategories(ctx)
	if err != nil {
		return nil, err
	}
	taxCfg, err := u.resolveTaxConfig(ctx)
	if err != nil {
		return nil, err
	}
	out := &domain.POSBootstrap{
		PaymentMethods: methods,
		Categories:     cats,
		TaxRate:        taxCfg.Rate,
		TaxMode:        taxCfg.Mode,
	}
	return out, nil
}

func (u *saleUsecase) SearchProducts(ctx context.Context, search string, categoryID, limit int) ([]domain.POSProduct, error) {
	return u.saleRepo.LookupProductsForPOS(ctx, strings.TrimSpace(search), categoryID, limit)
}

func (u *saleUsecase) SearchMembers(ctx context.Context, search string, limit int) ([]domain.Member, error) {
	return u.memberRepo.LookupForPOS(ctx, strings.TrimSpace(search), limit)
}
