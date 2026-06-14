package repository

import (
	"context"

	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
	"github.com/OpenListTeam/OpenList/v4/internal/media/subscription"
)

func (r *Repository) CreateSubscription(ctx context.Context, item mediamodel.Subscription) (*mediamodel.Subscription, error) {
	if err := r.db.WithContext(ctx).Create(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) ListEnabledSubscriptions(ctx context.Context, sourceID uint) ([]subscription.Rule, error) {
	var items []mediamodel.Subscription
	if err := r.db.WithContext(ctx).
		Where("enabled = ? AND (source_id = ? OR source_id = 0)", true, sourceID).
		Order("id asc").
		Find(&items).Error; err != nil {
		return nil, err
	}
	rules := make([]subscription.Rule, 0, len(items))
	for _, item := range items {
		rules = append(rules, subscription.Rule{
			ID:         item.ID,
			Name:       item.Name,
			Keyword:    item.Keyword,
			Include:    item.Include,
			Exclude:    item.Exclude,
			MinSize:    item.MinSize,
			MaxSize:    item.MaxSize,
			Downloader: item.Downloader,
			SavePath:   item.SavePath,
		})
	}
	return rules, nil
}
