package repository

import (
	"context"

	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
)

func (r *Repository) CreateDownloadRef(ctx context.Context, ref mediamodel.DownloadRef) (*mediamodel.DownloadRef, error) {
	if ref.Status == "" {
		ref.Status = mediamodel.DownloadStatusCreated
	}
	if err := r.db.WithContext(ctx).Create(&ref).Error; err != nil {
		return nil, err
	}
	return &ref, nil
}
