package service

import (
	"context"

	"github.com/OpenListTeam/OpenList/v4/internal/media/download"
	"github.com/OpenListTeam/OpenList/v4/internal/media/feed"
	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
	"github.com/OpenListTeam/OpenList/v4/internal/media/subscription"
)

type Repository interface {
	UpsertRelease(ctx context.Context, release feed.Release) (*mediamodel.Release, bool, error)
	ListEnabledSubscriptions(ctx context.Context, sourceID uint) ([]subscription.Rule, error)
	MarkReleaseMatched(ctx context.Context, releaseID uint, match subscription.MatchResult) error
	CreateDownloadRef(ctx context.Context, ref mediamodel.DownloadRef) (*mediamodel.DownloadRef, error)
}

type Downloader interface {
	Add(ctx context.Context, req download.Request) (*download.TaskRef, error)
}

type Service struct {
	repository Repository
	downloader Downloader
}

func New(repository Repository, downloader Downloader) *Service {
	return &Service{repository: repository, downloader: downloader}
}
