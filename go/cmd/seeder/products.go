package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// seedSupermarketProducts mengganti seluruh data master produk dengan 50 barang
// ritel yang biasa dijual di supermarket, terbagi rata ke lima kategori:
// Snack, Frozen Food, Sayur, Buah, dan Sembako.
//
// Urutan kerja:
//  1. Mengosongkan m_products beserta rantai transaksi yang bergantung padanya
//     (detail PO/GRN/penjualan) supaya data lama benar-benar tergantikan.
//  2. Menyiapkan master penunjang: kategori, satuan, dan supplier.
//  3. Menyuntikkan 50 produk.
//
// Fungsi ini idempoten untuk master penunjang: kategori/satuan/supplier yang
// sudah ada akan dipakai ulang, bukan diduplikasi.
func seedSupermarketProducts(ctx context.Context, pool *pgxpool.Pool) error {
	fmt.Println("🧹 Mengosongkan data produk lama dan transaksi turunannya...")
	if _, err := pool.Exec(ctx, `
		TRUNCATE
			m_products,
			t_purchase_order_details,
			t_goods_receipt_note_details,
			t_sales_details,
			t_sales_payments,
			t_member_points,
			t_purchase_orders,
			t_goods_receipt_notes,
			t_sales,
			t_po_number_counters,
			t_grn_number_counters,
			t_sales_number_counters
		RESTART IDENTITY CASCADE
	`); err != nil {
		return fmt.Errorf("gagal truncate data produk lama: %w", err)
	}

	villageID, err := firstVillageID(ctx, pool)
	if err != nil {
		return err
	}

	fmt.Println("📂 Menyiapkan master kategori produk...")
	categoryIDs := make(map[string]int32, len(seedCategories))
	for _, c := range seedCategories {
		id, err := ensureCategory(ctx, pool, c.Name, c.Slug, c.Description)
		if err != nil {
			return fmt.Errorf("gagal menyiapkan kategori %q: %w", c.Name, err)
		}
		categoryIDs[c.Slug] = id
	}

	fmt.Println("📏 Menyiapkan master satuan (UoM)...")
	unitIDs := make(map[string]int32, len(seedUnits))
	for _, u := range seedUnits {
		id, err := ensureUnit(ctx, pool, u.Kode, u.Name, u.Description)
		if err != nil {
			return fmt.Errorf("gagal menyiapkan satuan %q: %w", u.Kode, err)
		}
		unitIDs[u.Kode] = id
	}

	fmt.Println("🏢 Menyiapkan master supplier...")
	supplierIDs := make(map[string]int64, len(seedSuppliers))
	for _, s := range seedSuppliers {
		id, err := ensureSupplier(ctx, pool, s, villageID)
		if err != nil {
			return fmt.Errorf("gagal menyiapkan supplier %q: %w", s.Name, err)
		}
		supplierIDs[s.Key] = id
	}

	fmt.Println("📦 Menyuntikkan 50 produk supermarket...")
	for i, p := range seedProducts {
		categoryID, ok := categoryIDs[p.Category]
		if !ok {
			return fmt.Errorf("produk %s memakai kategori tak dikenal %q", p.SKU, p.Category)
		}
		supplierID, ok := supplierIDs[p.SupplierKey]
		if !ok {
			return fmt.Errorf("produk %s memakai supplier tak dikenal %q", p.SKU, p.SupplierKey)
		}
		sellUnitID, ok := unitIDs[p.SellUnit]
		if !ok {
			return fmt.Errorf("produk %s memakai satuan jual tak dikenal %q", p.SKU, p.SellUnit)
		}
		buyUnitID, ok := unitIDs[p.BuyUnit]
		if !ok {
			return fmt.Errorf("produk %s memakai satuan beli tak dikenal %q", p.SKU, p.BuyUnit)
		}

		barcode := fmt.Sprintf("89910000%05d", i+1)
		_, err := pool.Exec(ctx, `
			INSERT INTO m_products (
				sku, barcode, name, description, category_id, supplier_id,
				selling_unit_id, purchase_unit_id, conversion_rate,
				purchase_price, selling_price, min_stock_level,
				discount_type, discount_value, is_taxable, current_stock, avg_cost
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'NONE',0,TRUE,$13,$10)
		`,
			p.SKU, barcode, p.Name, p.Description,
			categoryID, supplierID,
			sellUnitID, buyUnitID, p.Conversion,
			p.BuyPrice, p.SellPrice, p.MinStock, p.Stock,
		)
		if err != nil {
			return fmt.Errorf("gagal menyuntikkan produk %s (%s): %w", p.SKU, p.Name, err)
		}
	}

	fmt.Println("✅ Seeder produk supermarket sukses: 50 produk dalam 5 kategori.")
	return nil
}

