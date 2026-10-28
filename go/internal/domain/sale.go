package domain

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/shopspring/decimal"
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
	SellingPrice  Money
	AvgCost       Decimal
	DiscountType  string
	DiscountValue Decimal
	CurrentStock  int
	IsTaxable     bool
}

type POSProduct struct {
	ID            int64   `json:"id"`
	SKU           string  `json:"sku"`
	Barcode       string  `json:"barcode"`
	Name          string  `json:"name"`
	SellingPrice  Money   `json:"selling_price"`
	CurrentStock  int     `json:"current_stock"`
	CategoryName  string  `json:"category_name"`
	SellingUnit   string  `json:"selling_unit"`
	DiscountType  string  `json:"discount_type"`
	DiscountValue Decimal `json:"discount_value"`
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
	UnitPrice      Money   `json:"unit_price"`
	DiscountType   string  `json:"discount_type"`
	DiscountValue  Decimal `json:"discount_value"`
	DiscountAmount Money   `json:"discount_amount"`
	GrossAmount    Money   `json:"gross_amount"`
	LineTotal      Money   `json:"line_total"`
	DppAmount      Money   `json:"dpp_amount"`
	TaxAmount      Money   `json:"tax_amount"`
	// CostUnit memakai Decimal karena kolom cost_unit NUMERIC(15,4): harga
	// pokok per satuan bisa punya pecahan di bawah sen.
	CostUnit  Decimal `json:"cost_unit"`
	CostTotal Money   `json:"cost_total"`
	IsTaxable bool    `json:"-"`
}

type SalePayment struct {
	ID                int64     `json:"id"`
	SaleID            int64     `json:"sale_id"`
	PaymentMethodID   int64     `json:"payment_method_id"`
	PaymentMethodName string    `json:"payment_method_name"`
	Channel           string    `json:"channel"`
	Amount            Money     `json:"amount"`
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
	Subtotal        Money         `json:"subtotal"`
	ItemDiscount    Money         `json:"item_discount"`
	DiscountType    string        `json:"discount_type"`
	DiscountValue   Decimal       `json:"discount_value"`
	DiscountAmount  Money         `json:"discount_amount"`
	DppAmount       Money         `json:"dpp_amount"`
	TaxAmount       Money         `json:"tax_amount"`
	TotalAmount     Money         `json:"total_amount"`
	PaidAmount      Money         `json:"paid_amount"`
	ChangeAmount    Money         `json:"change_amount"`
	CostTotal       Money         `json:"cost_total"`
	PointsEarned    int64         `json:"points_earned"`
	RedeemPoints    int64         `json:"redeem_points"`
	RedeemAmount    Money         `json:"redeem_amount"`
	VoucherID       int64         `json:"voucher_id,omitempty"`
	VoucherCode     string        `json:"voucher_code,omitempty"`
	VoucherDiscount Money         `json:"voucher_discount"`
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
	PaymentMethodID int64  `json:"payment_method_id"`
	Amount          Money  `json:"amount"`
	ReferenceNumber string `json:"reference_number"`
}

type CreateSaleInput struct {
	MemberID int64  `json:"member_id"`
	TaxType  string `json:"tax_type"`
	// DiscountType/DiscountValue dipertahankan hanya agar diskon nota manual
	// yang masih dikirim klien lama bisa ditolak secara eksplisit.
	DiscountType  string             `json:"discount_type"`
	DiscountValue Decimal            `json:"discount_value"`
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
	TotalDebit      Money
	TotalCredit     Money
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
	TaxRate        Decimal         `json:"tax_rate"`
	TaxMode        string          `json:"tax_mode"`
}

// TaxConfig adalah konfigurasi pajak penjualan global: tarif PPN dan mode
// harga (INCLUSIVE/EXCLUSIVE). Dibaca dari pengaturan aplikasi.
type TaxConfig struct {
	Rate Decimal
	Mode string
}

// =========================================================================
// PERHITUNGAN PENJUALAN (PURE)
// =========================================================================

