package domain

import (
	"errors"
	"testing"
)

// produkUji: id 1 & 2 kena PPN, id 3 & 4 tidak kena PPN.
func produkUji() map[int64]SaleProductInfo {
	return map[int64]SaleProductInfo{
		1: {ID: 1, SKU: "SKU-1", Name: "Barang A", SellingPrice: Rupiah(10000), AvgCost: DecInt(6000), IsTaxable: true},
		2: {ID: 2, SKU: "SKU-2", Name: "Barang B", SellingPrice: Rupiah(50000), AvgCost: DecInt(30000), IsTaxable: true},
		3: {ID: 3, SKU: "SKU-3", Name: "Barang C", SellingPrice: Rupiah(10000), AvgCost: DecInt(6000), IsTaxable: false},
		4: {ID: 4, SKU: "SKU-4", Name: "Barang D", SellingPrice: Rupiah(50000), AvgCost: DecInt(30000), IsTaxable: false},
	}
}

func produkUjiHarga(id int64, price Money) map[int64]SaleProductInfo {
	products := produkUji()
	p := products[id]
	p.SellingPrice = price
	products[id] = p
	return products
}

func produkUjiDiskon(id int64, dtype string, value Decimal) map[int64]SaleProductInfo {
	products := produkUji()
	p := products[id]
	p.DiscountType = dtype
	p.DiscountValue = value
	products[id] = p
	return products
}

func cfgInclusive() TaxConfig { return TaxConfig{Rate: MustDec("0.12"), Mode: TaxModeInclusive} }
func cfgExclusive() TaxConfig { return TaxConfig{Rate: MustDec("0.12"), Mode: TaxModeExclusive} }

func cekTotalBarisSamaHeader(t *testing.T, sale *Sale) {
	t.Helper()
	var dpp, tax Money
	for _, d := range sale.Details {
		dpp = dpp.Add(d.DppAmount)
		tax = tax.Add(d.TaxAmount)
	}
	if dpp.Cmp(sale.DppAmount) != 0 {
		t.Fatalf("jumlah DPP baris (%s) != DPP header (%s)", dpp, sale.DppAmount)
	}
	if tax.Cmp(sale.TaxAmount) != 0 {
		t.Fatalf("jumlah PPN baris (%s) != PPN header (%s)", tax, sale.TaxAmount)
	}
	if dpp.Add(tax).Cmp(sale.TotalAmount) != 0 {
		t.Fatalf("DPP+PPN (%s) != total (%s)", dpp.Add(tax), sale.TotalAmount)
	}
}

func TestCalculateSaleProdukTanpaPPN(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 2},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: Rupiah(20000)},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.Subtotal.Cmp(Rupiah(20000)) != 0 || sale.TotalAmount.Cmp(Rupiah(20000)) != 0 {
		t.Fatalf("subtotal/total harus 20000, dapat %s/%s", sale.Subtotal, sale.TotalAmount)
	}
	if !sale.TaxAmount.IsZero() {
		t.Fatalf("PPN harus 0 untuk produk tanpa PPN, dapat %s", sale.TaxAmount)
	}
	if sale.TaxType != SaleTaxNonPPN {
		t.Fatalf("tax_type harus NON_PPN, dapat %q", sale.TaxType)
	}
	if sale.CostTotal.Cmp(Rupiah(12000)) != 0 {
		t.Fatalf("HPP harus 12000, dapat %s", sale.CostTotal)
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
			{PaymentMethodID: 1, Amount: Rupiah(112000)},
		},
	}, produkUjiHarga(1, Rupiah(100000)), cfgExclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.DppAmount.Cmp(Rupiah(100000)) != 0 {
		t.Fatalf("DPP harus 100000, dapat %s", sale.DppAmount)
	}
	if sale.TaxAmount.Cmp(Rupiah(12000)) != 0 {
		t.Fatalf("PPN harus 12000, dapat %s", sale.TaxAmount)
	}
	if sale.TotalAmount.Cmp(Rupiah(112000)) != 0 {
		t.Fatalf("total harus 112000, dapat %s", sale.TotalAmount)
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
			{PaymentMethodID: 1, Amount: Rupiah(112000)},
		},
	}, produkUjiHarga(1, Rupiah(112000)), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.DppAmount.Cmp(Rupiah(100000)) != 0 {
		t.Fatalf("DPP harus 100000, dapat %s", sale.DppAmount)
	}
	if sale.TaxAmount.Cmp(Rupiah(12000)) != 0 {
		t.Fatalf("PPN harus 12000, dapat %s", sale.TaxAmount)
	}
	if sale.TotalAmount.Cmp(Rupiah(112000)) != 0 {
		t.Fatalf("total harus 112000, dapat %s", sale.TotalAmount)
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
			{PaymentMethodID: 1, Amount: Rupiah(20000)},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.TaxAmount.Cmp(NewMoney(107143)) != 0 {
		t.Fatalf("PPN harus 1071.43 (hanya baris kena PPN), dapat %s", sale.TaxAmount)
	}
	if sale.TotalAmount.Cmp(Rupiah(20000)) != 0 {
		t.Fatalf("total harus 20000 (harga inclusive tidak berubah), dapat %s", sale.TotalAmount)
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
			{PaymentMethodID: 1, Amount: Rupiah(21200)},
		},
	}, produkUji(), cfgExclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.TaxAmount.Cmp(Rupiah(1200)) != 0 {
		t.Fatalf("PPN harus 1200, dapat %s", sale.TaxAmount)
	}
	if sale.TotalAmount.Cmp(Rupiah(21200)) != 0 {
		t.Fatalf("total harus 21200, dapat %s", sale.TotalAmount)
	}
	cekTotalBarisSamaHeader(t, sale)
}

