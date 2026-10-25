package domain

import (
	"errors"
	"math"
	"testing"
)

// produkUji: id 1 & 2 kena PPN, id 3 & 4 tidak kena PPN.
func produkUji() map[int64]SaleProductInfo {
	return map[int64]SaleProductInfo{
		1: {ID: 1, SKU: "SKU-1", Name: "Barang A", SellingPrice: 10000, AvgCost: 6000, IsTaxable: true},
		2: {ID: 2, SKU: "SKU-2", Name: "Barang B", SellingPrice: 50000, AvgCost: 30000, IsTaxable: true},
		3: {ID: 3, SKU: "SKU-3", Name: "Barang C", SellingPrice: 10000, AvgCost: 6000, IsTaxable: false},
		4: {ID: 4, SKU: "SKU-4", Name: "Barang D", SellingPrice: 50000, AvgCost: 30000, IsTaxable: false},
	}
}

func produkUjiHarga(id int64, price float64) map[int64]SaleProductInfo {
	products := produkUji()
	p := products[id]
	p.SellingPrice = price
	products[id] = p
	return products
}

func produkUjiDiskon(id int64, dtype string, value float64) map[int64]SaleProductInfo {
	products := produkUji()
	p := products[id]
	p.DiscountType = dtype
	p.DiscountValue = value
	products[id] = p
	return products
}

func cfgInclusive() TaxConfig { return TaxConfig{Rate: 0.12, Mode: TaxModeInclusive} }
func cfgExclusive() TaxConfig { return TaxConfig{Rate: 0.12, Mode: TaxModeExclusive} }

func hampirSama(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func cekTotalBarisSamaHeader(t *testing.T, sale *Sale) {
	t.Helper()
	var dpp, tax float64
	for _, d := range sale.Details {
		dpp += d.DppAmount
		tax += d.TaxAmount
	}
	if !hampirSama(dpp, sale.DppAmount) {
		t.Fatalf("jumlah DPP baris (%.2f) != DPP header (%.2f)", dpp, sale.DppAmount)
	}
	if !hampirSama(tax, sale.TaxAmount) {
		t.Fatalf("jumlah PPN baris (%.2f) != PPN header (%.2f)", tax, sale.TaxAmount)
	}
	if !hampirSama(dpp+tax, sale.TotalAmount) {
		t.Fatalf("DPP+PPN (%.2f) != total (%.2f)", dpp+tax, sale.TotalAmount)
	}
}

func TestCalculateSaleProdukTanpaPPN(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 2},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 20000},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.Subtotal != 20000 || sale.TotalAmount != 20000 {
		t.Fatalf("subtotal/total harus 20000, dapat %.2f/%.2f", sale.Subtotal, sale.TotalAmount)
	}
	if sale.TaxAmount != 0 {
		t.Fatalf("PPN harus 0 untuk produk tanpa PPN, dapat %.2f", sale.TaxAmount)
	}
	if sale.TaxType != SaleTaxNonPPN {
		t.Fatalf("tax_type harus NON_PPN, dapat %q", sale.TaxType)
	}
	if sale.CostTotal != 12000 {
		t.Fatalf("HPP harus 12000, dapat %.2f", sale.CostTotal)
	}
	if sale.PaymentStatus != SalePaymentPaid {
		t.Fatalf("status harus PAID, dapat %q", sale.PaymentStatus)
	}
	cekTotalBarisSamaHeader(t, sale)
}

func TestCalculateSaleExclusiveMenghitungPPN(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 1, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 112000},
		},
	}, produkUjiHarga(1, 100000), cfgExclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.DppAmount != 100000 {
		t.Fatalf("DPP harus 100000, dapat %.2f", sale.DppAmount)
	}
	if sale.TaxAmount != 12000 {
		t.Fatalf("PPN harus 12000, dapat %.2f", sale.TaxAmount)
	}
	if sale.TotalAmount != 112000 {
		t.Fatalf("total harus 112000, dapat %.2f", sale.TotalAmount)
	}
	if sale.TaxType != SaleTaxExclusive {
		t.Fatalf("tax_type harus TAX_EXCLUSIVE, dapat %q", sale.TaxType)
	}
	if sale.PaymentStatus != SalePaymentPaid {
		t.Fatalf("status harus PAID, dapat %q", sale.PaymentStatus)
	}
	cekTotalBarisSamaHeader(t, sale)
}

