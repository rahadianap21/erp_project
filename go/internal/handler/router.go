package handler

import (
	"encoding/json"
	"go_backend/internal/config"
	"go_backend/internal/domain"
	"go_backend/internal/handler/limiter"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// RegisterRoutes mengelompokkan dan mendaftarkan seluruh endpoint API sistem
func RegisterRoutes(cfg *config.Config, resolver domain.PermissionResolver, userHandler *UserHandler, roleHandler *RoleHandler, regionHandler *RegionHandler, supplierHandler *SupplierHandler, bankHandler *BankHandler, catHandler *ProductCategoryHandler, productHandler *ProductHandler, unitHandler *UnitHandler, poHandler *PurchaseOrderHandler, lookupHandler *LookupHandler, grnHandler *GoodsReceiptNoteHandler, dashboardHandler *DashboardHandler, accountingHandler *AccountingHandler, saleHandler *SaleHandler, memberHandler *MemberHandler, paymentMethodHandler *PaymentMethodHandler, settingsHandler *SettingsHandler) {
	// 1. Endpoint Otentikasi Publik
	loginLimiter := limiter.NewIPRateLimiter(rate.Every(time.Second), 5, 5*time.Minute)
	http.HandleFunc("/api/login", EnableCORS(loginLimiter.Middleware(userHandler.Login)))
	http.HandleFunc("/api/logout", EnableCORS(userHandler.Logout))

	// 2. Endpoint Profil Terproteksi JWT
	protectedProfileHandler := JWTMiddlewareWithResolver(cfg, resolver, userHandler.GetProfile)
	http.HandleFunc("/api/me", EnableCORS(protectedProfileHandler))

	// 3. Endpoint Admin Lama (Opsional dipisahkan atau dipertahankan)
	adminDashboardHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Selamat datang Super Admin! Ini area rahasia sistem."})
	}
	protectedAdminRoute := JWTMiddlewareWithResolver(cfg, resolver, func(w http.ResponseWriter, r *http.Request) {
		RequireRole([]int32{1}, adminDashboardHandler)(w, r)
	})
	http.HandleFunc("/api/admin/dashboard", EnableCORS(protectedAdminRoute))

	// 4. Endpoint Utama Modul CRUD Users Terproteksi JWT & RBAC
	protectedUsersRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, userHandler.UsersCRUDHandler))
	http.HandleFunc("/api/users", protectedUsersRoute)

	// 5. 🌟 PERBAIKAN UTAMA: Satukan Rute GET, POST, PUT, DELETE Master Peran ke Dalam 1 Alamat Bersih
	protectedRolesRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, roleHandler.RolesCRUDHandler))
	http.HandleFunc("/api/roles", protectedRolesRoute)

	// 6. Jalur Opsi Pengambilan Master Permissions
	protectedPermsRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, roleHandler.GetAllMasterPermissions))
	http.HandleFunc("/api/permissions", protectedPermsRoute)

	// Data referensi. Keduanya GET dan dipakai form Supplier (dropdown bank &
	// wilayah), jadi dijaga permission baca sendiri-sendiri, bukan sekadar JWT.
	protectedRegionsRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, RequirePermission("regions:read", regionHandler.HandleRegions)))
	http.HandleFunc("/api/regions", protectedRegionsRoute)

	protectedSuppliersRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, supplierHandler.SuppliersCRUDHandler))
	http.HandleFunc("/api/suppliers", protectedSuppliersRoute)

	protectedBanksRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, RequirePermission("banks:read", bankHandler.HandleGetBanks)))
	http.HandleFunc("/api/banks", protectedBanksRoute)

	protectedCatsRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, catHandler.CategoriesCRUDHandler))
	http.HandleFunc("/api/categories", protectedCatsRoute)

	protectedProductRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, productHandler.ProductsCRUDHandler))
	http.HandleFunc("/api/products", protectedProductRoute)

	protectedUnitsRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, unitHandler.HandleGetUnits))
	http.HandleFunc("/api/units", protectedUnitsRoute)

	protectedPORoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, poHandler.HandlePurchaseOrders))
	http.HandleFunc("/api/purchase-orders", protectedPORoute)

	// Sub-rute aksi dokumen PO. Terpisah dari /api/purchase-orders karena frontend
	// memanggilnya lewat path sendiri; method tetap PUT + penanda ?id=.
	protectedPOApproveRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, poHandler.HandleApprovePO))
	http.HandleFunc("/api/purchase-orders/approve", protectedPOApproveRoute)

	protectedPOCancelRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, poHandler.HandleCancelPO))
	http.HandleFunc("/api/purchase-orders/cancel", protectedPOCancelRoute)
	protectedPORejectRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, poHandler.HandleRejectPO))
	http.HandleFunc("/api/purchase-orders/reject", protectedPORejectRoute)
	http.HandleFunc("/api/goods-receipt-notes", protectedGRNsRoute(cfg, resolver, grnHandler))
	http.HandleFunc("/api/purchase-orders/finish-receiving", protectedPOFinishReceivingRoute(cfg, resolver, grnHandler))

	// 12. Lookup ringkasan untuk form Purchase Order. Dipisah dari /api/suppliers
	// dan /api/products agar form PO tidak bergantung pada permission modul master.
	protectedPOSupplierLookupRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, lookupHandler.HandleLookupSuppliers))
	http.HandleFunc("/api/po-lookup/suppliers", protectedPOSupplierLookupRoute)

	protectedPOProductLookupRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, lookupHandler.HandleLookupProducts))
	http.HandleFunc("/api/po-lookup/products", protectedPOProductLookupRoute)

	// 13. Agregat ringkas dashboard. Cukup JWT tanpa permission modul karena
	// halaman Dashboard tampil untuk semua pengguna yang sudah login.
	protectedDashboardRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, RequirePermission("dashboard:read", dashboardHandler.HandleDashboard)))
	http.HandleFunc("/api/dashboard", protectedDashboardRoute)

	// 14. Modul Akuntansi — bagan akun, periode fiskal, jurnal umum.
	protectedCoaRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.AccountsCRUDHandler))
	http.HandleFunc("/api/chart-of-accounts", protectedCoaRoute)

	protectedPeriodsRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.PeriodsCRUDHandler))
	http.HandleFunc("/api/accounting-periods", protectedPeriodsRoute)

	protectedPeriodCloseRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleClosePeriod))
	http.HandleFunc("/api/accounting-periods/close", protectedPeriodCloseRoute)

	protectedPeriodReopenRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleReopenPeriod))
	http.HandleFunc("/api/accounting-periods/reopen", protectedPeriodReopenRoute)

	protectedJournalsRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.JournalsCRUDHandler))
	http.HandleFunc("/api/journals", protectedJournalsRoute)

	protectedJournalPostRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandlePostJournal))
	http.HandleFunc("/api/journals/post", protectedJournalPostRoute)

	protectedJournalVoidRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleVoidJournal))
	http.HandleFunc("/api/journals/void", protectedJournalVoidRoute)

	protectedJournalReverseRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleReverseJournal))
	http.HandleFunc("/api/journals/reverse", protectedJournalReverseRoute)

	// 14b. Laporan keuangan — hanya membaca jurnal berstatus POSTED dan hanya
	// menuntut accounting:read.
	protectedBalanceSheetRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleBalanceSheet))
	http.HandleFunc("/api/accounting/reports/balance-sheet", protectedBalanceSheetRoute)

	protectedIncomeStatementRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleIncomeStatement))
	http.HandleFunc("/api/accounting/reports/income-statement", protectedIncomeStatementRoute)

	protectedTrialBalanceRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleTrialBalance))
	http.HandleFunc("/api/accounting/reports/trial-balance", protectedTrialBalanceRoute)

	protectedGeneralLedgerRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, accountingHandler.HandleGeneralLedger))
	http.HandleFunc("/api/accounting/reports/general-ledger", protectedGeneralLedgerRoute)

	// 15. Modul Penjualan & POS — master member, metode pembayaran, nota
	// penjualan, dan endpoint khusus layar kasir.
	protectedMembersRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, memberHandler.MembersCRUDHandler))
	http.HandleFunc("/api/members", protectedMembersRoute)

	protectedMembersPointsRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, memberHandler.MembersPointsHandler))
	http.HandleFunc("/api/members/points", protectedMembersPointsRoute)

	protectedPaymentMethodsRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, paymentMethodHandler.PaymentMethodsCRUDHandler))
	http.HandleFunc("/api/payment-methods", protectedPaymentMethodsRoute)

	protectedSalesRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, saleHandler.HandleSales))
	http.HandleFunc("/api/sales", protectedSalesRoute)

	protectedSaleVoidRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, saleHandler.HandleSaleVoid))
	http.HandleFunc("/api/sales/void", protectedSaleVoidRoute)

	protectedPOSBootstrapRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, saleHandler.HandlePOSBootstrap))
	http.HandleFunc("/api/pos/bootstrap", protectedPOSBootstrapRoute)

	protectedPOSProductsRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, saleHandler.HandlePOSProducts))
	http.HandleFunc("/api/pos/products", protectedPOSProductsRoute)

	protectedPOSMembersRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, saleHandler.HandlePOSMembers))
	http.HandleFunc("/api/pos/members", protectedPOSMembersRoute)

	// 16. Pengaturan aplikasi (tarif PPN & mode harga).
	protectedSettingsRoute := EnableCORS(JWTMiddlewareWithResolver(cfg, resolver, settingsHandler.HandleSettings))
	http.HandleFunc("/api/settings", protectedSettingsRoute)
}
func protectedGRNsRoute(cfg *config.Config, resolver domain.PermissionResolver, grnHandler *GoodsReceiptNoteHandler) http.HandlerFunc {
	protected := JWTMiddlewareWithResolver(cfg, resolver, grnHandler.HandleGRNs)
	return EnableCORS(protected)
}

func protectedPOFinishReceivingRoute(cfg *config.Config, resolver domain.PermissionResolver, grnHandler *GoodsReceiptNoteHandler) http.HandlerFunc {
	protected := JWTMiddlewareWithResolver(cfg, resolver, grnHandler.HandleFinishReceiving)
	return EnableCORS(protected)
}
