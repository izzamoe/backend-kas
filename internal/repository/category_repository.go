package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"kas/generated"

	"github.com/pocketbase/pocketbase/core"
)

type CategoryInfo struct {
	ID        string
	FamilyID  string
	Name      string
	IsDefault bool
}

type CategoryRepository interface {
	GetByID(id string) (*CategoryInfo, error)
	SeedMasterCategories(app core.App, familyID string) error
}

type categoryRepo struct {
	app core.App
}

func NewCategoryRepository(app core.App) CategoryRepository {
	return &categoryRepo{app: app}
}

func (r *categoryRepo) GetByID(id string) (*CategoryInfo, error) {
	record, err := r.app.FindRecordById("categories", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find category: %w", err)
	}

	proxy, err := generated.WrapRecord[generated.Categories](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap category record: %w", err)
	}

	return &CategoryInfo{
		ID:        proxy.Id,
		FamilyID:  record.GetString("family_id"),
		Name:      proxy.Name(),
		IsDefault: proxy.IsDefault(),
	}, nil
}

func (r *categoryRepo) SeedMasterCategories(app core.App, familyID string) error {
	return app.RunInTransaction(func(txApp core.App) error {
		masterRecords, err := txApp.FindRecordsByFilter(
			"categories",
			"is_master = true && family_id = ''",
			"type,name",
			-1,
			0,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to find master categories: %w", err)
		}
		if len(masterRecords) == 0 {
			return fmt.Errorf("no master categories found in database")
		}

		collection, err := txApp.FindCachedCollectionByNameOrId("categories")
		if err != nil {
			return fmt.Errorf("failed to find categories collection: %w", err)
		}

		for _, master := range masterRecords {
			record := core.NewRecord(collection)
			record.Set("family_id", familyID)
			record.Set("name", master.GetString("name"))
			record.Set("icon", master.GetString("icon"))
			record.Set("color", master.GetString("color"))
			record.Set("type", master.GetString("type"))
			record.Set("is_default", true)
			record.Set("is_master", false)
			if err := txApp.Save(record); err != nil {
				return fmt.Errorf("failed to save category %s: %w", master.GetString("name"), err)
			}
		}
		return nil
	})
}