func TestCalculateSaleInclusiveMenguraiPPN(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 1, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 112000},
		},
	}, produkUjiHarga(1, 112000), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.DppAmount != 100000 {
		t.Fatalf("DPP harus 100000, dapat %.2f", sale.DppAmount)
	}
	if sale.TaxAmount != 12000 {
		t.Fatalf("PPN harus 12000, dapat %.2f", sale.TaxAmount)
	}
	if sale.TotalAmount != 112000 {
		t.Fatalf("total harus 112000, dapat %.2f", sale.TotalAmount)
	}
	if sale.TaxType != SaleTaxInclusive {
		t.Fatalf("tax_type harus TAX_INCLUSIVE, dapat %q", sale.TaxType)
	}
	cekTotalBarisSamaHeader(t, sale)
}

func TestCalculateSaleCampuranKenaDanTanpaPPNInclusive(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 1, Qty: 1}, // kena PPN
			{ProductID: 3, Qty: 1}, // tanpa PPN
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 20000},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if !hampirSama(sale.TaxAmount, 1071.43) {
		t.Fatalf("PPN harus 1071.43 (hanya baris kena PPN), dapat %.2f", sale.TaxAmount)
	}
	if !hampirSama(sale.TotalAmount, 20000) {
		t.Fatalf("total harus 20000 (harga inclusive tidak berubah), dapat %.2f", sale.TotalAmount)
	}
	cekTotalBarisSamaHeader(t, sale)
}

func TestCalculateSaleCampuranKenaDanTanpaPPNExclusive(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 1, Qty: 1}, // kena PPN
			{ProductID: 3, Qty: 1}, // tanpa PPN
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 21200},
		},
	}, produkUji(), cfgExclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.TaxAmount != 1200 {
		t.Fatalf("PPN harus 1200, dapat %.2f", sale.TaxAmount)
	}
	if !hampirSama(sale.TotalAmount, 21200) {
		t.Fatalf("total harus 21200, dapat %.2f", sale.TotalAmount)
	}
	cekTotalBarisSamaHeader(t, sale)
}

func TestCalculateSaleMenolakDiskonNotaManual(t *testing.T) {
	_, err := CalculateSale(CreateSaleInput{
		MemberID:      1,
		DiscountType:  SaleDiscountPercent,
		DiscountValue: 10,
		Items: []SaleLineInput{
			{ProductID: 1, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 9000},
		},
	}, produkUji(), cfgInclusive())
	if !errors.Is(err, ErrSaleDiscountInvalid) {
		t.Fatalf("harus ErrSaleDiscountInvalid, dapat %v", err)
	}
}

func TestCalculateSaleMenolakQtyTidakPositif(t *testing.T) {
	for _, qty := range []int{0, -1, -5} {
		_, err := CalculateSale(CreateSaleInput{
			Items: []SaleLineInput{{ProductID: 1, Qty: qty}},
		}, produkUji(), cfgInclusive())
		if !errors.Is(err, ErrSaleLineInvalid) {
			t.Fatalf("qty %d harus ErrSaleLineInvalid, dapat %v", qty, err)
		}
	}
}

func TestCalculateSaleMemakaiHargaServer(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 2},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 25000},
		},
	}, produkUjiHarga(3, 12500), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.Details[0].UnitPrice != 12500 {
		t.Fatalf("harga harus dari master produk 12500, dapat %.2f", sale.Details[0].UnitPrice)
	}
	if sale.Subtotal != 25000 || sale.TotalAmount != 25000 {
		t.Fatalf("subtotal/total harus 25000, dapat %.2f/%.2f", sale.Subtotal, sale.TotalAmount)
	}
}

func TestCalculateSaleMemakaiDiskonDefaultProduk(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 2},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 19000},
		},
	}, produkUjiDiskon(3, SaleDiscountAmount, 1000), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.Subtotal != 20000 {
		t.Fatalf("subtotal harus 20000, dapat %.2f", sale.Subtotal)
	}
	if sale.ItemDiscount != 1000 {
		t.Fatalf("diskon item harus 1000, dapat %.2f", sale.ItemDiscount)
	}
	if sale.DiscountType != SaleDiscountNone || sale.DiscountAmount != 0 {
		t.Fatalf("diskon nota harus NONE/0, dapat %q/%.2f", sale.DiscountType, sale.DiscountAmount)
	}
	if sale.TotalAmount != 19000 {
		t.Fatalf("total harus 19000, dapat %.2f", sale.TotalAmount)
	}
	if sale.Details[0].DiscountType != SaleDiscountAmount || sale.Details[0].DiscountValue != 1000 || sale.Details[0].DiscountAmount != 1000 {
		t.Fatalf("detail harus memuat diskon default produk, dapat %+v", sale.Details[0])
	}
}

