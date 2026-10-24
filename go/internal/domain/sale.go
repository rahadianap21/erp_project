package domain

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

// =========================================================================
// KONSTANTA PENJUALAN
// =========================================================================

const (
	SaleTaxNonPPN    = "NON_PPN"
	SaleTaxExclusive = "TAX_EXCLUSIVE"
	SaleTaxInclusive = "TAX_INCLUSIVE"

	// TaxModeInclusive/Exclusive adalah nilai pengaturan PPN_PRICE_MODE:
	// INCLUSIVE berarti harga jual sudah termasuk PPN, EXCLUSIVE berarti PPN
	// ditambahkan di atas harga jual.
	TaxModeInclusive = "INCLUSIVE"
	TaxModeExclusive = "EXCLUSIVE"

	SaleDiscountNone    = "NONE"
	SaleDiscountPercent = "PERCENT"
	SaleDiscountAmount  = "AMOUNT"

	SalePaymentUnpaid  = "UNPAID"
	SalePaymentPartial = "PARTIAL"
	SalePaymentPaid    = "PAID"

	SaleStatusPosted = "POSTED"
	SaleStatusVoid   = "VOID"

	JournalSourceSale        = "SALE"
	JournalSourceSalePayment = "SALE_PAYMENT"

	PaymentChannelCash    = "CASH"
	PaymentChannelBank    = "BANK"
	PaymentChannelEWallet = "EWALLET"
	PaymentChannelCredit  = "CREDIT"
)

// =========================================================================
// ERROR
// =========================================================================

var (
	ErrSaleItemsRequired       = errors.New("penjualan harus memiliki minimal satu barang")
	ErrSaleLineInvalid         = errors.New("baris penjualan tidak valid")
	ErrSaleProductNotFound     = errors.New("produk pada baris penjualan tidak ditemukan")
	ErrSaleStokTidakCukup      = errors.New("stok produk tidak mencukupi")
	ErrSalePaymentInvalid      = errors.New("pembayaran tidak valid")
	ErrSalePaymentExceeds      = errors.New("total pembayaran melebihi total nota")
	ErrSalePaymentKurang       = errors.New("pembayaran kurang, penjualan harus lunas")
	ErrSaleMetodeGanda         = errors.New("penjualan hanya menerima satu metode pembayaran")
	ErrSaleRedeemTanpaMember   = errors.New("tukar poin membutuhkan member")
	ErrSaleRedeemMelebihiSaldo = errors.New("poin tidak cukup untuk ditukar")
	ErrSaleTaxInvalid          = errors.New("tipe pajak tidak dikenal")
	ErrSaleDiscountInvalid     = errors.New("diskon tidak valid")
	ErrSaleSnapshotChanged     = errors.New("harga atau biaya produk berubah, ulangi transaksi")
	ErrSaleNotFound            = errors.New("transaksi penjualan tidak ditemukan")
	ErrSaleNotPosted           = errors.New("transaksi penjualan tidak dalam status POSTED")
	ErrSaleAlreadyVoid         = errors.New("transaksi penjualan sudah dibatalkan")
)

// =========================================================================
// METODE PEMBAYARAN
// =========================================================================