func TestCalculateSaleMenolakDiskonNotaManual(t *testing.T) {
	_, err := CalculateSale(CreateSaleInput{
		MemberID:      1,
		DiscountType:  SaleDiscountPercent,
		DiscountValue: DecInt(10),
		Items: []SaleLineInput{
			{ProductID: 1, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: Rupiah(9000)},
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
			{PaymentMethodID: 1, Amount: Rupiah(25000)},
		},
	}, produkUjiHarga(3, Rupiah(12500)), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.Details[0].UnitPrice.Cmp(Rupiah(12500)) != 0 {
		t.Fatalf("harga harus dari master produk 12500, dapat %s", sale.Details[0].UnitPrice)
	}
	if sale.Subtotal.Cmp(Rupiah(25000)) != 0 || sale.TotalAmount.Cmp(Rupiah(25000)) != 0 {
		t.Fatalf("subtotal/total harus 25000, dapat %s/%s", sale.Subtotal, sale.TotalAmount)
	}
}

func TestCalculateSaleMemakaiDiskonDefaultProduk(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 2},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: Rupiah(19000)},
		},
	}, produkUjiDiskon(3, SaleDiscountAmount, DecInt(1000)), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.Subtotal.Cmp(Rupiah(20000)) != 0 {
		t.Fatalf("subtotal harus 20000, dapat %s", sale.Subtotal)
	}
	if sale.ItemDiscount.Cmp(Rupiah(1000)) != 0 {
		t.Fatalf("diskon item harus 1000, dapat %s", sale.ItemDiscount)
	}
	if sale.DiscountType != SaleDiscountNone || !sale.DiscountAmount.IsZero() {
		t.Fatalf("diskon nota harus NONE/0, dapat %q/%s", sale.DiscountType, sale.DiscountAmount)
	}
	if sale.TotalAmount.Cmp(Rupiah(19000)) != 0 {
		t.Fatalf("total harus 19000, dapat %s", sale.TotalAmount)
	}
	d := sale.Details[0]
	if d.DiscountType != SaleDiscountAmount || d.DiscountValue.Cmp(DecInt(1000)) != 0 || d.DiscountAmount.Cmp(Rupiah(1000)) != 0 {
		t.Fatalf("detail harus memuat diskon default produk, dapat %+v", d)
	}
}

func TestCalculateSaleLebihBayarMenjadiKembalian(t *testing.T) {
	sale, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: Rupiah(20000)},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("lebih bayar harus diterima: %v", err)
	}
	if sale.PaymentStatus != SalePaymentPaid {
		t.Fatalf("status harus PAID, dapat %q", sale.PaymentStatus)
	}
	if sale.PaidAmount.Cmp(Rupiah(20000)) != 0 {
		t.Fatalf("dibayar harus 20000, dapat %s", sale.PaidAmount)
	}
	if sale.ChangeAmount.Cmp(Rupiah(10000)) != 0 {
		t.Fatalf("kembalian harus 10000, dapat %s", sale.ChangeAmount)
	}
}

