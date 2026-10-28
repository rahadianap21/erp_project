package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go_backend/internal/alert"
	"go_backend/internal/config"
	"go_backend/internal/handler"
	"go_backend/internal/middleware"
	"go_backend/internal/middleware/limiter"
	"go_backend/internal/repository/postgres"
	"go_backend/internal/router"
	"go_backend/internal/usecase"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

func main() {
	// Muat .env supaya `go run ./cmd/api` langsung jalan tanpa export manual.
	// Nilai yang sudah ada di environment selalu menang, jadi in container
	// production override tetap bekerja.
	if err := godotenv.Load(); err != nil {
		log.Printf("Peringatan: file .env tidak terbaca (%v), memakai environment yang ada\n", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Startup dibatalkan: %v\n", err)
	}

	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Gagal inisialisasi database pool: %v\n", err)
	}
	defer dbPool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("Database lokal tidak merespon: %v\n", err)
	}
	fmt.Println("Database Lokal Terhubung Sempurna!")

	// Pelacakan error: simpan ke database, alert ke webhook hanya untuk error
	// baru (atau yang muncul lagi setelah cooldown).
	errorStore := postgres.NewSQLErrorStore(dbPool)
	webhookAlerter := alert.NewWebhookAlerter(cfg.ErrorAlertWebhookURL)
	errorTrackingUsecase := usecase.NewErrorTrackingUsecase(
		errorStore, webhookAlerter, cfg.ErrorAlertCooldown,
	)
	errorTrackingHandler := handler.NewErrorTrackingHandler(
		errorTrackingUsecase, cfg.ErrorIngestKey,
	)

	idempotencyStore := postgres.NewSQLIdempotencyRepository(dbPool)

	roleRepo := postgres.NewSQLRoleRepository(dbPool)
	roleUseCase := usecase.NewRoleUsecase(roleRepo)
	roleHandler := handler.NewRoleHandler(roleUseCase)

	userRepo := postgres.NewSQLUserRepository(dbPool)
	userUseCase := usecase.NewUserUsecase(userRepo, roleRepo, cfg)
	userHandler := &handler.UserHandler{
		UserUsecase:       userUseCase,
		LoginEmailLimiter: limiter.NewKeyedLimiter(rate.Every(6*time.Second), 10, 10*time.Minute),
	}

	regionRepo := postgres.NewSQLRegionRepository(dbPool)
	regionUseCase := usecase.NewRegionUsecase(regionRepo)
	regionHandler := handler.NewRegionHandler(regionUseCase)

	supplierRepo := postgres.NewSQLSupplierRepository(dbPool)
	supplierUseCase := usecase.NewSupplierUsecase(supplierRepo)
	supplierHandler := handler.NewSupplierHandler(supplierUseCase)

	bankRepo := postgres.NewSQLBankRepository(dbPool)
	bankUseCase := usecase.NewBankUsecase(bankRepo)
	bankHandler := handler.NewBankHandler(bankUseCase)

	catRepo := postgres.NewSQLProductCategoryRepository(dbPool)
	catUseCase := usecase.NewProductCategoryUsecase(catRepo)
	catHandler := handler.NewProductCategoryHandler(catUseCase)

	productRepo := postgres.NewSQLProductRepository(dbPool)
	productUseCase := usecase.NewProductUsecase(productRepo)
	productHandler := handler.NewProductHandler(productUseCase)

	unitRepo := postgres.NewSQLUnitRepository(dbPool)
	unitUseCase := usecase.NewUnitUsecase(unitRepo)
	unitHandler := handler.NewUnitHandler(unitUseCase)

	poRepo := postgres.NewSQLPurchaseOrderRepository(dbPool)
	poUsecase := usecase.NewPurchaseOrderUsecase(poRepo, productRepo)
	poHandler := handler.NewPurchaseOrderHandler(poUsecase)

	lookupRepo := postgres.NewSQLLookupRepository(dbPool)
	lookupUsecase := usecase.NewLookupUsecase(lookupRepo)
	lookupHandler := handler.NewLookupHandler(lookupUsecase)

	grnRepo := postgres.NewSQLGoodsReceiptNoteRepository(dbPool)

	dashboardRepo := postgres.NewSQLDashboardRepository(dbPool)
	dashboardUsecase := usecase.NewDashboardUsecase(dashboardRepo)
	dashboardHandler := handler.NewDashboardHandler(dashboardUsecase)

	coaRepo := postgres.NewSQLChartOfAccountRepository(dbPool)
	periodRepo := postgres.NewSQLAccountingPeriodRepository(dbPool)
	journalRepo := postgres.NewSQLJournalRepository(dbPool)
	settingsRepo := postgres.NewSQLAccountingSettingsRepository(dbPool)
	reportRepo := postgres.NewSQLAccountingReportRepository(dbPool)
	coaUsecase := usecase.NewChartOfAccountUsecase(coaRepo)
	journalUsecase := usecase.NewJournalUsecase(journalRepo, coaRepo, periodRepo, settingsRepo, reportRepo)
	// Pengaturan akun otomatis memakai repository CoA untuk memastikan akun
	// tujuan ada, aktif, dan bukan akun header sebelum disimpan.
	accountingSettingsUsecase := usecase.NewAccountingSettingsUsecase(settingsRepo, coaRepo)
	// Periode fiskal memakai journalUsecase sebagai pengelola jurnal penutup:
	// menutup periode menerbitkan jurnal penutup, membuka kembali membaliknya.
	periodUsecase := usecase.NewAccountingPeriodUsecase(periodRepo, journalUsecase)
	reportUsecase := usecase.NewAccountingReportUsecase(reportRepo, coaRepo)
	accountingHandler := handler.NewAccountingHandler(coaUsecase, periodUsecase, journalUsecase, reportUsecase, accountingSettingsUsecase)

	// GRN menerima journalUsecase sebagai auto-poster: penutupan penerimaan
	// menerbitkan jurnal otomatis sebelum stok dan status PO berubah.
	grnUsecase := usecase.NewGoodsReceiptNoteUsecase(grnRepo, poRepo, journalUsecase)
	grnHandler := handler.NewGoodsReceiptNoteHandler(grnUsecase)

	saleRepo := postgres.NewSQLSaleRepository(dbPool)
	memberRepo := postgres.NewSQLMemberRepository(dbPool)
	paymentMethodRepo := postgres.NewSQLPaymentMethodRepository(dbPool)
	voucherRepo := postgres.NewSQLVoucherRepository(dbPool)
	appSettingsRepo := postgres.NewSQLSettingsRepository(dbPool)
	settingsUsecase := usecase.NewSettingsUsecase(appSettingsRepo)
	saleUsecase := usecase.NewSaleUsecase(saleRepo, memberRepo, paymentMethodRepo, voucherRepo, journalUsecase, settingsUsecase)
	posUsecase := usecase.NewPOSUsecase(saleRepo, memberRepo, paymentMethodRepo, settingsUsecase)
	saleHandler := handler.NewSaleHandler(saleUsecase, posUsecase)
	memberUsecase := usecase.NewMemberUsecase(memberRepo)
	memberHandler := handler.NewMemberHandler(memberUsecase)
	paymentMethodUsecase := usecase.NewPaymentMethodUsecase(paymentMethodRepo)
	paymentMethodHandler := handler.NewPaymentMethodHandler(paymentMethodUsecase)
	voucherUsecase := usecase.NewVoucherUsecase(voucherRepo)
	voucherHandler := handler.NewVoucherHandler(voucherUsecase)
	settingsHandler := handler.NewSettingsHandler(settingsUsecase)

	mux := http.NewServeMux()
	handler := router.RegisterRoutes(mux, cfg, idempotencyStore, userRepo, userHandler, roleHandler, regionHandler, supplierHandler, bankHandler, catHandler, productHandler, unitHandler, poHandler, lookupHandler, grnHandler, dashboardHandler, accountingHandler, saleHandler, memberHandler, paymentMethodHandler, voucherHandler, settingsHandler, errorTrackingHandler)

	// Middleware terluar: menangkap panic dan seluruh respons 5xx, termasuk
	// kegagalan di middleware JWT/rate limit.
	handler = middleware.TrackErrors(errorTrackingUsecase, handler)

	fmt.Printf("Server produksi siap melayani di http://localhost:%s\n", cfg.Port)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