func TestCalculateSaleLebihBayarMenjadiKembalian(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 20000},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("lebih bayar harus diterima: %v", err)
	}
	if sale.PaymentStatus != SalePaymentPaid {
		t.Fatalf("status harus PAID, dapat %q", sale.PaymentStatus)
	}
	if sale.PaidAmount != 20000 {
		t.Fatalf("dibayar harus 20000, dapat %.2f", sale.PaidAmount)
	}
	if sale.ChangeAmount != 10000 {
		t.Fatalf("kembalian harus 10000, dapat %.2f", sale.ChangeAmount)
	}
}

func TestCalculateSaleTolakPembayaranKurang(t *testing.T) {
	_, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 9000},
		},
	}, produkUji(), cfgInclusive())
	if !errors.Is(err, ErrSalePaymentKurang) {
		t.Fatalf("harus ErrSalePaymentKurang, dapat %v", err)
	}
}

func TestCalculateSaleTolakTanpaPembayaran(t *testing.T) {
	_, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 1},
		},
	}, produkUji(), cfgInclusive())
	if !errors.Is(err, ErrSalePaymentInvalid) {
		t.Fatalf("harus ErrSalePaymentInvalid, dapat %v", err)
	}
}

func TestCalculateSaleTolakMetodePembayaranGanda(t *testing.T) {
	_, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 10000},
			{PaymentMethodID: 2, Amount: 10000},
		},
	}, produkUji(), cfgInclusive())
	if !errors.Is(err, ErrSaleMetodeGanda) {
		t.Fatalf("harus ErrSaleMetodeGanda, dapat %v", err)
	}
}

func TestCalculateSaleTolakProdukTidakAda(t *testing.T) {
	_, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 999, Qty: 1},
		},
	}, produkUji(), cfgInclusive())
	if err == nil {
		t.Fatal("harus menolak produk yang tidak ada")
	}
}

func TestCalculateSaleTanpaMemberTetapSah(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID: 0,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 10000},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("penjualan tanpa member harus sah: %v", err)
	}
	if sale.PointsEarned != 0 {
		t.Fatalf("tanpa member tidak boleh dapat poin, dapat %d", sale.PointsEarned)
	}
}

func TestCalculateSaleTukarPoinMengurangiTotal(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID:     1,
		RedeemPoints: 100,
		Items: []SaleLineInput{
			{ProductID: 4, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 40000},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.RedeemPoints != 100 || sale.RedeemAmount != 10000 {
		t.Fatalf("tukar poin harus 100 poin / 10000, dapat %d/%.2f", sale.RedeemPoints, sale.RedeemAmount)
	}
	if sale.TotalAmount != 40000 {
		t.Fatalf("total setelah tukar poin harus 40000, dapat %.2f", sale.TotalAmount)
	}
	if sale.PointsEarned != 4 {
		t.Fatalf("poin perolehan harus 4, dapat %d", sale.PointsEarned)
	}
}

func TestCalculateSaleTukarPoinDenganPPN(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID:     1,
		RedeemPoints: 100,
		Items: []SaleLineInput{
			{ProductID: 1, Qty: 1}, // inclusive, DPP 100000 + PPN 12000
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 102000},
		},
	}, produkUjiHarga(1, 112000), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if !hampirSama(sale.TotalAmount, 102000) {
		t.Fatalf("total setelah tukar poin harus 102000, dapat %.2f", sale.TotalAmount)
	}
	if !hampirSama(sale.DppAmount+sale.TaxAmount, sale.TotalAmount) {
		t.Fatalf("DPP+PPN harus sama dengan total")
	}
	cekTotalBarisSamaHeader(t, sale)
}

