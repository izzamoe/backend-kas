package migrations

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Find the superusers collection
		superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
		if err != nil {
			return err
		}

		// Create new superuser record
		record := core.NewRecord(superusers)

		// Get email and password from environment variables or use defaults
		email := os.Getenv("ADMIN_EMAIL")
		if email == "" {
			email = "admin@example.com"
			log.Println("Warning: Using default admin email. Set ADMIN_EMAIL env var for production.")
		}

		password := os.Getenv("ADMIN_PASSWORD")
		if password == "" {
			password = "admin123456"
			log.Println("Warning: Using default admin password. Set ADMIN_PASSWORD env var for production.")
		}

		record.Set("email", email)
		record.Set("password", password)

		// Save the superuser record
		if err := app.Save(record); err != nil {
			return err
		}

		log.Printf("✓ Initial superuser created: %s", email)
		return nil
	}, func(app core.App) error {
		// Revert operation: delete the created superuser
		email := os.Getenv("ADMIN_EMAIL")
		if email == "" {
			email = "admin@example.com"
		}

		record, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, email)
		if err != nil || record == nil {
			// Superuser might already be deleted or doesn't exist
			return nil
		}

		return app.Delete(record)
	})
}
