package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go_backend/internal/config"
	"go_backend/internal/handler"
	"go_backend/internal/repository/postgres"
	"go_backend/internal/usecase"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Muat .env supaya `go run ./cmd/api` langsung jalan tanpa export manual.
	// Variabel yang sudah ada di environment selalu menang, jadi in container
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

	roleRepo := postgres.NewSQLRoleRepository(dbPool)
	roleUseCase := usecase.NewRoleUsecase(roleRepo)
	roleHandler := handler.NewRoleHandler(roleUseCase)

	userRepo := postgres.NewSQLUserRepository(dbPool)
	userUseCase := usecase.NewUserUsecase(userRepo, roleRepo, cfg)
	userHandler := &handler.UserHandler{UserUsecase: userUseCase}

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
	// Periode fiskal memakai journalUsecase sebagai pengelola jurnal penutup:
	// menutup periode menerbitkan jurnal penutup, membuka kembali membaliknya.
	periodUsecase := usecase.NewAccountingPeriodUsecase(periodRepo, journalUsecase)
	reportUsecase := usecase.NewAccountingReportUsecase(reportRepo, coaRepo)
	accountingHandler := handler.NewAccountingHandler(coaUsecase, periodUsecase, journalUsecase, reportUsecase)

	// GRN menerima journalUsecase sebagai auto-poster: penutupan penerimaan
	// menerbitkan jurnal otomatis sebelum stok dan status PO berubah.
	grnUsecase := usecase.NewGoodsReceiptNoteUsecase(grnRepo, poRepo, journalUsecase)
	grnHandler := handler.NewGoodsReceiptNoteHandler(grnUsecase)

	saleRepo := postgres.NewSQLSaleRepository(dbPool)
	memberRepo := postgres.NewSQLMemberRepository(dbPool)
	paymentMethodRepo := postgres.NewSQLPaymentMethodRepository(dbPool)
	appSettingsRepo := postgres.NewSQLSettingsRepository(dbPool)
	settingsUsecase := usecase.NewSettingsUsecase(appSettingsRepo)
	saleUsecase := usecase.NewSaleUsecase(saleRepo, memberRepo, paymentMethodRepo, journalUsecase, settingsUsecase)
	posUsecase := usecase.NewPOSUsecase(saleRepo, memberRepo, paymentMethodRepo, settingsUsecase)
	saleHandler := handler.NewSaleHandler(saleUsecase, posUsecase)
	memberUsecase := usecase.NewMemberUsecase(memberRepo)
	memberHandler := handler.NewMemberHandler(memberUsecase)
	paymentMethodUsecase := usecase.NewPaymentMethodUsecase(paymentMethodRepo)
	paymentMethodHandler := handler.NewPaymentMethodHandler(paymentMethodUsecase)
	settingsHandler := handler.NewSettingsHandler(settingsUsecase)

	handler.RegisterRoutes(cfg, userRepo, userHandler, roleHandler, regionHandler, supplierHandler, bankHandler, catHandler, productHandler, unitHandler, poHandler, lookupHandler, grnHandler, dashboardHandler, accountingHandler, saleHandler, memberHandler, paymentMethodHandler, settingsHandler)

	fmt.Printf("Server produksi siap melayani di http://localhost:%s\n", cfg.Port)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