type PaymentMethod struct {
	ID        int64  `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Channel   string `json:"channel"`
	AccountID *int64 `json:"account_id,omitempty"`
	IsCash    bool   `json:"is_cash"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

type PaymentMethodInput struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Channel   string `json:"channel"`
	AccountID *int64 `json:"account_id"`
	IsCash    bool   `json:"is_cash"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

type PaymentMethodRepository interface {
	Create(ctx context.Context, in PaymentMethodInput, actorID int64) (*PaymentMethod, error)
	Update(ctx context.Context, id int64, in PaymentMethodInput, actorID int64) error
	Delete(ctx context.Context, id, actorID int64) error
	GetByID(ctx context.Context, id int64) (*PaymentMethod, error)
	GetByIDs(ctx context.Context, ids []int64) ([]PaymentMethod, error)
	ListActive(ctx context.Context) ([]PaymentMethod, error)
	ListAll(ctx context.Context) ([]PaymentMethod, error)
}

type PaymentMethodUsecase interface {
	Create(ctx context.Context, in PaymentMethodInput, actorID int64) (*PaymentMethod, error)
	Update(ctx context.Context, id int64, in PaymentMethodInput, actorID int64) error
	Delete(ctx context.Context, id, actorID int64) error
	GetByID(ctx context.Context, id int64) (*PaymentMethod, error)
	ListActive(ctx context.Context) ([]PaymentMethod, error)
	ListAll(ctx context.Context) ([]PaymentMethod, error)
}

// =========================================================================
// PRODUK UNTUK POS
// =========================================================================

type SaleProductInfo struct {
	ID            int64
	SKU           string
	Name          string
	SellingPrice  float64
	AvgCost       float64
	DiscountType  string
	DiscountValue float64
	CurrentStock  int
	IsTaxable     bool
}

type POSProduct struct {
	ID            int64   `json:"id"`
	SKU           string  `json:"sku"`
	Barcode       string  `json:"barcode"`
	Name          string  `json:"name"`
	SellingPrice  float64 `json:"selling_price"`
	CurrentStock  int     `json:"current_stock"`
	CategoryName  string  `json:"category_name"`
	SellingUnit   string  `json:"selling_unit"`
	DiscountType  string  `json:"discount_type"`
	DiscountValue float64 `json:"discount_value"`
	IsTaxable     bool    `json:"is_taxable"`
}

type POSCategory struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

// =========================================================================
// TRANSAKSI PENJUALAN
// =========================================================================

type SaleDetail struct {
	ID             int64   `json:"id"`
	SaleID         int64   `json:"sale_id"`
	ProductID      int64   `json:"product_id"`
	SKU            string  `json:"sku"`
	ProductName    string  `json:"product_name"`
	Qty            int     `json:"qty"`
	UnitPrice      float64 `json:"unit_price"`
	DiscountType   string  `json:"discount_type"`
	DiscountValue  float64 `json:"discount_value"`
	DiscountAmount float64 `json:"discount_amount"`
	GrossAmount    float64 `json:"gross_amount"`
	LineTotal      float64 `json:"line_total"`
	DppAmount      float64 `json:"dpp_amount"`
	TaxAmount      float64 `json:"tax_amount"`
	CostUnit       float64 `json:"cost_unit"`
	CostTotal      float64 `json:"cost_total"`
	IsTaxable      bool    `json:"-"`
}

type SalePayment struct {
	ID                int64     `json:"id"`
	SaleID            int64     `json:"sale_id"`
	PaymentMethodID   int64     `json:"payment_method_id"`
	PaymentMethodName string    `json:"payment_method_name"`
	Channel           string    `json:"channel"`
	Amount            float64   `json:"amount"`
	ReferenceNumber   string    `json:"reference_number"`
	CreatedAt         time.Time `json:"created_at"`
}

type Sale struct {
	ID              int64         `json:"id"`
	SaleNumber      string        `json:"sale_number"`
	SaleDate        time.Time     `json:"sale_date"`
	MemberID        int64         `json:"member_id"`
	MemberName      string        `json:"member_name"`
	TaxType         string        `json:"tax_type"`
	Subtotal        float64       `json:"subtotal"`
	ItemDiscount    float64       `json:"item_discount"`
	DiscountType    string        `json:"discount_type"`
	DiscountValue   float64       `json:"discount_value"`
	DiscountAmount  float64       `json:"discount_amount"`
	DppAmount       float64       `json:"dpp_amount"`
	TaxAmount       float64       `json:"tax_amount"`
	TotalAmount     float64       `json:"total_amount"`
	PaidAmount      float64       `json:"paid_amount"`
	ChangeAmount    float64       `json:"change_amount"`
	CostTotal       float64       `json:"cost_total"`
	PointsEarned    int64         `json:"points_earned"`
	RedeemPoints    int64         `json:"redeem_points"`
	RedeemAmount    float64       `json:"redeem_amount"`
	VoucherID       int64         `json:"voucher_id,omitempty"`
	VoucherCode     string        `json:"voucher_code,omitempty"`
	VoucherDiscount float64       `json:"voucher_discount"`
	PaymentStatus   string        `json:"payment_status"`
	Status          string        `json:"status"`
	DueDate         *time.Time    `json:"due_date,omitempty"`
	Notes           string        `json:"notes"`
	JournalEntryID  *int64        `json:"journal_entry_id,omitempty"`
	VoidedAt        *time.Time    `json:"voided_at,omitempty"`
	VoidReason      string        `json:"void_reason,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	CreatedBy       *int64        `json:"created_by,omitempty"`
	Details         []SaleDetail  `json:"details,omitempty"`
	Payments        []SalePayment `json:"payments,omitempty"`
}