// discountAmount menghitung nominal diskon dari tipe + nilai terhadap basis.
func discountAmount(dtype string, value Decimal, base Money) (Money, error) {
	switch dtype {
	case "", SaleDiscountNone:
		return Money{}, nil
	case SaleDiscountPercent:
		if value.IsNegative() || value.Cmp(DecInt(100)) > 0 {
			return Money{}, ErrSaleDiscountInvalid
		}
		return base.MulDecimal(value.DivInt(100).Raw()), nil
	case SaleDiscountAmount:
		if value.IsNegative() {
			return Money{}, ErrSaleDiscountInvalid
		}
		d := MoneyFromDecimal(value.Raw())
		if d.Cmp(base) > 0 {
			d = base
		}
		return d, nil
	default:
		return Money{}, ErrSaleDiscountInvalid
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
	if (in.DiscountType != "" && in.DiscountType != SaleDiscountNone) || !in.DiscountValue.IsZero() {
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
	lineNet := make([]Money, 0, len(in.Items))
	lineTaxable := make([]bool, 0, len(in.Items))
	var subtotal, itemDiscount, costTotal Money
	for _, it := range in.Items {
		if it.ProductID <= 0 || it.Qty <= 0 {
			return nil, ErrSaleLineInvalid
		}
		p, ok := products[it.ProductID]
		if !ok {
			return nil, fmt.Errorf("%w: id %d", ErrSaleProductNotFound, it.ProductID)
		}
		unitPrice := p.SellingPrice
		if unitPrice.IsNegative() {
			return nil, ErrSaleLineInvalid
		}
		gross := unitPrice.MulInt(int64(it.Qty))
		discountType := normalizeDiscountType(p.DiscountType)
		disc, err := discountAmount(discountType, p.DiscountValue, gross)
		if err != nil {
			return nil, err
		}
		net := gross.Sub(disc)
		costUnit := p.AvgCost
		cost := MoneyFromDecimal(costUnit.MulInt(int64(it.Qty)).Raw())

		sale.Details = append(sale.Details, SaleDetail{
			ProductID:      p.ID,
			SKU:            p.SKU,
			ProductName:    p.Name,
			Qty:            it.Qty,
			UnitPrice:      unitPrice,
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
		subtotal = subtotal.Add(gross)
		itemDiscount = itemDiscount.Add(disc)
		costTotal = costTotal.Add(cost)
	}

	// 2. Diskon voucher dihitung dari nilai setelah diskon item, lalu
	// dialokasikan proporsional ke tiap baris agar DPP/PPN tiap baris tetap
	// mencerminkan potongan sebenarnya. Minimal belanja diukur dari subtotal
	// sebelum diskon item.
	if voucher != nil {
		if err := voucher.ValidateSubtotal(subtotal); err != nil {
			return nil, err
		}
		voucherAmount, err := voucher.DiscountAmount(subtotal.Sub(itemDiscount))
		if err != nil {
			return nil, err
		}
		voucherAlloc := allocateProportional(voucherAmount, lineNet)
		for i := range lineNet {
			lineNet[i] = lineNet[i].Sub(voucherAlloc[i])
		}
		sale.VoucherID = voucher.ID
		sale.VoucherCode = voucher.Code
		sale.VoucherDiscount = voucherAmount
	}

	// Nilai neto tiap baris sudah final setelah diskon produk + voucher.
	postInvNet := lineNet

	// lineTax menghitung DPP dan PPN dari sebuah nilai neto satu baris.
	lineTax := func(net Money, taxable bool) (dpp, tax Money) {
		if !taxable {
			return net, Money{}
		}
		switch cfg.Mode {
		case TaxModeExclusive:
			dpp = net
			tax = dpp.MulDecimal(cfg.Rate.Raw())
		case TaxModeInclusive:
			dpp = net.DivDecimal(decimal.NewFromInt(1).Add(cfg.Rate.Raw()))
			tax = net.Sub(dpp)
		default:
			dpp = net
			tax = Money{}
		}
		return dpp, tax
	}

	// 3. Total sebelum tukar poin dipakai menghitung batas redeem 50%.
	var preRedeemTotal Money
	for i := range postInvNet {
		dpp, tax := lineTax(postInvNet[i], lineTaxable[i])
		preRedeemTotal = preRedeemTotal.Add(dpp).Add(tax)
	}

	// 4. Tukar poin diperlakukan seperti diskon uang: mengurangi neto tiap baris
	// (dan pajaknya), dialokasikan proporsional. Batas 50% total sebelum redeem.
	usedPoints := int64(0)
	var redeemAmount Money
	if in.MemberID > 0 && in.RedeemPoints > 0 {
		maxAmount := preRedeemTotal.MulDecimal(MemberRedeemMaxRatio.Raw())
		maxPoints := maxAmount.Cents() / (MemberPointValueRupiah * 100)
		usedPoints = in.RedeemPoints
		if usedPoints > maxPoints {
			usedPoints = maxPoints
		}
		if usedPoints < 0 {
			usedPoints = 0
		}
		redeemAmount = Rupiah(usedPoints * MemberPointValueRupiah)
	}
	redeemAlloc := allocateProportional(redeemAmount, postInvNet)

	// 5. Pajak final tiap baris + agregasi header.
	var dppTotal, taxTotal Money
	anyTaxable := false
	for i := range postInvNet {
		finalNet := postInvNet[i].Sub(redeemAlloc[i])
		dpp, tax := lineTax(finalNet, lineTaxable[i])
		sale.Details[i].DppAmount = dpp
		sale.Details[i].TaxAmount = tax
		if lineTaxable[i] {
			anyTaxable = true
		}
		dppTotal = dppTotal.Add(dpp)
		taxTotal = taxTotal.Add(tax)
	}
	total := dppTotal.Add(taxTotal)

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
	var paid Money
	for _, pay := range in.Payments {
		if pay.PaymentMethodID <= 0 || !pay.Amount.IsPositive() {
			return nil, ErrSalePaymentInvalid
		}
		paid = paid.Add(pay.Amount)
		sale.Payments = append(sale.Payments, SalePayment{
			PaymentMethodID: pay.PaymentMethodID,
			Amount:          pay.Amount,
			ReferenceNumber: pay.ReferenceNumber,
		})
	}
	if paid.Cmp(total) < 0 && total.Sub(paid).Cents() > 1 {
		return nil, ErrSalePaymentKurang
	}
	if paid.Cmp(total) < 0 {
		paid = total
	}
	sale.ChangeAmount = paid.Sub(total)

	switch {
	case !paid.IsPositive():
		sale.PaymentStatus = SalePaymentUnpaid
	case paid.Cmp(total) < 0:
		sale.PaymentStatus = SalePaymentPartial
	default:
		sale.PaymentStatus = SalePaymentPaid
	}

	sale.Subtotal = subtotal
	sale.ItemDiscount = itemDiscount
	sale.DiscountAmount = Money{}
	sale.DppAmount = dppTotal
	sale.TaxAmount = taxTotal
	sale.TotalAmount = total
	sale.PaidAmount = paid
	sale.CostTotal = costTotal
	sale.RedeemPoints = usedPoints
	sale.RedeemAmount = redeemAmount
	// Poin diperoleh hanya bila nota memakai member; 1 poin per Rp 10.000 total akhir.
	if sale.MemberID > 0 {
		sale.PointsEarned = total.Cents() / (MemberPointEarnPerRupiah * 100)
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
// nol. Seluruh perhitungan dilakukan dalam satuan sen sehingga eksak.
func allocateProportional(total Money, weights []Money) []Money {
	out := make([]Money, len(weights))
	if len(weights) == 0 || total.IsZero() {
		return out
	}
	var sumW int64
	for _, w := range weights {
		if w.Cents() > 0 {
			sumW += w.Cents()
		}
	}
	if sumW <= 0 {
		return out
	}

	type fractional struct {
		idx int
		rem *big.Int
	}
	fracs := make([]fractional, 0, len(weights))
	var allocated int64
	totalCents := big.NewInt(total.Cents())
	divisor := big.NewInt(sumW)
	for i, w := range weights {
		if w.Cents() <= 0 {
			continue
		}
		num := new(big.Int).Mul(totalCents, big.NewInt(w.Cents()))
		quotient, remainder := new(big.Int).QuoRem(num, divisor, new(big.Int))
		cents := quotient.Int64()
		out[i] = NewMoney(cents)
		allocated += cents
		fracs = append(fracs, fractional{idx: i, rem: remainder})
	}

	remaining := total.Cents() - allocated
	if remaining > int64(len(fracs)) {
		remaining = int64(len(fracs))
	}
	for k := int64(0); k < remaining && len(fracs) > 0; k++ {
		best := 0
		for j := 1; j < len(fracs); j++ {
			if fracs[j].rem.Cmp(fracs[best].rem) > 0 {
				best = j
			}
		}
		out[fracs[best].idx] = NewMoney(out[fracs[best].idx].Cents() + 1)
		fracs[best].rem = new(big.Int)
	}
	return out
}

// UnpaidAmount adalah sisa tagihan yang menjadi piutang (AR).
func (s *Sale) UnpaidAmount() Money {
	return s.TotalAmount.Sub(s.PaidAmount)
}
