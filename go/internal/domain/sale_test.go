package domain

import (
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
			{ProductID: 3, Qty: 2, UnitPrice: 10000},
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
			{ProductID: 1, Qty: 1, UnitPrice: 100000},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 112000},
		},
	}, produkUji(), cfgExclusive())
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
			{ProductID: 1, Qty: 1, UnitPrice: 112000},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 112000},
		},
	}, produkUji(), cfgInclusive())
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
			{ProductID: 1, Qty: 1, UnitPrice: 10000}, // kena PPN
			{ProductID: 3, Qty: 1, UnitPrice: 10000}, // tanpa PPN
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
			{ProductID: 1, Qty: 1, UnitPrice: 10000}, // kena PPN
			{ProductID: 3, Qty: 1, UnitPrice: 10000}, // tanpa PPN
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

func TestCalculateSaleDiskonNotaDialokasikanProporsional(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID:      1,
		DiscountType:  SaleDiscountPercent,
		DiscountValue: 10,
		Items: []SaleLineInput{
			{ProductID: 1, Qty: 1, UnitPrice: 10000}, // neto 10000
			{ProductID: 2, Qty: 1, UnitPrice: 50000}, // neto 50000
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 54000},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if !hampirSama(sale.DiscountAmount, 6000) {
		t.Fatalf("diskon nota harus 6000, dapat %.2f", sale.DiscountAmount)
	}
	if !hampirSama(sale.TotalAmount, 54000) {
		t.Fatalf("total harus 54000, dapat %.2f", sale.TotalAmount)
	}
	cekTotalBarisSamaHeader(t, sale)
}

func TestCalculateSaleDiskonItemDanNota(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID:      1,
		DiscountType:  SaleDiscountPercent,
		DiscountValue: 10,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 2, UnitPrice: 10000, DiscountType: SaleDiscountAmount, DiscountValue: 1000},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 17100},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.Subtotal != 20000 {
		t.Fatalf("subtotal harus 20000, dapat %.2f", sale.Subtotal)
	}
	if sale.ItemDiscount != 1000 {
		t.Fatalf("diskon item harus 1000, dapat %.2f", sale.ItemDiscount)
	}
	if sale.DiscountAmount != 1900 {
		t.Fatalf("diskon nota harus 1900, dapat %.2f", sale.DiscountAmount)
	}
	if sale.TotalAmount != 17100 {
		t.Fatalf("total harus 17100, dapat %.2f", sale.TotalAmount)
	}
}

func TestCalculateSaleLebihBayarMenjadiKembalian(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 1, UnitPrice: 10000},
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
			{ProductID: 3, Qty: 1, UnitPrice: 10000},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 9000},
		},
	}, produkUji(), cfgInclusive())
	if err != ErrSalePaymentKurang {
		t.Fatalf("harus ErrSalePaymentKurang, dapat %v", err)
	}
}

func TestCalculateSaleTolakTanpaPembayaran(t *testing.T) {
	_, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 1, UnitPrice: 10000},
		},
	}, produkUji(), cfgInclusive())
	if err != ErrSalePaymentInvalid {
		t.Fatalf("harus ErrSalePaymentInvalid, dapat %v", err)
	}
}

func TestCalculateSaleTolakMetodePembayaranGanda(t *testing.T) {
	_, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 1, UnitPrice: 20000},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 10000},
			{PaymentMethodID: 2, Amount: 10000},
		},
	}, produkUji(), cfgInclusive())
	if err != ErrSaleMetodeGanda {
		t.Fatalf("harus ErrSaleMetodeGanda, dapat %v", err)
	}
}

func TestCalculateSaleTolakProdukTidakAda(t *testing.T) {
	_, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 999, Qty: 1, UnitPrice: 10000},
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
			{ProductID: 3, Qty: 1, UnitPrice: 10000},
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
			{ProductID: 4, Qty: 1, UnitPrice: 50000},
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
			{ProductID: 1, Qty: 1, UnitPrice: 112000}, // inclusive, DPP 100000 + PPN 12000
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 102000},
		},
	}, produkUji(), cfgInclusive())
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
			{ProductID: 4, Qty: 1, UnitPrice: 50000},
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
			{ProductID: 3, Qty: 1, UnitPrice: 10000},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: 9000},
		},
	}, produkUji(), cfgInclusive())
	if err != ErrSaleRedeemTanpaMember {
		t.Fatalf("harus ErrSaleRedeemTanpaMember, dapat %v", err)
	}
}