func TestCalculateSaleTukarPoinDibatasiLimaPuluhPersen(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID:     1,
		RedeemPoints: 600,
		Items: []SaleLineInput{
			{ProductID: 4, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 25000},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.RedeemPoints != 250 || sale.RedeemAmount != 25000 {
		t.Fatalf("tukar poin dibatasi 50%% -> 250 poin/25000, dapat %d/%.2f", sale.RedeemPoints, sale.RedeemAmount)
	}
	if sale.TotalAmount != 25000 {
		t.Fatalf("total harus 25000, dapat %.2f", sale.TotalAmount)
	}
}

func TestCalculateSaleTolakRedeemTanpaMember(t *testing.T) {
	_, err := CalculateSale(CreateSaleInput{
		MemberID:     0,
		RedeemPoints: 10,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 9000},
		},
	}, produkUji(), cfgInclusive())
	if !errors.Is(err, ErrSaleRedeemTanpaMember) {
		t.Fatalf("harus ErrSaleRedeemTanpaMember, dapat %v", err)
	}
}

func TestCalculateSaleVoucherPersenMengurangiTotalDanPPN(t *testing.T) {
	sale, err := CalculateSaleWithVoucher(CreateSaleInput{
		Items: []SaleLineInput{
			{ProductID: 2, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 45000},
		},
	}, produkUji(), cfgInclusive(), &Voucher{
		ID:            9,
		Code:          "PROMO10",
		DiscountType:  VoucherDiscountPercent,
		DiscountValue: 10,
	})
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.VoucherID != 9 || sale.VoucherCode != "PROMO10" || sale.VoucherDiscount != 5000 {
		t.Fatalf("voucher harus tercatat 9/PROMO10/5000, dapat %d/%s/%.2f", sale.VoucherID, sale.VoucherCode, sale.VoucherDiscount)
	}
	if sale.TotalAmount != 45000 {
		t.Fatalf("total setelah voucher harus 45000, dapat %.2f", sale.TotalAmount)
	}
	if sale.TaxAmount != 4821.43 {
		t.Fatalf("PPN harus 4821.43, dapat %.2f", sale.TaxAmount)
	}
	cekTotalBarisSamaHeader(t, sale)
}

func TestCalculateSaleVoucherNominalDenganPoin(t *testing.T) {
	sale, err := CalculateSaleWithVoucher(CreateSaleInput{
		MemberID:     1,
		RedeemPoints: 100,
		Items: []SaleLineInput{
			{ProductID: 2, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 30000},
		},
	}, produkUji(), cfgInclusive(), &Voucher{
		ID:            10,
		Code:          "POTONG10",
		DiscountType:  VoucherDiscountAmount,
		DiscountValue: 10000,
	})
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.VoucherDiscount != 10000 || sale.RedeemAmount != 10000 {
		t.Fatalf("voucher/poin harus 10000/10000, dapat %.2f/%.2f", sale.VoucherDiscount, sale.RedeemAmount)
	}
	if sale.TotalAmount != 30000 {
		t.Fatalf("total harus 30000, dapat %.2f", sale.TotalAmount)
	}
	cekTotalBarisSamaHeader(t, sale)
}

func TestCalculateSaleVoucherPersenDibatasiMaksimalDiskon(t *testing.T) {
	sale, err := CalculateSaleWithVoucher(CreateSaleInput{
		Items: []SaleLineInput{
			{ProductID: 2, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 45000},
		},
	}, produkUji(), cfgInclusive(), &Voucher{
		ID:            11,
		Code:          "PROMO50",
		DiscountType:  VoucherDiscountPercent,
		DiscountValue: 50,
		MaxDiscount:   5000,
	})
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.VoucherDiscount != 5000 {
		t.Fatalf("diskon harus dibatasi 5000, dapat %.2f", sale.VoucherDiscount)
	}
}

func TestCalculateSaleVoucherMinimalBelanjaDitolak(t *testing.T) {
	_, err := CalculateSaleWithVoucher(CreateSaleInput{
		Items: []SaleLineInput{
			{ProductID: 2, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 45000},
		},
	}, produkUji(), cfgInclusive(), &Voucher{
		ID:            12,
		Code:          "MIN60",
		DiscountType:  VoucherDiscountAmount,
		DiscountValue: 10000,
		MinPurchase:   60000,
	})
	if !errors.Is(err, ErrVoucherMinPurchase) {
		t.Fatalf("harus ErrVoucherMinPurchase, dapat %v", err)
	}
}