type seedCategory struct {
	Name        string
	Slug        string
	Description string
}

var seedCategories = []seedCategory{
	{"Snack", "snack", "Makanan ringan kemasan: keripik, biskuit, wafer, dan cokelat."},
	{"Frozen Food", "frozen-food", "Makanan beku siap masak: nugget, sosis, katsu, dan dimsum."},
	{"Sayur", "sayur", "Sayuran segar harian dari pemasok lokal."},
	{"Buah", "buah", "Buah segar lokal maupun impor."},
	{"Sembako", "sembako", "Sembilan bahan pokok: beras, minyak, gula, tepung, dan lainnya."},
}

type seedUnit struct {
	Kode        string
	Name        string
	Description string
}

var seedUnits = []seedUnit{
	{"PCS", "PIECES", "Satuan per buah/potong."},
	{"PACK", "PACK", "Satuan kemasan pack."},
	{"BOX", "BOX", "Satuan karton/box."},
	{"KG", "KILOGRAM", "Satuan berat kilogram."},
	{"IKAT", "IKAT", "Satuan ikatan sayuran."},
	{"BOTOL", "BOTOL", "Satuan botol."},
	{"SACHET", "SACHET", "Satuan sachet."},
	{"LITER", "LITER", "Satuan volume liter."},
}

type seedSupplier struct {
	Key     string
	Name    string
	PICName string
	Phone   string
	Email   string
	Address string
}

var seedSuppliers = []seedSupplier{
	{
		Key:     "sembako",
		Name:    "PT Sumber Pangan Nusantara",
		PICName: "Budi Santoso",
		Phone:   "0215550101",
		Email:   "procurement@sumberpangan.co.id",
		Address: "Jl. Industri Raya No. 12, Jakarta",
	},
	{
		Key:     "snack",
		Name:    "CV Aneka Snack Jaya",
		PICName: "Siti Rahayu",
		Phone:   "0341550202",
		Email:   "sales@anekasnack.co.id",
		Address: "Jl. Niaga No. 45, Malang",
	},
	{
		Key:     "fresh",
		Name:    "PT Fresh Agro Segar",
		PICName: "Andi Wijaya",
		Phone:   "0341550303",
		Email:   "order@freshagrosegar.co.id",
		Address: "Jl. Pasar Induk No. 8, Malang",
	},
}

type seedProduct struct {
	SKU         string
	Name        string
	Description string
	Category    string
	SupplierKey string
	SellUnit    string
	BuyUnit     string
	Conversion  int
	BuyPrice    int64
	SellPrice   int64
	MinStock    int
	Stock       int
}