// SaleLineInput hanya memuat produk dan jumlah. Harga jual dan diskon selalu
// diambil dari master produk di server; nilai harga/diskon dari klien diabaikan.
type SaleLineInput struct {
	ProductID int64 `json:"product_id"`
	Qty       int   `json:"qty"`
}

type SalePaymentInput struct {
	PaymentMethodID int64   `json:"payment_method_id"`
	Amount          float64 `json:"amount"`
	ReferenceNumber string  `json:"reference_number"`
}

type CreateSaleInput struct {
	MemberID int64  `json:"member_id"`
	TaxType  string `json:"tax_type"`
	// DiscountType/DiscountValue dipertahankan hanya agar diskon nota manual
	// yang masih dikirim klien lama bisa ditolak secara eksplisit.
	DiscountType  string             `json:"discount_type"`
	DiscountValue float64            `json:"discount_value"`
	RedeemPoints  int64              `json:"redeem_points"`
	VoucherCode   string             `json:"voucher_code"`
	DueDate       *time.Time         `json:"due_date"`
	Notes         string             `json:"notes"`
	Items         []SaleLineInput    `json:"items"`
	Payments      []SalePaymentInput `json:"payments"`

	CreatedBy int64 `json:"-"`
}

type SaleListParam struct {
	Search       string
	StatusFilter string
	CursorID     int64
	Limit        int
	Direction    string
}

// SaleJournalPlan adalah rencana jurnal yang dibangun usecase akuntansi lalu
// ditulis repository penjualan di dalam transaksi yang sama dengan nota.
type SaleJournalPlan struct {
	EntryDate       time.Time
	Description     string
	SourceType      string
	SourceID        *int64
	SourceReference string
	ReversalOfID    *int64
	Lines           []JournalLine
	TotalDebit      float64
	TotalCredit     float64
	CreatedBy       *int64
}

// SaleJournalPoster membangun rencana jurnal penjualan (dan pembalikannya).
// Implementasinya adalah usecase akuntansi; penulisan tetap dilakukan repository
// penjualan agar nota, stok, dan jurnal atomik.
type SaleJournalPoster interface {
	PlanSaleJournal(ctx context.Context, sale *Sale, paymentAccounts map[int64]int64, actorID int64) (*SaleJournalPlan, error)
	PlanSaleReversal(ctx context.Context, sale *Sale, reason string, actorID int64) (*SaleJournalPlan, error)
}

type SaleRepository interface {
	GetProductsForSale(ctx context.Context, ids []int64) (map[int64]SaleProductInfo, error)
	// CreateSale menyimpan nota, mengurangi stok, dan menulis jurnal POSTED dalam
	// SATU transaksi. journal boleh nil bila tidak ada nilai yang perlu dijurnal.
	CreateSale(ctx context.Context, sale *Sale, journal *SaleJournalPlan) (*Sale, error)
	// VoidSale membatalkan nota POSTED: kembalikan stok, tulis jurnal pembalik,
	// dan tandai status VOID dalam satu transaksi.
	VoidSale(ctx context.Context, id, actorID int64, reason string, reversal *SaleJournalPlan) (*Sale, error)
	GetByID(ctx context.Context, id int64) (*Sale, error)
	GetListCursor(ctx context.Context, param SaleListParam) ([]Sale, int64, error)
	LookupProductsForPOS(ctx context.Context, search string, categoryID int, limit int) ([]POSProduct, error)
	GetActiveCategories(ctx context.Context) ([]POSCategory, error)
}

type SaleUsecase interface {
	CreateSale(ctx context.Context, in CreateSaleInput) (*Sale, error)
	VoidSale(ctx context.Context, id, actorID int64, reason string) error
	GetByID(ctx context.Context, id int64) (*Sale, error)
	GetAll(ctx context.Context, param SaleListParam) ([]Sale, int64, error)
}

// POSUsecase melayani kebutuhan layar kasir: bootstrap, cari produk, cari member.
type POSUsecase interface {
	Bootstrap(ctx context.Context) (*POSBootstrap, error)
	SearchProducts(ctx context.Context, search string, categoryID, limit int) ([]POSProduct, error)
	SearchMembers(ctx context.Context, search string, limit int) ([]Member, error)
}

type POSBootstrap struct {
	PaymentMethods []PaymentMethod `json:"payment_methods"`
	Categories     []POSCategory   `json:"categories"`
	TaxRate        float64         `json:"tax_rate"`
	TaxMode        string          `json:"tax_mode"`
}

