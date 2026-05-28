package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"kas/generated"
	"kas/internal/domain"

	"github.com/pocketbase/pocketbase/core"
)

// CategoryInfo holds the essential fields of a category record for validation purposes.
type CategoryInfo struct {
	ID        string
	FamilyID  string
	Name      string
	IsDefault bool
}

// CategoryRepository defines the data access contract for category records.
type CategoryRepository interface {
	GetByID(id string) (*CategoryInfo, error)
	SeedMasterCategories(app core.App, familyID string) error
}

// categoryRepo is the concrete PocketBase implementation of CategoryRepository.
type categoryRepo struct {
	app core.App
}

// NewCategoryRepository creates a new PocketBase-backed CategoryRepository.
func NewCategoryRepository(app core.App) CategoryRepository {
	return &categoryRepo{app: app}
}

// GetByID retrieves a category by its unique ID. Returns domain.ErrCategoryNotFound if the category is not found.
func (r *categoryRepo) GetByID(id string) (*CategoryInfo, error) {
	record, err := r.app.FindRecordById("categories", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to find category: %w", err)
	}

	proxy, err := generated.WrapRecord[generated.Categories](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap category record: %w", err)
	}

	return &CategoryInfo{
		ID:        proxy.Id,
		FamilyID:  proxy.Record.GetString("family_id"),
		Name:      proxy.Name(),
		IsDefault: proxy.IsDefault(),
	}, nil
}

func (r *categoryRepo) FindByFamilyNameAndType(familyID, name, txType string) (*CategoryInfo, error) {
	record, err := r.app.FindFirstRecordByFilter(
		"categories",
		"family_id = {:familyID} && name = {:name} && type = {:type}",
		map[string]any{"familyID": familyID, "name": name, "type": txType},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find category by family/name/type: %w", err)
	}

	proxy, err := generated.WrapRecord[generated.Categories](record)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap category record: %w", err)
	}

	return &CategoryInfo{
		ID:        proxy.Id,
		FamilyID:  proxy.Record.GetString("family_id"),
		Name:      proxy.Name(),
		IsDefault: proxy.IsDefault(),
	}, nil
}

// SeedMasterCategories copies all master categories into the given family, making per-family default categories.
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

		for _, master := range masterRecords {
			masterProxy, err := generated.WrapRecord[generated.Categories](master)
			if err != nil {
				return fmt.Errorf("failed to wrap master category: %w", err)
			}

			newProxy, err := generated.NewProxy[generated.Categories](txApp)
			if err != nil {
				return fmt.Errorf("failed to create category proxy: %w", err)
			}

			newProxy.Record.Set("family_id", familyID)
			newProxy.SetName(masterProxy.Name())
			newProxy.SetIcon(masterProxy.Icon())
			newProxy.SetColor(masterProxy.Color())
			newProxy.Record.Set("type", masterProxy.GetString("type"))
			newProxy.SetIsDefault(true)
			newProxy.SetIsMaster(false)
			if err := txApp.Save(newProxy.Record); err != nil {
				return fmt.Errorf("failed to save category %s: %w", masterProxy.Name(), err)
			}
		}
		return nil
	})
}
