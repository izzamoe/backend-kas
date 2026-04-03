package main

import (
	"kas/internal/handler"
	"kas/internal/middleware"
	"kas/internal/repository"
	"kas/internal/service"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"

	// Enable migrations
	_ "kas/migrations"
)

func main() {
	app := pocketbase.New()

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
	requireFamily := middleware.RequireFamily(familyMemberRepo)

	// Service layer
	transactionService := service.NewTransactionService(transactionRepo)
	reportService := service.NewReportService(transactionRepo)

	// Handler layer
	transactionHandler := handler.NewTransactionHandler(transactionService)
	reportHandler := handler.NewReportHandler(reportService, requireFamily)

	// Register routes
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Register custom API routes
		transactionHandler.RegisterRoutes(se)
		reportHandler.RegisterRoutes(se)

		// Serves static files from the provided public dir (if exists)
		se.Router.GET("/{path...}", apis.Static(os.DirFS("./pb_public"), false))

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