// TaxConfig adalah konfigurasi pajak penjualan global: tarif PPN dan mode
// harga (INCLUSIVE/EXCLUSIVE). Dibaca dari pengaturan aplikasi.
type TaxConfig struct {
	Rate float64
	Mode string
}

// =========================================================================
// PERHITUNGAN PENJUALAN (PURE)
// =========================================================================

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

// discountAmount menghitung nominal diskon dari tipe + nilai terhadap basis.
func discountAmount(dtype string, value, base float64) (float64, error) {
	switch dtype {
	case "", SaleDiscountNone:
		return 0, nil
	case SaleDiscountPercent:
		if value < 0 || value > 100 {
			return 0, ErrSaleDiscountInvalid
		}
		return roundMoney(base * value / 100), nil
	case SaleDiscountAmount:
		if value < 0 {
			return 0, ErrSaleDiscountInvalid
		}
		d := roundMoney(value)
		if d > base {
			d = base
		}
		return d, nil
	default:
		return 0, ErrSaleDiscountInvalid
	}
}

// CalculateSale menghitung seluruh nilai nota (baris, diskon produk, DPP, PPN,
// total, HPP, status pembayaran) dari input, snapshot produk, dan konfigurasi
// pajak global. Harga jual serta diskon selalu berasal dari master produk.
// Diskon nota manual tidak didukung; tukar poin dialokasikan proporsional ke
// tiap baris dengan koreksi pembulatan (largest remainder). Fungsi ini murni
// sehingga bisa diuji tanpa database. Kecukupan stok TIDAK diperiksa di sini
// karena pengecekan final dilakukan repository di bawah kunci baris.
func CalculateSale(in CreateSaleInput, products map[int64]SaleProductInfo, cfg TaxConfig) (*Sale, error) {
	return calculateSale(in, products, cfg, nil)
}

// CalculateSaleWithVoucher menghitung nota dengan potongan voucher pada urutan:
// diskon produk, voucher, tukar poin, lalu PPN. Voucher yang diberikan sudah
// divalidasi statusnya oleh usecase; fungsi ini tetap memeriksa nilai diskon
// dan minimal belanja sebagai lapisan pertahanan.
func CalculateSaleWithVoucher(in CreateSaleInput, products map[int64]SaleProductInfo, cfg TaxConfig, voucher *Voucher) (*Sale, error) {
	return calculateSale(in, products, cfg, voucher)
}

