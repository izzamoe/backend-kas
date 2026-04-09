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
