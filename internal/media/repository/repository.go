package repository

import (
	"context"
	"errors"

	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Migrate(ctx context.Context) error {
	if r == nil || r.db == nil {
		return errors.New("media repository database is not configured")
	}
	return r.db.WithContext(ctx).AutoMigrate(mediamodel.Models()...)
}