var seedProducts = []seedProduct{
	// Snack (10)
	{"SNK-001", "Chitato Sapi Panggang 68g", "Keripik kentang rasa sapi panggang.", "snack", "snack", "PACK", "BOX", 24, 8500, 11000, 10, 120},
	{"SNK-002", "Lays Rumput Laut 68g", "Keripik kentang rasa rumput laut.", "snack", "snack", "PACK", "BOX", 24, 8800, 11500, 10, 96},
	{"SNK-003", "Oreo Original 133g", "Biskuit sandwich krim vanila.", "snack", "snack", "PACK", "BOX", 12, 7800, 9900, 12, 144},
	{"SNK-004", "Tango Wafer Vanilla 130g", "Wafer krim vanila.", "snack", "snack", "PACK", "BOX", 12, 8500, 10900, 12, 84},
	{"SNK-005", "Roma Kelapa 300g", "Biskuit kelapa.", "snack", "snack", "PACK", "BOX", 12, 10000, 12900, 12, 72},
	{"SNK-006", "Beng Beng 20g", "Wafer cokelat kacang.", "snack", "snack", "PCS", "BOX", 24, 1800, 2500, 24, 240},
	{"SNK-007", "Nabati Richeese 145g", "Wafer rasa keju.", "snack", "snack", "PACK", "BOX", 12, 9500, 12000, 12, 60},
	{"SNK-008", "Silverqueen Delfi 65g", "Cokelat batang kacang.", "snack", "snack", "PCS", "BOX", 24, 12000, 15500, 12, 48},
	{"SNK-009", "Better Corn Choco 90g", "Keripik jagung rasa cokelat.", "snack", "snack", "PACK", "BOX", 24, 6500, 8500, 12, 96},
	{"SNK-010", "Qtela Singkong Original 60g", "Keripik singkong original.", "snack", "snack", "PACK", "BOX", 24, 6000, 8000, 12, 108},

	// Frozen Food (10)
	{"FRZ-001", "Fiesta Chicken Nugget 500g", "Nugget ayam beku siap masak.", "frozen-food", "fresh", "PACK", "BOX", 12, 32000, 39000, 6, 48},
	{"FRZ-002", "So Good Chicken Nugget 500g", "Nugget ayam beku.", "frozen-food", "fresh", "PACK", "BOX", 12, 31000, 38000, 6, 36},
	{"FRZ-003", "Kanzler Single Cheese Sausage 250g", "Sosis keju beku.", "frozen-food", "fresh", "PACK", "BOX", 12, 22000, 28500, 6, 40},
	{"FRZ-004", "Fiesta Chicken Katsu 400g", "Katsu ayam beku.", "frozen-food", "fresh", "PACK", "BOX", 12, 33000, 41000, 6, 30},
	{"FRZ-005", "Belfoods Chicken Wing 500g", "Sayap ayam beku.", "frozen-food", "fresh", "PACK", "BOX", 12, 30000, 37500, 6, 24},
	{"FRZ-006", "Champ Chicken Pok 500g", "Pok ayam beku.", "frozen-food", "fresh", "PACK", "BOX", 12, 29000, 36500, 6, 24},
	{"FRZ-007", "Fiesta Spicy Chicken Fillet 400g", "Fillet ayam pedas beku.", "frozen-food", "fresh", "PACK", "BOX", 12, 34000, 42000, 6, 18},
	{"FRZ-008", "Bernardi Chicken Sausage 300g", "Sosis ayam beku.", "frozen-food", "fresh", "PACK", "BOX", 12, 24000, 30500, 6, 30},
	{"FRZ-009", "So Nice Beef Sausage 300g", "Sosis sapi beku.", "frozen-food", "fresh", "PACK", "BOX", 12, 26000, 33000, 6, 24},
	{"FRZ-010", "Fiesta Chicken Dumpling 500g", "Dimsum ayam beku.", "frozen-food", "fresh", "PACK", "BOX", 12, 30000, 38000, 6, 18},

	// Sayur (10)
	{"SYR-001", "Bayam Segar", "Bayam segar ikatan.", "sayur", "fresh", "IKAT", "IKAT", 1, 3000, 4500, 10, 40},
	{"SYR-002", "Kangkung Segar", "Kangkung segar ikatan.", "sayur", "fresh", "IKAT", "IKAT", 1, 2500, 4000, 10, 45},
	{"SYR-003", "Wortel Brastagi", "Wortel segar per kilogram.", "sayur", "fresh", "KG", "KG", 1, 12000, 16000, 5, 30},
	{"SYR-004", "Kentang Dieng", "Kentang segar per kilogram.", "sayur", "fresh", "KG", "KG", 1, 13000, 17000, 5, 35},
	{"SYR-005", "Brokoli Segar", "Brokoli segar per kilogram.", "sayur", "fresh", "KG", "KG", 1, 20000, 26000, 3, 15},
	{"SYR-006", "Sawi Hijau", "Sawi hijau segar ikatan.", "sayur", "fresh", "IKAT", "IKAT", 1, 3000, 4500, 10, 40},
	{"SYR-007", "Buncis Segar", "Buncis segar per kilogram.", "sayur", "fresh", "KG", "KG", 1, 11000, 15000, 5, 20},
	{"SYR-008", "Tomat Merah", "Tomat merah segar per kilogram.", "sayur", "fresh", "KG", "KG", 1, 10000, 14000, 5, 25},
	{"SYR-009", "Timun Lokal", "Timun lokal segar per kilogram.", "sayur", "fresh", "KG", "KG", 1, 8000, 11000, 5, 20},
	{"SYR-010", "Jagung Manis", "Jagung manis segar per kilogram.", "sayur", "fresh", "KG", "KG", 1, 9000, 13000, 5, 25},

	// Buah (10)
	{"BUH-001", "Apel Fuji", "Apel Fuji impor per kilogram.", "buah", "fresh", "KG", "KG", 1, 28000, 35000, 5, 30},
	{"BUH-002", "Jeruk Medan", "Jeruk Medan segar per kilogram.", "buah", "fresh", "KG", "KG", 1, 18000, 24000, 5, 35},
	{"BUH-003", "Pisang Cavendish", "Pisang Cavendish per kilogram.", "buah", "fresh", "KG", "KG", 1, 15000, 20000, 5, 40},
	{"BUH-004", "Semangka Merah", "Semangka merah per kilogram.", "buah", "fresh", "KG", "KG", 1, 7000, 10000, 5, 50},
	{"BUH-005", "Melon Golden", "Melon golden per kilogram.", "buah", "fresh", "KG", "KG", 1, 14000, 19000, 5, 25},
	{"BUH-006", "Anggur Merah Import", "Anggur merah impor per kilogram.", "buah", "fresh", "KG", "KG", 1, 45000, 58000, 3, 12},
	{"BUH-007", "Mangga Harum Manis", "Mangga harum manis per kilogram.", "buah", "fresh", "KG", "KG", 1, 22000, 28000, 5, 30},
	{"BUH-008", "Pear Xiang Li", "Pear xiang li impor per kilogram.", "buah", "fresh", "KG", "KG", 1, 26000, 33000, 5, 20},
	{"BUH-009", "Strawberry", "Strawberry segar per pack.", "buah", "fresh", "PACK", "PACK", 1, 25000, 32000, 5, 18},
	{"BUH-010", "Kiwi Hijau", "Kiwi hijau impor per pack.", "buah", "fresh", "PACK", "PACK", 1, 30000, 38000, 5, 15},

	// Sembako (10)
	{"SBK-001", "Beras Pandan Wangi 5kg", "Beras premium 5 kilogram.", "sembako", "sembako", "PACK", "PACK", 1, 68000, 78000, 5, 40},
	{"SBK-002", "Minyak Goreng Sania 2L", "Minyak goreng kemasan 2 liter.", "sembako", "sembako", "BOTOL", "BOX", 6, 34000, 40000, 6, 48},
	{"SBK-003", "Gula Pasir Gulaku 1kg", "Gula pasir premium 1 kilogram.", "sembako", "sembako", "PACK", "PACK", 1, 15000, 18000, 12, 60},
	{"SBK-004", "Tepung Segitiga Biru 1kg", "Tepung terigu serbaguna 1 kilogram.", "sembako", "sembako", "PACK", "PACK", 1, 12000, 14500, 12, 72},
	{"SBK-005", "Telur Ayam Negeri", "Telur ayam negeri per kilogram.", "sembako", "sembako", "KG", "KG", 1, 26000, 31000, 5, 50},
	{"SBK-006", "Garam Beryodium Refina 250g", "Garam beryodium 250 gram.", "sembako", "sembako", "PACK", "BOX", 24, 2500, 4000, 12, 96},
	{"SBK-007", "Kecap Manis Bango 520ml", "Kecap manis 520 mililiter.", "sembako", "sembako", "BOTOL", "BOX", 12, 18000, 22500, 6, 36},
	{"SBK-008", "Sarden ABC 155g", "Sarden dalam saus tomat 155 gram.", "sembako", "sembako", "PCS", "BOX", 48, 9000, 12000, 12, 96},
	{"SBK-009", "Mie Instan Indomie Goreng", "Mie instan goreng per bungkus.", "sembako", "sembako", "PCS", "BOX", 40, 2800, 3500, 40, 400},
	{"SBK-010", "Susu Kental Manis Frisian Flag 370g", "Susu kental manis 370 gram.", "sembako", "sembako", "PCS", "BOX", 48, 10000, 12500, 12, 72},
}

