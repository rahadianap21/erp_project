package router

import (
	"encoding/json"
	"go_backend/internal/config"
	"go_backend/internal/domain"
	"go_backend/internal/handler"
	"go_backend/internal/middleware"
	"go_backend/internal/middleware/limiter"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// RegisterRoutes mengelompokkan dan mendaftarkan seluruh endpoint API sistem
func RegisterRoutes(cfg *config.Config, resolver domain.PermissionResolver, userHandler *handler.UserHandler, roleHandler *handler.RoleHandler, regionHandler *handler.RegionHandler, supplierHandler *handler.SupplierHandler, bankHandler *handler.BankHandler, catHandler *handler.ProductCategoryHandler, productHandler *handler.ProductHandler, unitHandler *handler.UnitHandler, poHandler *handler.PurchaseOrderHandler, lookupHandler *handler.LookupHandler, grnHandler *handler.GoodsReceiptNoteHandler, dashboardHandler *handler.DashboardHandler, accountingHandler *handler.AccountingHandler, saleHandler *handler.SaleHandler, memberHandler *handler.MemberHandler, paymentMethodHandler *handler.PaymentMethodHandler, settingsHandler *handler.SettingsHandler) {
	// 1. Endpoint Otentikasi Publik
	loginLimiter := limiter.NewIPRateLimiter(rate.Every(time.Second), 5, 5*time.Minute)
	http.HandleFunc("/api/login", middleware.EnableCORS(loginLimiter.Middleware(userHandler.Login)))
	http.HandleFunc("/api/logout", middleware.EnableCORS(userHandler.Logout))

	// 2. Endpoint Profil Terproteksi JWT
	protectedProfileHandler := middleware.JWTMiddlewareWithResolver(cfg, resolver, userHandler.GetProfile)
	http.HandleFunc("/api/me", middleware.EnableCORS(protectedProfileHandler))

	// 3. Endpoint Admin Lama (Opsional dipisahkan atau dipertahankan)
	adminDashboardHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Selamat datang Super Admin! Ini area rahasia sistem."})
	}
	protectedAdminRoute := middleware.JWTMiddlewareWithResolver(cfg, resolver, func(w http.ResponseWriter, r *http.Request) {
		middleware.RequireRole([]int32{1}, adminDashboardHandler)(w, r)
	})
	http.HandleFunc("/api/admin/dashboard", middleware.EnableCORS(protectedAdminRoute))

	// 4. Endpoint Utama Modul CRUD Users Terproteksi JWT & RBAC
	protectedUsersRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, userHandler.UsersCRUDHandler))
	http.HandleFunc("/api/users", protectedUsersRoute)

	// 5. 🌟 PERBAIKAN UTAMA: Satukan Rute GET, POST, PUT, DELETE Master Peran ke Dalam 1 Alamat Bersih
	protectedRolesRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, roleHandler.RolesCRUDHandler))
	http.HandleFunc("/api/roles", protectedRolesRoute)

	// 6. Jalur Opsi Pengambilan Master Permissions
	protectedPermsRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, roleHandler.GetAllMasterPermissions))
	http.HandleFunc("/api/permissions", protectedPermsRoute)

	// Data referensi. Keduanya GET dan dipakai form Supplier (dropdown bank &
	// wilayah), jadi dijaga permission baca sendiri-sendiri, bukan sekadar JWT.
	protectedRegionsRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, middleware.RequirePermission("regions:read", regionHandler.HandleRegions)))
	http.HandleFunc("/api/regions", protectedRegionsRoute)

	protectedSuppliersRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, supplierHandler.SuppliersCRUDHandler))
	http.HandleFunc("/api/suppliers", protectedSuppliersRoute)

	protectedBanksRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, middleware.RequirePermission("banks:read", bankHandler.HandleGetBanks)))
	http.HandleFunc("/api/banks", protectedBanksRoute)

	protectedCatsRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, catHandler.CategoriesCRUDHandler))
	http.HandleFunc("/api/categories", protectedCatsRoute)

	protectedProductRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, productHandler.ProductsCRUDHandler))
	http.HandleFunc("/api/products", protectedProductRoute)

	protectedUnitsRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, unitHandler.HandleGetUnits))
	http.HandleFunc("/api/units", protectedUnitsRoute)

	protectedPORoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, poHandler.HandlePurchaseOrders))
	http.HandleFunc("/api/purchase-orders", protectedPORoute)

	// Sub-rute aksi dokumen PO. Terpisah dari /api/purchase-orders karena frontend
	// memanggilnya lewat path sendiri; method tetap PUT + penanda ?id=.
	protectedPOApproveRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, poHandler.HandleApprovePO))
	http.HandleFunc("/api/purchase-orders/approve", protectedPOApproveRoute)

	protectedPOCancelRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, poHandler.HandleCancelPO))
	http.HandleFunc("/api/purchase-orders/cancel", protectedPOCancelRoute)
	protectedPORejectRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, poHandler.HandleRejectPO))
	http.HandleFunc("/api/purchase-orders/reject", protectedPORejectRoute)
	http.HandleFunc("/api/goods-receipt-notes", protectedGRNsRoute(cfg, resolver, grnHandler))
	http.HandleFunc("/api/purchase-orders/finish-receiving", protectedPOFinishReceivingRoute(cfg, resolver, grnHandler))

	// 12. Lookup ringkasan untuk form Purchase Order. Dipisah dari /api/suppliers
	// dan /api/products agar form PO tidak bergantung pada permission modul master.
	protectedPOSupplierLookupRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, lookupHandler.HandleLookupSuppliers))
	http.HandleFunc("/api/po-lookup/suppliers", protectedPOSupplierLookupRoute)

	protectedPOProductLookupRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, lookupHandler.HandleLookupProducts))
	http.HandleFunc("/api/po-lookup/products", protectedPOProductLookupRoute)

	// 13. Agregat ringkas dashboard. Cukup JWT tanpa permission modul karena
	// halaman Dashboard tampil untuk semua pengguna yang sudah login.
	protectedDashboardRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, middleware.RequirePermission("dashboard:read", dashboardHandler.HandleDashboard)))
	http.HandleFunc("/api/dashboard", protectedDashboardRoute)

	// 14. Modul Akuntansi — bagan akun, periode fiskal, jurnal umum.
	protectedCoaRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.AccountsCRUDHandler))
	http.HandleFunc("/api/chart-of-accounts", protectedCoaRoute)

	protectedPeriodsRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.PeriodsCRUDHandler))
	http.HandleFunc("/api/accounting-periods", protectedPeriodsRoute)

	protectedPeriodCloseRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleClosePeriod))
	http.HandleFunc("/api/accounting-periods/close", protectedPeriodCloseRoute)

	protectedPeriodReopenRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleReopenPeriod))
	http.HandleFunc("/api/accounting-periods/reopen", protectedPeriodReopenRoute)

	protectedJournalsRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.JournalsCRUDHandler))
	http.HandleFunc("/api/journals", protectedJournalsRoute)

	protectedJournalPostRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandlePostJournal))
	http.HandleFunc("/api/journals/post", protectedJournalPostRoute)

	protectedJournalVoidRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleVoidJournal))
	http.HandleFunc("/api/journals/void", protectedJournalVoidRoute)

	protectedJournalReverseRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleReverseJournal))
	http.HandleFunc("/api/journals/reverse", protectedJournalReverseRoute)

	// 14b. Laporan keuangan — hanya membaca jurnal berstatus POSTED dan hanya
	// menuntut accounting:read.
	protectedBalanceSheetRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleBalanceSheet))
	http.HandleFunc("/api/accounting/reports/balance-sheet", protectedBalanceSheetRoute)

	protectedIncomeStatementRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleIncomeStatement))
	http.HandleFunc("/api/accounting/reports/income-statement", protectedIncomeStatementRoute)

	protectedTrialBalanceRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleTrialBalance))
	http.HandleFunc("/api/accounting/reports/trial-balance", protectedTrialBalanceRoute)

	protectedGeneralLedgerRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleGeneralLedger))
	http.HandleFunc("/api/accounting/reports/general-ledger", protectedGeneralLedgerRoute)

	// 15. Modul Penjualan & POS — master member, metode pembayaran, nota
	// penjualan, dan endpoint khusus layar kasir.
	protectedMembersRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, memberHandler.MembersCRUDHandler))
	http.HandleFunc("/api/members", protectedMembersRoute)

	protectedMembersPointsRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, memberHandler.MembersPointsHandler))
	http.HandleFunc("/api/members/points", protectedMembersPointsRoute)

	protectedPaymentMethodsRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, paymentMethodHandler.PaymentMethodsCRUDHandler))
	http.HandleFunc("/api/payment-methods", protectedPaymentMethodsRoute)

	protectedSalesRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, saleHandler.HandleSales))
	http.HandleFunc("/api/sales", protectedSalesRoute)

	protectedSaleVoidRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, saleHandler.HandleSaleVoid))
	http.HandleFunc("/api/sales/void", protectedSaleVoidRoute)

	protectedPOSBootstrapRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, saleHandler.HandlePOSBootstrap))
	http.HandleFunc("/api/pos/bootstrap", protectedPOSBootstrapRoute)

	protectedPOSProductsRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, saleHandler.HandlePOSProducts))
	http.HandleFunc("/api/pos/products", protectedPOSProductsRoute)

	protectedPOSMembersRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, saleHandler.HandlePOSMembers))
	http.HandleFunc("/api/pos/members", protectedPOSMembersRoute)

	// 16. Pengaturan aplikasi (tarif PPN & mode harga).
	protectedSettingsRoute := middleware.EnableCORS(middleware.JWTMiddlewareWithResolver(cfg, resolver, settingsHandler.HandleSettings))
	http.HandleFunc("/api/settings", protectedSettingsRoute)
}
func protectedGRNsRoute(cfg *config.Config, resolver domain.PermissionResolver, grnHandler *handler.GoodsReceiptNoteHandler) http.HandlerFunc {
	protected := middleware.JWTMiddlewareWithResolver(cfg, resolver, grnHandler.HandleGRNs)
	return middleware.EnableCORS(protected)
}

func protectedPOFinishReceivingRoute(cfg *config.Config, resolver domain.PermissionResolver, grnHandler *handler.GoodsReceiptNoteHandler) http.HandlerFunc {
	protected := middleware.JWTMiddlewareWithResolver(cfg, resolver, grnHandler.HandleFinishReceiving)
	return middleware.EnableCORS(protected)
}