func calculateSale(in CreateSaleInput, products map[int64]SaleProductInfo, cfg TaxConfig, voucher *Voucher) (*Sale, error) {
	if len(in.Items) == 0 {
		return nil, ErrSaleItemsRequired
	}
	if (in.DiscountType != "" && in.DiscountType != SaleDiscountNone) || in.DiscountValue != 0 {
		return nil, fmt.Errorf("%w: diskon nota manual tidak diizinkan", ErrSaleDiscountInvalid)
	}
	if in.MemberID <= 0 && in.RedeemPoints > 0 {
		return nil, ErrSaleRedeemTanpaMember
	}

	sale := &Sale{
		MemberID:     in.MemberID,
		DiscountType: SaleDiscountNone,
		Notes:        in.Notes,
		Status:       SaleStatusPosted,
		Details:      make([]SaleDetail, 0, len(in.Items)),
		Payments:     make([]SalePayment, 0, len(in.Payments)),
	}

	// 1. Hitung tiap baris dan kumpulkan basis untuk alokasi diskon/pajak.
	lineNet := make([]float64, 0, len(in.Items))
	lineTaxable := make([]bool, 0, len(in.Items))
	var subtotal, itemDiscount, costTotal float64
	for _, it := range in.Items {
		if it.ProductID <= 0 || it.Qty <= 0 {
			return nil, ErrSaleLineInvalid
		}
		p, ok := products[it.ProductID]
		if !ok {
			return nil, fmt.Errorf("%w: id %d", ErrSaleProductNotFound, it.ProductID)
		}
		unitPrice := p.SellingPrice
		if unitPrice < 0 {
			return nil, ErrSaleLineInvalid
		}
		gross := roundMoney(float64(it.Qty) * unitPrice)
		discountType := normalizeDiscountType(p.DiscountType)
		disc, err := discountAmount(discountType, p.DiscountValue, gross)
		if err != nil {
			return nil, err
		}
		net := roundMoney(gross - disc)
		costUnit := p.AvgCost
		cost := roundMoney(float64(it.Qty) * costUnit)

		sale.Details = append(sale.Details, SaleDetail{
			ProductID:      p.ID,
			SKU:            p.SKU,
			ProductName:    p.Name,
			Qty:            it.Qty,
			UnitPrice:      roundMoney(unitPrice),
			DiscountType:   discountType,
			DiscountValue:  p.DiscountValue,
			DiscountAmount: disc,
			GrossAmount:    gross,
			LineTotal:      net,
			CostUnit:       costUnit,
			CostTotal:      cost,
			IsTaxable:      p.IsTaxable,
		})
		lineNet = append(lineNet, net)
		lineTaxable = append(lineTaxable, p.IsTaxable)
		subtotal += gross
		itemDiscount += disc
		costTotal += cost
	}
	subtotal = roundMoney(subtotal)
	itemDiscount = roundMoney(itemDiscount)
	costTotal = roundMoney(costTotal)

	// 2. Diskon voucher dihitung dari nilai setelah diskon item, lalu
	// dialokasikan proporsional ke tiap baris agar DPP/PPN tiap baris tetap
	// mencerminkan potongan sebenarnya. Minimal belanja diukur dari subtotal
	// sebelum diskon item.
	if voucher != nil {
		if err := voucher.ValidateSubtotal(subtotal); err != nil {
			return nil, err
		}
		voucherAmount, err := voucher.DiscountAmount(roundMoney(subtotal - itemDiscount))
		if err != nil {
			return nil, err
		}
		voucherAlloc := allocateProportional(voucherAmount, lineNet)
		for i := range lineNet {
			lineNet[i] = roundMoney(lineNet[i] - voucherAlloc[i])
		}
		sale.VoucherID = voucher.ID
		sale.VoucherCode = voucher.Code
		sale.VoucherDiscount = voucherAmount
	}

	// Nilai neto tiap baris sudah final setelah diskon produk + voucher.
	postInvNet := lineNet

	// lineTax menghitung DPP dan PPN dari sebuah nilai neto satu baris.
	lineTax := func(net float64, taxable bool) (dpp, tax float64) {
		if !taxable {
			return roundMoney(net), 0
		}
		switch cfg.Mode {
		case TaxModeExclusive:
			dpp = roundMoney(net)
			tax = roundMoney(dpp * cfg.Rate)
		case TaxModeInclusive:
			dpp = roundMoney(net / (1 + cfg.Rate))
			tax = roundMoney(net - dpp)
		default:
			dpp = roundMoney(net)
			tax = 0
		}
		return dpp, tax
	}

	// 3. Total sebelum tukar poin dipakai menghitung batas redeem 50%.
	preRedeemTotal := 0.0
	for i := range postInvNet {
		dpp, tax := lineTax(postInvNet[i], lineTaxable[i])
		preRedeemTotal += dpp + tax
	}
	preRedeemTotal = roundMoney(preRedeemTotal)

	// 4. Tukar poin diperlakukan seperti diskon uang: mengurangi neto tiap baris
	// (dan pajaknya), dialokasikan proporsional. Batas 50% total sebelum redeem.
	usedPoints := int64(0)
	redeemAmount := 0.0
	if in.MemberID > 0 && in.RedeemPoints > 0 {
		maxAmount := roundMoney(preRedeemTotal * MemberRedeemMaxRatio)
		maxPoints := int64(maxAmount / MemberPointValueRupiah)
		usedPoints = in.RedeemPoints
		if usedPoints > maxPoints {
			usedPoints = maxPoints
		}
		if usedPoints < 0 {
			usedPoints = 0
		}
		redeemAmount = roundMoney(float64(usedPoints) * MemberPointValueRupiah)
	}
	redeemAlloc := allocateProportional(redeemAmount, postInvNet)

	// 5. Pajak final tiap baris + agregasi header.
	var dppTotal, taxTotal float64
	anyTaxable := false
	for i := range postInvNet {
		finalNet := roundMoney(postInvNet[i] - redeemAlloc[i])
		dpp, tax := lineTax(finalNet, lineTaxable[i])
		sale.Details[i].DppAmount = dpp
		sale.Details[i].TaxAmount = tax
		if lineTaxable[i] {
			anyTaxable = true
		}
		dppTotal += dpp
		taxTotal += tax
	}
	dppTotal = roundMoney(dppTotal)
	taxTotal = roundMoney(taxTotal)
	total := roundMoney(dppTotal + taxTotal)

	// TaxType diturunkan dari mode + ada/tidaknya baris kena PPN; pemilih pajak
	// per-nota sudah dihapus dari POS.
	switch {
	case !anyTaxable:
		sale.TaxType = SaleTaxNonPPN
	case cfg.Mode == TaxModeExclusive:
		sale.TaxType = SaleTaxExclusive
	default:
		sale.TaxType = SaleTaxInclusive
	}

	// Penjualan wajib lunas dengan satu metode pembayaran. Kelebihan bayar
	// dicatat sebagai kembalian (ChangeAmount) dan tidak dijurnal.
	if len(in.Payments) == 0 {
		return nil, ErrSalePaymentInvalid
	}
	if len(in.Payments) > 1 {
		return nil, ErrSaleMetodeGanda
	}
	var paid float64
	for _, pay := range in.Payments {
		if pay.PaymentMethodID <= 0 || pay.Amount <= 0 {
			return nil, ErrSalePaymentInvalid
		}
		paid += pay.Amount
		sale.Payments = append(sale.Payments, SalePayment{
			PaymentMethodID: pay.PaymentMethodID,
			Amount:          pay.Amount,
			ReferenceNumber: pay.ReferenceNumber,
		})
	}
	paid = roundMoney(paid)
	if paid < total && total-paid > 0.01 {
		return nil, ErrSalePaymentKurang
	}
	if paid < total {
		paid = total
	}
	sale.ChangeAmount = roundMoney(paid - total)

	switch {
	case paid <= 0:
		sale.PaymentStatus = SalePaymentUnpaid
	case paid < total:
		sale.PaymentStatus = SalePaymentPartial
	default:
		sale.PaymentStatus = SalePaymentPaid
	}

	sale.Subtotal = subtotal
	sale.ItemDiscount = itemDiscount
	sale.DiscountAmount = 0
	sale.DppAmount = dppTotal
	sale.TaxAmount = taxTotal
	sale.TotalAmount = total
	sale.PaidAmount = paid
	sale.CostTotal = costTotal
	sale.RedeemPoints = usedPoints
	sale.RedeemAmount = redeemAmount
	// Poin diperoleh hanya bila nota memakai member; 1 poin per Rp 10.000 total akhir.
	if sale.MemberID > 0 {
		sale.PointsEarned = int64(total / MemberPointEarnPerRupiah)
	}
	return sale, nil
}