func firstVillageID(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `SELECT id FROM m_villages ORDER BY id LIMIT 1`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("tabel m_villages kosong: jalankan seeder wilayah terlebih dahulu")
	}
	if err != nil {
		return "", fmt.Errorf("gagal membaca master kelurahan: %w", err)
	}
	return id, nil
}

func ensureCategory(ctx context.Context, pool *pgxpool.Pool, name, slug, description string) (int32, error) {
	var id int32
	err := pool.QueryRow(ctx, `
		SELECT id FROM m_product_categories WHERE slug = $1 AND deleted_at IS NULL
	`, slug).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	err = pool.QueryRow(ctx, `
		INSERT INTO m_product_categories (name, slug, description)
		VALUES ($1, $2, $3)
		RETURNING id
	`, name, slug, description).Scan(&id)
	return id, err
}

func ensureUnit(ctx context.Context, pool *pgxpool.Pool, kode, name, description string) (int32, error) {
	var id int32
	err := pool.QueryRow(ctx, `
		SELECT id FROM m_units WHERE kode = $1 AND deleted_at IS NULL
	`, kode).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	err = pool.QueryRow(ctx, `
		INSERT INTO m_units (kode, name, description)
		VALUES ($1, $2, $3)
		RETURNING id
	`, kode, name, description).Scan(&id)
	return id, err
}

func ensureSupplier(ctx context.Context, pool *pgxpool.Pool, s seedSupplier, villageID string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
		SELECT id FROM m_suppliers WHERE email = $1 AND deleted_at IS NULL
	`, s.Email).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	err = pool.QueryRow(ctx, `
		INSERT INTO m_suppliers (name, pic_name, phone, email, address, village_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, s.Name, s.PICName, s.Phone, s.Email, s.Address, villageID).Scan(&id)
	return id, err
}