func TestCalculateSaleTolakPembayaranKurang(t *testing.T) {
	_, err := CalculateSale(CreateSaleInput{
		MemberID: 1,
		Items: []SaleLineInput{
			{ProductID: 3, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: Rupiah(9000)},
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
			{PaymentMethodID: 1, Amount: Rupiah(10000)},
			{PaymentMethodID: 2, Amount: Rupiah(10000)},
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
			{PaymentMethodID: 1, Amount: Rupiah(10000)},
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
			{PaymentMethodID: 1, Amount: Rupiah(40000)},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.RedeemPoints != 100 || sale.RedeemAmount.Cmp(Rupiah(10000)) != 0 {
		t.Fatalf("tukar poin harus 100 poin / 10000, dapat %d/%s", sale.RedeemPoints, sale.RedeemAmount)
	}
	if sale.TotalAmount.Cmp(Rupiah(40000)) != 0 {
		t.Fatalf("total setelah tukar poin harus 40000, dapat %s", sale.TotalAmount)
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
			{PaymentMethodID: 1, Amount: Rupiah(102000)},
		},
	}, produkUjiHarga(1, Rupiah(112000)), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.TotalAmount.Cmp(Rupiah(102000)) != 0 {
		t.Fatalf("total setelah tukar poin harus 102000, dapat %s", sale.TotalAmount)
	}
	if sale.DppAmount.Add(sale.TaxAmount).Cmp(sale.TotalAmount) != 0 {
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
			{PaymentMethodID: 1, Amount: Rupiah(25000)},
		},
	}, produkUji(), cfgInclusive())
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.RedeemPoints != 250 || sale.RedeemAmount.Cmp(Rupiah(25000)) != 0 {
		t.Fatalf("tukar poin dibatasi 50%% -> 250 poin/25000, dapat %d/%s", sale.RedeemPoints, sale.RedeemAmount)
	}
	if sale.TotalAmount.Cmp(Rupiah(25000)) != 0 {
		t.Fatalf("total harus 25000, dapat %s", sale.TotalAmount)
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
			{PaymentMethodID: 1, Amount: Rupiah(9000)},
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
			{PaymentMethodID: 1, Amount: Rupiah(45000)},
		},
	}, produkUji(), cfgInclusive(), &Voucher{
		ID:            9,
		Code:          "PROMO10",
		DiscountType:  VoucherDiscountPercent,
		DiscountValue: DecInt(10),
	})
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.VoucherID != 9 || sale.VoucherCode != "PROMO10" || sale.VoucherDiscount.Cmp(Rupiah(5000)) != 0 {
		t.Fatalf("voucher harus tercatat 9/PROMO10/5000, dapat %d/%s/%s", sale.VoucherID, sale.VoucherCode, sale.VoucherDiscount)
	}
	if sale.TotalAmount.Cmp(Rupiah(45000)) != 0 {
		t.Fatalf("total setelah voucher harus 45000, dapat %s", sale.TotalAmount)
	}
	if sale.TaxAmount.Cmp(NewMoney(482143)) != 0 {
		t.Fatalf("PPN harus 4821.43, dapat %s", sale.TaxAmount)
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
			{PaymentMethodID: 1, Amount: Rupiah(30000)},
		},
	}, produkUji(), cfgInclusive(), &Voucher{
		ID:            10,
		Code:          "POTONG10",
		DiscountType:  VoucherDiscountAmount,
		DiscountValue: DecInt(10000),
	})
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.VoucherDiscount.Cmp(Rupiah(10000)) != 0 || sale.RedeemAmount.Cmp(Rupiah(10000)) != 0 {
		t.Fatalf("voucher/poin harus 10000/10000, dapat %s/%s", sale.VoucherDiscount, sale.RedeemAmount)
	}
	if sale.TotalAmount.Cmp(Rupiah(30000)) != 0 {
		t.Fatalf("total harus 30000, dapat %s", sale.TotalAmount)
	}
	cekTotalBarisSamaHeader(t, sale)
}

func TestCalculateSaleVoucherPersenDibatasiMaksimalDiskon(t *testing.T) {
	sale, err := CalculateSaleWithVoucher(CreateSaleInput{
		Items: []SaleLineInput{
			{ProductID: 2, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: Rupiah(45000)},
		},
	}, produkUji(), cfgInclusive(), &Voucher{
		ID:            11,
		Code:          "PROMO50",
		DiscountType:  VoucherDiscountPercent,
		DiscountValue: DecInt(50),
		MaxDiscount:   Rupiah(5000),
	})
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if sale.VoucherDiscount.Cmp(Rupiah(5000)) != 0 {
		t.Fatalf("diskon harus dibatasi 5000, dapat %s", sale.VoucherDiscount)
	}
}

func TestCalculateSaleVoucherMinimalBelanjaDitolak(t *testing.T) {
	_, err := CalculateSaleWithVoucher(CreateSaleInput{
		Items: []SaleLineInput{
			{ProductID: 2, Qty: 1},
		},
		Payments: []SalePaymentInput{
			{PaymentMethodID: 1, Amount: Rupiah(45000)},
		},
	}, produkUji(), cfgInclusive(), &Voucher{
		ID:            12,
		Code:          "MIN60",
		DiscountType:  VoucherDiscountAmount,
		DiscountValue: DecInt(10000),
		MinPurchase:   Rupiah(60000),
	})
	if !errors.Is(err, ErrVoucherMinPurchase) {
		t.Fatalf("harus ErrVoucherMinPurchase, dapat %v", err)
	}
}
