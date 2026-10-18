package postgres

import (
	"errors"
	"testing"

	"go_backend/internal/domain"
)

// TestSellingRateSUsesConversionRateProduk memastikan barang yang dibeli dalam
// satuan besar dikonversi ke satuan jual memakai conversion_rate produk.
func TestSellingRateSUsesConversionRateProduk(t *testing.T) {
	// Purchase unit produk = 3, selling unit produk = 6, rate = 12.
	// Baris PO memakai satuan beli produk, jadi 1 pcs Beli = 12 pcs Jual.
	l := poLine{PurchaseUnitID: 3, ProductPurchaseUnitID: 3, ProductSellingUnitID: 6, ConversionRate: 12}

	got, err := sellingRate(l)
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if got != 12 {
		t.Fatalf("rate harus 12, dapat %d", got)
	}
}

// TestSellingRateSatuanSamaSelaluSatu memastikan pair beli/jual yang identik
// tidak perlu konversi walau conversion_rate produk tidak default.
func TestSellingRateSatuanSamaSelaluSatu(t *testing.T) {
	l := poLine{PurchaseUnitID: 4, ProductPurchaseUnitID: 4, ProductSellingUnitID: 4, ConversionRate: 9}

	got, err := sellingRate(l)
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if got != 1 {
		t.Fatalf("rate harus 1 untuk satuan identik, dapat %d", got)
	}
}

// TestSellingRateRateTidakValidDijepitSatu menjaga data produk yang
// conversion_rate-nya nol atau negatif (mis. hasil default yang lupa diisi).
func TestSellingRateRateTidakValidDijepitSatu(t *testing.T) {
	for _, bad := range []int32{0, -1, -100} {
		l := poLine{PurchaseUnitID: 3, ProductPurchaseUnitID: 3, ProductSellingUnitID: 6, ConversionRate: bad}
		got, err := sellingRate(l)
		if err != nil {
			t.Fatalf("rate %d tidak boleh error: %v", bad, err)
		}
		if got != 1 {
			t.Fatalf("rate %d harus dijepit jadi 1, dapat %d", bad, got)
		}
	}
}

// TestSellingRateSatuanTidakDikenalDitolak mencegah konversi asal-asalan saat
// satuan baris PO tidak ada di master produk.
func TestSellingRateSatuanTidakDikenalDitolak(t *testing.T) {
	l := poLine{PurchaseUnitID: 99, ProductPurchaseUnitID: 3, ProductSellingUnitID: 6, ConversionRate: 12}

	_, err := sellingRate(l)
	if !errors.Is(err, domain.ErrGRNSatuanBermasalah) {
		t.Fatalf("satuan asing harus ditolak, dapat %v", err)
	}
}

// TestSellingRateSatuanJualProdukDiterima memastikan baris PO yang dicatat
// dalam satuan jual produk diperlakukan sebagai 1:1.
func TestSellingRateSatuanJualProdukDiterima(t *testing.T) {
	// PurchaseUnitID baris = selling unit produk, jadi tidak perlu konversi.
	l := poLine{PurchaseUnitID: 6, ProductPurchaseUnitID: 3, ProductSellingUnitID: 6, ConversionRate: 12}

	got, err := sellingRate(l)
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}
	if got != 1 {
		t.Fatalf("rate harus 1, dapat %d", got)
	}
}
