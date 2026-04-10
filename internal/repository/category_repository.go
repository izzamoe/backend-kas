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

type masterCategorySeed struct {
	Name  string
	Icon  string
	Color string
	Type  string
}

var masterCategories = []masterCategorySeed{
	{Name: "Makanan & Minuman", Icon: "🍔", Color: "#FF6B6B", Type: "expense"},
	{Name: "Transportasi", Icon: "🚗", Color: "#4ECDC4", Type: "expense"},
	{Name: "Belanja", Icon: "🛒", Color: "#45B7D1", Type: "expense"},
	{Name: "Tagihan & Utilitas", Icon: "💡", Color: "#96CEB4", Type: "expense"},
	{Name: "Kesehatan", Icon: "🏥", Color: "#FFEAA7", Type: "expense"},
	{Name: "Pendidikan", Icon: "📚", Color: "#DDA0DD", Type: "expense"},
	{Name: "Hiburan", Icon: "🎮", Color: "#98D8C8", Type: "expense"},
	{Name: "Lainnya", Icon: "📦", Color: "#B0B0B0", Type: "expense"},
	{Name: "Gaji", Icon: "💰", Color: "#2ECC71", Type: "income"},
	{Name: "Bonus", Icon: "🎁", Color: "#F39C12", Type: "income"},
	{Name: "Investasi", Icon: "📈", Color: "#3498DB", Type: "income"},
	{Name: "Lainnya", Icon: "💵", Color: "#95A5A6", Type: "income"},
}

func (r *categoryRepo) SeedMasterCategories(app core.App, familyID string) error {
	return app.RunInTransaction(func(txApp core.App) error {
		collection, err := txApp.FindCachedCollectionByNameOrId("categories")
		if err != nil {
			return fmt.Errorf("failed to find categories collection: %w", err)
		}
		for _, cat := range masterCategories {
			record := core.NewRecord(collection)
			record.Set("family_id", familyID)
			record.Set("name", cat.Name)
			record.Set("icon", cat.Icon)
			record.Set("color", cat.Color)
			record.Set("type", cat.Type)
			record.Set("is_default", true)
			record.Set("is_master", true)
			if err := txApp.Save(record); err != nil {
				return fmt.Errorf("failed to save category %s: %w", cat.Name, err)
			}
		}
		return nil
	})
}