func normalizeDiscountType(t string) string {
	if t == "" {
		return SaleDiscountNone
	}
	return t
}

// allocateProportional membagi total ke tiap baris sebanding bobotnya, lalu
// mengoreksi sisa pembulatan ke baris dengan pecahan terbesar (metode largest
// remainder) supaya jumlah hasil tepat sama dengan total. Bobot nol menerima
// nol. Nilai rupiah dibulatkan ke sen.
func allocateProportional(total float64, weights []float64) []float64 {
	out := make([]float64, len(weights))
	if len(weights) == 0 {
		return out
	}
	total = roundMoney(total)
	if total == 0 {
		return out
	}
	var sumW float64
	for _, w := range weights {
		if w > 0 {
			sumW += w
		}
	}
	if sumW <= 0 {
		return out
	}

	type fractional struct {
		idx int
		rem float64
	}
	fracs := make([]fractional, 0, len(weights))
	var allocated float64
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		exact := total * w / sumW
		floorCents := math.Floor(exact*100+1e-9) / 100
		out[i] = floorCents
		allocated = roundMoney(allocated + floorCents)
		fracs = append(fracs, fractional{idx: i, rem: exact - floorCents})
	}

	remaining := int(math.Round((total - allocated) * 100))
	if remaining > len(fracs) {
		remaining = len(fracs)
	}
	for k := 0; k < remaining && len(fracs) > 0; k++ {
		best := 0
		for j := 1; j < len(fracs); j++ {
			if fracs[j].rem > fracs[best].rem {
				best = j
			}
		}
		out[fracs[best].idx] = roundMoney(out[fracs[best].idx] + 0.01)
		fracs[best].rem -= 0.01
	}
	return out
}

// UnpaidAmount adalah sisa tagihan yang menjadi piutang (AR).
func (s *Sale) UnpaidAmount() float64 {
	return roundMoney(s.TotalAmount - s.PaidAmount)
}
