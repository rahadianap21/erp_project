package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go_backend/internal/config"
	"go_backend/internal/domain"
	"go_backend/internal/handler"
	"go_backend/internal/middleware"
	"go_backend/internal/middleware/limiter"

	"golang.org/x/time/rate"
)

const (
	loginPerMinute  = 5
	globalPerMinute = 120
	readPerMinute   = 120
	writePerMinute  = 30
	reportPerMinute = 20
)

func perMinute(n int) rate.Limit {
	return rate.Every(time.Minute / time.Duration(n))
}

func writeRateLimited(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "1")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// RegisterRoutes mengelompokkan dan mendaftarkan seluruh endpoint API sistem.
// Fungsi mengembalikan handler lengkap (sudah dilapisi rate limit global) agar
// cmd/api bisa memakainya langsung sebagai http.Server.Handler.
func RegisterRoutes(mux *http.ServeMux, cfg *config.Config, idempotencyStore domain.IdempotencyStore, resolver domain.PermissionResolver, userHandler *handler.UserHandler, roleHandler *handler.RoleHandler, regionHandler *handler.RegionHandler, supplierHandler *handler.SupplierHandler, bankHandler *handler.BankHandler, catHandler *handler.ProductCategoryHandler, productHandler *handler.ProductHandler, unitHandler *handler.UnitHandler, poHandler *handler.PurchaseOrderHandler, lookupHandler *handler.LookupHandler, grnHandler *handler.GoodsReceiptNoteHandler, dashboardHandler *handler.DashboardHandler, accountingHandler *handler.AccountingHandler, saleHandler *handler.SaleHandler, memberHandler *handler.MemberHandler, paymentMethodHandler *handler.PaymentMethodHandler, voucherHandler *handler.VoucherHandler, settingsHandler *handler.SettingsHandler, errorTrackingHandler *handler.ErrorTrackingHandler) http.Handler {
	// 0. Terapkan origin CORS dari konfigurasi (default localhost:3000).
	middleware.SetAllowedOrigin(cfg.CORSOrigin)

	trustedProxies := limiter.ParseTrustedProxies(cfg.TrustedProxies)
	keyByIP := func(r *http.Request) string {
		return limiter.ClientIP(r, trustedProxies)
	}

	// Rate limit berlapis:
	//   - global per IP: mencegah banjir request sebelum autentikasi;
	//   - per user untuk baca/tulis/laporan agar satu akun tidak menghabiskan
	//     kuota pengguna lain.
	loginLimiter := limiter.NewIPRateLimiter(
		perMinute(loginPerMinute), loginPerMinute, 5*time.Minute, cfg.TrustedProxies,
	)
	globalLimiter := limiter.NewKeyedLimiter(
		perMinute(globalPerMinute), 30, 5*time.Minute,
	)
	readLimiter := limiter.NewKeyedLimiter(
		perMinute(readPerMinute), 30, 5*time.Minute,
	)
	writeLimiter := limiter.NewKeyedLimiter(
		perMinute(writePerMinute), 10, 5*time.Minute,
	)
	reportLimiter := limiter.NewKeyedLimiter(
		perMinute(reportPerMinute), 5, 5*time.Minute,
	)

	// protectWith membungkus handler dengan JWT + resolver permission, lalu
	// menerapkan bucket per user sesuai jenis request. GetRequestAuth sudah
	// terisi karena wrapper ini berada di dalam JWTMiddlewareWithResolver.
	//
	// Idempotency berada di dalam JWT (butuh auth.UserID) tetapi di luar bucket
	// rate limit, sehingga retry yang diputar ulang tidak menghabiskan kuota
	// tulis. required=false berarti key opsional; endpoint POS memakai
	// required=true agar klien yang lupa mengirim key tidak bisa membuat nota
	// ganda saat tombol terkirim dua kali.
	protectWith := func(requireIdempotencyKey bool, next http.HandlerFunc) http.HandlerFunc {
		limited := func(w http.ResponseWriter, r *http.Request) {
			auth, ok := middleware.GetRequestAuth(r)
			if !ok {
				writeRateLimited(w, "Sesi otorisasi tidak ditemukan")
				return
			}

			var bucket *limiter.KeyedLimiter
			var key string
			switch {
			case strings.Contains(r.URL.Path, "/reports/"):
				bucket = reportLimiter
				key = fmt.Sprintf("user:%d:report", auth.UserID)
			case r.Method != http.MethodGet:
				bucket = writeLimiter
				key = fmt.Sprintf("user:%d:write", auth.UserID)
			default:
				bucket = readLimiter
				key = fmt.Sprintf("user:%d:read", auth.UserID)
			}

			if !bucket.Allow(key) {
				writeRateLimited(w, "Terlalu banyak permintaan. Coba beberapa saat lagi.")
				return
			}
			next(w, r)
		}

		var authed http.HandlerFunc = limited
		if idempotencyStore != nil {
			authed = middleware.Idempotency(idempotencyStore, requireIdempotencyKey, limited)
		}
		return middleware.EnableCORS(
			middleware.JWTMiddlewareWithResolver(cfg, resolver, authed),
		)
	}
	protect := func(next http.HandlerFunc) http.HandlerFunc {
		return protectWith(false, next)
	}
	protectIdempotent := func(next http.HandlerFunc) http.HandlerFunc {
		return protectWith(true, next)
	}

	handle := func(path string, h http.HandlerFunc) {
		mux.HandleFunc(path, h)
	}

	// 1. Endpoint Otentikasi Publik
	handle("/api/login", middleware.EnableCORS(loginLimiter.Middleware(userHandler.Login)))
	handle("/api/logout", middleware.EnableCORS(userHandler.Logout))

	// 1b. Ingest error frontend. Tanpa JWT karena pengguna yang gagal login
	// juga perlu terlacak; dilindungi kunci ingest dan rate limit per IP.
	errorIngestLimiter := limiter.NewIPRateLimiter(
		perMinute(60), 20, 5*time.Minute, cfg.TrustedProxies,
	)
	handle("/api/errors", errorIngestLimiter.Middleware(errorTrackingHandler.HandleIngest))

	// 2. Endpoint Profil Terproteksi JWT
	handle("/api/me", protect(userHandler.GetProfile))

	// 3. Endpoint Admin Lama (Opsional dipisahkan atau dipertahankan)
	adminDashboardHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Selamat datang Super Admin! Ini area rahasia sistem."})
	}
	handle("/api/admin/dashboard", protect(func(w http.ResponseWriter, r *http.Request) {
		middleware.RequireRole([]int32{1}, adminDashboardHandler)(w, r)
	}))

	// 4. Endpoint Utama Modul CRUD Users Terproteksi JWT & RBAC
	handle("/api/users", protect(userHandler.UsersCRUDHandler))

	// 5. Satukan Rute GET, POST, PUT, DELETE Master Peran ke Dalam 1 Alamat Bersih
	handle("/api/roles", protect(roleHandler.RolesCRUDHandler))

	// 6. Jalur Opsi Pengambilan Master Permissions
	handle("/api/permissions", protect(roleHandler.GetAllMasterPermissions))

	// Data referensi. Keduanya GET dan dipakai form Supplier (dropdown bank &
	// wilayah), jadi dijaga permission baca sendiri-sendiri, bukan sekadar JWT.
	handle("/api/regions", protect(middleware.RequirePermission("regions:read", regionHandler.HandleRegions)))

	handle("/api/suppliers", protect(supplierHandler.SuppliersCRUDHandler))

	handle("/api/banks", protect(middleware.RequirePermission("banks:read", bankHandler.HandleGetBanks)))

	handle("/api/categories", protect(catHandler.CategoriesCRUDHandler))

	handle("/api/products", protect(productHandler.ProductsCRUDHandler))

	handle("/api/units", protect(unitHandler.HandleGetUnits))

	handle("/api/purchase-orders", protect(poHandler.HandlePurchaseOrders))

	// Sub-rute aksi dokumen PO. Terpisah dari /api/purchase-orders karena frontend
	// memanggilnya lewat path sendiri; method tetap PUT + penanda ?id=.
	handle("/api/purchase-orders/approve", protect(poHandler.HandleApprovePO))

	handle("/api/purchase-orders/cancel", protect(poHandler.HandleCancelPO))
	handle("/api/purchase-orders/reject", protect(poHandler.HandleRejectPO))
	handle("/api/goods-receipt-notes", protect(grnHandler.HandleGRNs))
	handle("/api/purchase-orders/finish-receiving", protect(grnHandler.HandleFinishReceiving))

	// 12. Lookup ringkasan untuk form Purchase Order. Dipisah dari /api/suppliers
	// dan /api/products agar form PO tidak bergantung pada permission modul master.
	handle("/api/po-lookup/suppliers", protect(lookupHandler.HandleLookupSuppliers))

	handle("/api/po-lookup/products", protect(lookupHandler.HandleLookupProducts))

	// 13. Agregat ringkas dashboard. Cukup JWT tanpa permission modul karena
	// halaman Dashboard tampil untuk semua pengguna yang sudah login.
	handle("/api/dashboard", protect(middleware.RequirePermission("dashboard:read", dashboardHandler.HandleDashboard)))

	// 14. Modul Akuntansi — bagan akun, periode fiskal, jurnal umum.
	handle("/api/chart-of-accounts", protect(accountingHandler.AccountsCRUDHandler))

	handle("/api/accounting-periods", protect(accountingHandler.PeriodsCRUDHandler))

	handle("/api/accounting-periods/close", protect(accountingHandler.HandleClosePeriod))

	handle("/api/accounting-periods/reopen", protect(accountingHandler.HandleReopenPeriod))

	handle("/api/journals", protect(accountingHandler.JournalsCRUDHandler))

	handle("/api/journals/post", protect(accountingHandler.HandlePostJournal))

	handle("/api/journals/void", protect(accountingHandler.HandleVoidJournal))

	handle("/api/journals/reverse", protect(accountingHandler.HandleReverseJournal))

	// 14b. Laporan keuangan — hanya membaca jurnal berstatus POSTED dan hanya
	// menuntut accounting:read.
	handle("/api/accounting/reports/balance-sheet", protect(accountingHandler.HandleBalanceSheet))

	handle("/api/accounting/reports/income-statement", protect(accountingHandler.HandleIncomeStatement))

	handle("/api/accounting/reports/trial-balance", protect(accountingHandler.HandleTrialBalance))

	handle("/api/accounting/reports/general-ledger", protect(accountingHandler.HandleGeneralLedger))

	// 15. Modul Penjualan & POS — master member, metode pembayaran, nota
	// penjualan, dan endpoint khusus layar kasir.
	handle("/api/members", protect(memberHandler.MembersCRUDHandler))

	handle("/api/members/points", protect(memberHandler.MembersPointsHandler))

	handle("/api/payment-methods", protect(paymentMethodHandler.PaymentMethodsCRUDHandler))

	handle("/api/vouchers", protect(voucherHandler.VouchersCRUDHandler))

	handle("/api/sales", protectIdempotent(saleHandler.HandleSales))

	handle("/api/sales/void", protectIdempotent(saleHandler.HandleSaleVoid))

	handle("/api/pos/bootstrap", protect(saleHandler.HandlePOSBootstrap))

	handle("/api/pos/products", protect(saleHandler.HandlePOSProducts))

	handle("/api/pos/members", protect(saleHandler.HandlePOSMembers))

	handle("/api/pos/vouchers", protect(voucherHandler.HandlePOSLookupVoucher))

	// 16. Pengaturan aplikasi (tarif PPN & mode harga).
	handle("/api/settings", protect(settingsHandler.HandleSettings))

	return globalLimiter.Middleware(
		keyByIP,
		"Terlalu banyak permintaan. Coba beberapa saat lagi.",
		mux.ServeHTTP,
	)
}
