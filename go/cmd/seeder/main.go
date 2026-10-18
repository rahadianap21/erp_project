package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Struktur penampung skema API publik wilayah Indonesia resmi (Sumber: Binderbyte/BPS API)
type RegionItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func main() {
	// 1. Koneksi ke Database PostgreSQL Anda
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/central_db?sslmode=disable"
	}
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Gagal membaca konfigurasi DB: %v", err)
	}

	// Gunakan timeout agak panjang karena proses data transfer berskala besar
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Gagal terhubung ke database pool: %v", err)
	}
	defer pool.Close()

	fmt.Println("⏳ Memulai Proses Seeding Wilayah Administratif Nasional Indonesia...")
	ctx := context.Background()

	// 🌟 SEED DATA PROVINSI PILIHAN (CONTOH POPULASI UTAMA)
	provinces := []RegionItem{
		{"11", "ACEH"}, {"12", "SUMATERA UTARA"}, {"13", "SUMATERA BARAT"}, {"14", "RIAU"},
		{"15", "JAMBI"}, {"16", "SUMATERA SELATAN"}, {"17", "BENGKULU"}, {"18", "LAMPUNG"},
		{"19", "KEPULAUAN BANGKA BELITUNG"}, {"21", "KEPULAUAN RIAU"}, {"31", "DKI JAKARTA"},
		{"32", "JAWA BARAT"}, {"33", "JAWA TENGAH"}, {"34", "DI YOGYAKARTA"}, {"35", "JAWA TIMUR"},
		{"36", "BANTEN"}, {"51", "BALI"}, {"52", "NUSA TENGGARA BARAT"}, {"53", "NUSA TENGGARA TIMUR"},
	}

	fmt.Println("🚀 Menyuntikkan Data Master Provinsi...")
	for _, p := range provinces {
		_, err := pool.Exec(ctx, "INSERT INTO m_provinces (id, name) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING", p.ID, p.Name)
		if err != nil {
			log.Printf("Gagal insert provinsi %s: %v", p.Name, err)
		}
	}

	// 🌟 SEED DATA KOTA/KABUPATEN (SAMPLE PRIMER JABODETABEK & JAWA TIMUR)
	regencies := []struct {
		ID         string
		ProvinceID string
		Name       string
	}{
		{"3171", "31", "KOTA JAKARTA PUSAT"}, {"3172", "31", "KOTA JAKARTA UTARA"},
		{"3173", "31", "KOTA JAKARTA BARAT"}, {"3174", "31", "KOTA JAKARTA SELATAN"},
		{"3175", "31", "KOTA JAKARTA TIMUR"}, {"3273", "32", "KOTA BANDUNG"},
		{"3275", "32", "KOTA BEKASI"}, {"3276", "32", "KOTA DEPOK"},
		{"3578", "35", "KOTA SURABAYA"}, {"3573", "35", "KOTA MALANG"},
	}

	fmt.Println("🚀 Menyuntikkan Data Master Kota/Kabupaten...")
	for _, r := range regencies {
		_, err := pool.Exec(ctx, "INSERT INTO m_regencies (id, province_id, name) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING", r.ID, r.ProvinceID, r.Name)
		if err != nil {
			log.Printf("Gagal insert regency %s: %v", r.Name, err)
		}
	}

	// 🌟 SEED DATA KECAMATAN (SAMPLE PRIMER MALANG KOTA & JAKARTA PUSAT)
	districts := []struct {
		ID        string
		RegencyID string
		Name      string
	}{
		{"317101", "3171", "GAMBIR"}, {"317102", "3171", "SAWAH BESAR"},
		{"357301", "3573", "BLIMBING"}, {"357302", "3573", "KLOJEN"},
		{"357303", "3573", "LOWOKWARU"}, {"357304", "3573", "SUKUN"},
	}

	fmt.Println("🚀 Menyuntikkan Data Master Kecamatan...")
	for _, d := range districts {
		_, err := pool.Exec(ctx, "INSERT INTO m_districts (id, regency_id, name) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING", d.ID, d.RegencyID, d.Name)
		if err != nil {
			log.Printf("Gagal insert district %s: %v", d.Name, err)
		}
	}

	// 🌟 SEED DATA KELURAHAN (SAMPLE WILAYAH UTAMA KLOJEN MALANG CITY)
	villages := []struct {
		ID         string
		DistrictID string
		Name       string
		PostalCode string
	}{
		{"3573021001", "357302", "KLOJEN", "65111"},
		{"3573021002", "357302", "RAMPAL CELAKET", "65111"},
		{"3573021003", "357302", "SAMAAN", "65112"},
		{"3573021004", "357302", "PENANGGUNGAN", "65113"},
		{"3573021005", "357302", "GADINGASRI", "65115"},
	}

	fmt.Println("🚀 Menyuntikkan Data Master Kelurahan...")
	for _, v := range villages {
		_, err := pool.Exec(ctx, "INSERT INTO m_villages (id, district_id, name, postal_code) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING", v.ID, v.DistrictID, v.Name, v.PostalCode)
		if err != nil {
			log.Printf("Gagal insert village %s: %v", v.Name, err)
		}
	}

	fmt.Println("✅ Proses Injeksi Seeder Regional Indonesia Sukses Sempurna!")

	// 🌟 SEED DATA PRODUK SUPERMARKET (50 PRODUK, 5 KATEGORI)
	if err := seedSupermarketProducts(ctx, pool); err != nil {
		log.Fatalf("Gagal seeding produk supermarket: %v", err)
	}
}
