package main

import (
	"embed"
	"io/fs"
	"log"
	"os"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"

	"kas/generated"
	"kas/internal/handler"
	"kas/internal/hooks"
	"kas/internal/middleware"
	"kas/internal/repository"
	"kas/internal/scheduler"
	"kas/internal/service"

	// Enable migrations
	_ "kas/migrations"
)

//go:embed pb_public
var publicFiles embed.FS

// @title Uang Kas Keluarga API
// @version 1.0.0
// @description API for managing family finances, including transactions, reports, and Digiflazz integration.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() { //nolint:funlen // Dependency injection wiring requires all components
	publicFS, err := fs.Sub(publicFiles, "pb_public")
	if err != nil {
		log.Fatal(err)
	}

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DBConnect: func(dbPath string) (*dbx.DB, error) {
			// PocketBase defaults + mmap_size for memory-mapped reads
			pragmas := "?_pragma=busy_timeout(10000)" +
				"&_pragma=journal_mode(WAL)" +
				"&_pragma=journal_size_limit(200000000)" +
				"&_pragma=synchronous(NORMAL)" +
				"&_pragma=foreign_keys(ON)" +
				"&_pragma=temp_store(MEMORY)" +
				"&_pragma=cache_size(-32000)" +
				"&_pragma=mmap_size(268435456)" // 256MB memory-mapped I/O
			return dbx.Open("sqlite", dbPath+pragmas)
		},
	})

	// Register migrate command with auto-migration enabled during development
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// Enable auto creation of migration files when making collection changes in Dashboard
		// (only enabled during development with "go run")
		Automigrate: osutils.IsProbablyGoRun(),
	})

	// Dependency Injection - Wire up layers
	// Repository layer
	transactionRepo := repository.NewTransactionRepository(app)
	familyMemberRepo := repository.NewFamilyMemberRepository(app)
	categoryRepo := repository.NewCategoryRepository(app)
	familyRepo := repository.NewFamilyRepository(app)
	digiflazzCredentialRepo := repository.NewDigiflazzCredentialRepository(app)
	digiflazzProductRepo := repository.NewDigiflazzProductRepository(app)
	digiflazzOrderRepo := repository.NewDigiflazzOrderRepository(app)
	digiflazzEventRepo := repository.NewDigiflazzEventRepository(app)
	requireFamily := middleware.RequireFamily(familyMemberRepo)
	requireFamilyOwner := middleware.RequireFamilyOwner()

	// Service layer
	transactionService := service.NewTransactionService(transactionRepo, categoryRepo)
	reportService := service.NewReportService(transactionRepo)
	familyService := service.NewFamilyService(familyRepo, familyMemberRepo, app, middleware.InvalidateFamily)
	digiflazzProductService := service.NewDigiflazzProductService(app, digiflazzProductRepo, digiflazzCredentialRepo, nil)
	digiflazzCredentialService := service.NewDigiflazzCredentialService(digiflazzCredentialRepo, app, nil, digiflazzProductRepo, digiflazzProductService)
	digiflazzOrderService := service.NewDigiflazzOrderService(digiflazzOrderRepo, service.DigiflazzOrderServiceDeps{
		App:             app,
		CredentialRepo:  digiflazzCredentialRepo,
		ProductService:  digiflazzProductService,
		EventRepo:       digiflazzEventRepo,
		TransactionRepo: transactionRepo,
		CategoryRepo:    categoryRepo,
	})
	digiflazzCronService := service.NewDigiflazzCronService(app, digiflazzProductService, digiflazzOrderService, digiflazzCredentialRepo, digiflazzOrderRepo, digiflazzEventRepo)

	// Handler layer
	transactionHandler := handler.NewTransactionHandler(transactionService, middleware.RequireAuth, requireFamily)
	reportHandler := handler.NewReportHandler(reportService, familyMemberRepo, middleware.RequireAuth, requireFamily)
	familyHandler := handler.NewFamilyHandler(familyService, middleware.RequireAuth)
	digiflazzCredentialHandler := handler.NewDigiflazzCredentialHandler(digiflazzCredentialService, middleware.RequireAuth, requireFamily, requireFamilyOwner)
	digiflazzProductHandler := handler.NewDigiflazzProductHandler(digiflazzProductService, middleware.RequireAuth, requireFamily, requireFamilyOwner)
	digiflazzOrderHandler := handler.NewDigiflazzOrderHandler(digiflazzOrderService, middleware.RequireAuth, requireFamily)
	digiflazzWebhookHandler := handler.NewDigiflazzWebhookHandler(digiflazzCredentialRepo, digiflazzOrderRepo, digiflazzEventRepo, digiflazzOrderService)

	// Register routes
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Register custom API routes
		transactionHandler.RegisterRoutes(se)
		reportHandler.RegisterRoutes(se)
		familyHandler.RegisterRoutes(se)
		digiflazzCredentialHandler.RegisterRoutes(se)
		digiflazzProductHandler.RegisterRoutes(se)
		digiflazzOrderHandler.RegisterRoutes(se)
		digiflazzWebhookHandler.RegisterRoutes(se)

		if _, err := os.Stat("./files_public"); err == nil {
			se.Router.GET("/files/{path...}", apis.Static(os.DirFS("./files_public"), false))
		}

		// Serves embedded static files from pb_public.
		se.Router.GET("/{path...}", apis.Static(publicFS, false))

		return se.Next()
	})

	ph := generated.NewProxyHooks(app)

	hooks.RegisterFamilyHooks(ph, categoryRepo)
	hooks.RegisterTransactionHooks(ph)

	scheduler.Register(app, digiflazzCronService)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
