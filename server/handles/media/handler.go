package media

import (
	"context"

	"github.com/OpenListTeam/OpenList/v4/internal/db"
	"github.com/OpenListTeam/OpenList/v4/internal/media/download"
	"github.com/OpenListTeam/OpenList/v4/internal/media/feed"
	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
	mediarepo "github.com/OpenListTeam/OpenList/v4/internal/media/repository"
	mediaservice "github.com/OpenListTeam/OpenList/v4/internal/media/service"
)

type Repository interface {
	CreateFeedSource(ctx context.Context, source mediamodel.FeedSource) (*mediamodel.FeedSource, error)
	ListFeedSources(ctx context.Context) ([]mediamodel.FeedSource, error)
	GetFeedSource(ctx context.Context, id uint) (*mediamodel.FeedSource, error)
	CreateSubscription(ctx context.Context, subscription mediamodel.Subscription) (*mediamodel.Subscription, error)
	ListSubscriptions(ctx context.Context) ([]mediamodel.Subscription, error)
	ListReleases(ctx context.Context, filter mediarepo.ReleaseFilter) ([]mediamodel.Release, error)
	GetRelease(ctx context.Context, id uint) (*mediamodel.Release, error)
}

type Service interface {
	RefreshSource(ctx context.Context, source feed.Source, input []byte) ([]mediaservice.RefreshResult, error)
	DispatchDownload(ctx context.Context, input mediaservice.DispatchInput) (*mediaservice.DispatchResult, error)
}

type Fetcher interface {
	Fetch(ctx context.Context, source mediamodel.FeedSource) ([]byte, error)
}

type Handler struct {
	repository Repository
	service    Service
	fetcher    Fetcher
}

func NewHandler(repository Repository, service Service, fetcher Fetcher) *Handler {
	return &Handler{repository: repository, service: service, fetcher: fetcher}
}

func NewDefaultHandler() *Handler {
	repo := mediarepo.New(db.GetDb())
	svc := mediaservice.New(repo, download.NewOpenListToolDownloader())
	return NewHandler(repo, svc, HTTPFetcher{})
}

func feedSourceFromModel(source mediamodel.FeedSource) feed.Source {
	return feed.Source{
		ID:       source.ID,
		Name:     source.Name,
		URL:      source.URL,
		Kind:     feed.SourceKind(source.Kind),
		Enabled:  source.Enabled,
		ProxyURL: source.ProxyURL,
	}
}
