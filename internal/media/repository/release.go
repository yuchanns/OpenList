package repository

import (
	"context"
	"errors"

	"github.com/OpenListTeam/OpenList/v4/internal/media/feed"
	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
	"github.com/OpenListTeam/OpenList/v4/internal/media/subscription"
	"gorm.io/gorm"
)

type ReleaseFilter struct {
	Status string
	Limit  int
}

func (r *Repository) UpsertRelease(ctx context.Context, release feed.Release) (*mediamodel.Release, bool, error) {
	fingerprint := release.Fingerprint()
	var existing mediamodel.Release
	err := r.db.WithContext(ctx).Where("fingerprint = ?", fingerprint).First(&existing).Error
	if err == nil {
		updates := map[string]interface{}{
			"source_id":       release.SourceID,
			"title":           release.Title,
			"description":     release.Description,
			"link":            release.Link,
			"download_url":    release.DownloadURL,
			"published_at":    release.PublishedAt,
			"size":            release.Size,
			"fingerprint":     fingerprint,
			"match_reason":    existing.MatchReason,
			"downloader":      existing.Downloader,
			"save_path":       existing.SavePath,
			"status":          existing.Status,
			"subscription_id": existing.SubscriptionID,
		}
		if err := r.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			return nil, false, err
		}
		if err := r.db.WithContext(ctx).First(&existing, existing.ID).Error; err != nil {
			return nil, false, err
		}
		return &existing, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	item := mediamodel.Release{
		SourceID:    release.SourceID,
		Fingerprint: fingerprint,
		Title:       release.Title,
		Description: release.Description,
		Link:        release.Link,
		DownloadURL: release.DownloadURL,
		PublishedAt: release.PublishedAt,
		Size:        release.Size,
		Status:      mediamodel.ReleaseStatusPending,
	}
	if err := r.db.WithContext(ctx).Create(&item).Error; err != nil {
		return nil, false, err
	}
	return &item, true, nil
}

func (r *Repository) GetRelease(ctx context.Context, id uint) (*mediamodel.Release, error) {
	var release mediamodel.Release
	if err := r.db.WithContext(ctx).First(&release, id).Error; err != nil {
		return nil, err
	}
	return &release, nil
}

func (r *Repository) ListReleases(ctx context.Context, filter ReleaseFilter) ([]mediamodel.Release, error) {
	query := r.db.WithContext(ctx).Model(&mediamodel.Release{})
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	var releases []mediamodel.Release
	if err := query.Order("id desc").Find(&releases).Error; err != nil {
		return nil, err
	}
	return releases, nil
}

func (r *Repository) MarkReleaseMatched(ctx context.Context, releaseID uint, match subscription.MatchResult) error {
	updates := map[string]interface{}{
		"subscription_id": match.SubscriptionID,
		"status":          mediamodel.ReleaseStatusMatched,
		"match_reason":    match.Reason,
		"downloader":      match.Downloader,
		"save_path":       match.SavePath,
	}
	return r.db.WithContext(ctx).Model(&mediamodel.Release{}).Where("id = ?", releaseID).Updates(updates).Error
}
