package repository

import (
	"context"

	"github.com/OpenListTeam/OpenList/v4/internal/media/feed"
	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
)

func (r *Repository) CreateFeedSource(ctx context.Context, source mediamodel.FeedSource) (*mediamodel.FeedSource, error) {
	if err := r.db.WithContext(ctx).Create(&source).Error; err != nil {
		return nil, err
	}
	return &source, nil
}

func (r *Repository) ListFeedSources(ctx context.Context) ([]mediamodel.FeedSource, error) {
	var sources []mediamodel.FeedSource
	if err := r.db.WithContext(ctx).Order("id asc").Find(&sources).Error; err != nil {
		return nil, err
	}
	return sources, nil
}

func (r *Repository) GetFeedSource(ctx context.Context, id uint) (*mediamodel.FeedSource, error) {
	var source mediamodel.FeedSource
	if err := r.db.WithContext(ctx).First(&source, id).Error; err != nil {
		return nil, err
	}
	return &source, nil
}

func (r *Repository) ListEnabledFeedSources(ctx context.Context) ([]feed.Source, error) {
	var sources []mediamodel.FeedSource
	if err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("id asc").
		Find(&sources).Error; err != nil {
		return nil, err
	}
	result := make([]feed.Source, 0, len(sources))
	for _, source := range sources {
		result = append(result, feed.Source{
			ID:       source.ID,
			Name:     source.Name,
			URL:      source.URL,
			Kind:     feed.SourceKind(source.Kind),
			Enabled:  source.Enabled,
			ProxyURL: source.ProxyURL,
		})
	}
	return result, nil
}
