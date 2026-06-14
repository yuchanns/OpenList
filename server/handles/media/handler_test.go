package media

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenListTeam/OpenList/v4/internal/media/feed"
	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
	mediarepo "github.com/OpenListTeam/OpenList/v4/internal/media/repository"
	mediaservice "github.com/OpenListTeam/OpenList/v4/internal/media/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestCreateAndListFeeds(t *testing.T) {
	fakes := newFakes()
	router := newTestRouter(fakes)

	resp := performJSON(router, http.MethodPost, "/feeds", map[string]interface{}{
		"name":    "Custom RSS",
		"url":     "https://example.test/rss.xml",
		"kind":    "rss",
		"enabled": true,
	})
	assertCode(t, resp, http.StatusOK)

	resp = performJSON(router, http.MethodGet, "/feeds", nil)
	assertCode(t, resp, http.StatusOK)
	var body responseEnvelope[[]mediamodel.FeedSource]
	decodeResponse(t, resp, &body)
	if len(body.Data) != 1 {
		t.Fatalf("expected 1 feed, got %d", len(body.Data))
	}
	if body.Data[0].Name != "Custom RSS" || body.Data[0].Kind != "rss" {
		t.Fatalf("unexpected feed: %+v", body.Data[0])
	}
}

func TestCreateAndListSubscriptions(t *testing.T) {
	fakes := newFakes()
	router := newTestRouter(fakes)

	resp := performJSON(router, http.MethodPost, "/subscriptions", map[string]interface{}{
		"name":       "Example Movie",
		"enabled":    true,
		"source_id":  1,
		"keyword":    "Example Movie",
		"include":    "2160p",
		"downloader": "qBittorrent",
		"save_path":  "/downloads/incoming",
	})
	assertCode(t, resp, http.StatusOK)

	resp = performJSON(router, http.MethodGet, "/subscriptions", nil)
	assertCode(t, resp, http.StatusOK)
	var body responseEnvelope[[]mediamodel.Subscription]
	decodeResponse(t, resp, &body)
	if len(body.Data) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(body.Data))
	}
	if body.Data[0].Keyword != "Example Movie" || body.Data[0].Downloader != "qBittorrent" {
		t.Fatalf("unexpected subscription: %+v", body.Data[0])
	}
}

func TestRefreshFeedFetchesContentAndCallsService(t *testing.T) {
	fakes := newFakes()
	fakes.repository.feeds = []mediamodel.FeedSource{{
		ID:      7,
		Name:    "Custom RSS",
		URL:     "https://example.test/rss.xml",
		Kind:    "rss",
		Enabled: true,
	}}
	fakes.fetcher.content = []byte("<rss></rss>")
	fakes.service.refreshResults = []mediaservice.RefreshResult{{
		ReleaseID:      1,
		Inserted:       true,
		Matched:        true,
		Reason:         "matched",
		SubscriptionID: 10,
	}}
	router := newTestRouter(fakes)

	resp := performJSON(router, http.MethodPost, "/feeds/7/refresh", nil)
	assertCode(t, resp, http.StatusOK)
	if fakes.fetcher.lastURL != "https://example.test/rss.xml" {
		t.Fatalf("unexpected fetched url: %q", fakes.fetcher.lastURL)
	}
	if fakes.service.refreshedSource.ID != 7 || string(fakes.service.refreshedInput) != "<rss></rss>" {
		t.Fatalf("unexpected refresh call: %+v %q", fakes.service.refreshedSource, fakes.service.refreshedInput)
	}
}

func TestListReleases(t *testing.T) {
	fakes := newFakes()
	fakes.repository.releases = []mediamodel.Release{{
		ID:          11,
		Title:       "Example Movie",
		Status:      mediamodel.ReleaseStatusMatched,
		DownloadURL: "magnet:?xt=urn:btih:abc",
	}}
	router := newTestRouter(fakes)

	resp := performJSON(router, http.MethodGet, "/releases?status=matched", nil)
	assertCode(t, resp, http.StatusOK)
	if fakes.repository.lastReleaseFilter.Status != mediamodel.ReleaseStatusMatched {
		t.Fatalf("unexpected release filter: %+v", fakes.repository.lastReleaseFilter)
	}
	var body responseEnvelope[[]mediamodel.Release]
	decodeResponse(t, resp, &body)
	if len(body.Data) != 1 || body.Data[0].ID != 11 {
		t.Fatalf("unexpected releases: %+v", body.Data)
	}
}

func TestDownloadMatchedRelease(t *testing.T) {
	subscriptionID := uint(30)
	fakes := newFakes()
	fakes.repository.releases = []mediamodel.Release{{
		ID:             20,
		SubscriptionID: &subscriptionID,
		Title:          "Example Movie",
		DownloadURL:    "magnet:?xt=urn:btih:abc",
		Downloader:     "qBittorrent",
		SavePath:       "/downloads/incoming",
		Status:         mediamodel.ReleaseStatusMatched,
	}}
	fakes.service.dispatchResult = &mediaservice.DispatchResult{TaskID: "task-1"}
	router := newTestRouter(fakes)

	resp := performJSON(router, http.MethodPost, "/releases/20/download", nil)
	assertCode(t, resp, http.StatusOK)
	if fakes.service.dispatchedInput.ReleaseID != 20 || fakes.service.dispatchedInput.SubscriptionID != 30 {
		t.Fatalf("unexpected dispatch input: %+v", fakes.service.dispatchedInput)
	}
	if fakes.service.dispatchedInput.DownloaderKey != "qBittorrent" {
		t.Fatalf("unexpected downloader key: %q", fakes.service.dispatchedInput.DownloaderKey)
	}
}

type responseEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type fakeBundle struct {
	repository *fakeRepository
	service    *fakeService
	fetcher    *fakeFetcher
}

func newFakes() *fakeBundle {
	return &fakeBundle{
		repository: &fakeRepository{},
		service:    &fakeService{},
		fetcher:    &fakeFetcher{},
	}
}

func newTestRouter(fakes *fakeBundle) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group(""), NewHandler(fakes.repository, fakes.service, fakes.fetcher))
	return router
}

func performJSON(router *gin.Engine, method string, path string, payload interface{}) *httptest.ResponseRecorder {
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			panic(err)
		}
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertCode(t *testing.T, resp *httptest.ResponseRecorder, code int) {
	t.Helper()
	if resp.Code != http.StatusOK {
		t.Fatalf("expected http 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	var body responseEnvelope[json.RawMessage]
	decodeResponse(t, resp, &body)
	if body.Code != code {
		t.Fatalf("expected response code %d, got %d body=%s", code, body.Code, resp.Body.String())
	}
}

func decodeResponse[T any](t *testing.T, resp *httptest.ResponseRecorder, output *T) {
	t.Helper()
	if err := json.Unmarshal(resp.Body.Bytes(), output); err != nil {
		t.Fatalf("decode response %s: %v", resp.Body.String(), err)
	}
}

type fakeRepository struct {
	feeds             []mediamodel.FeedSource
	subscriptions     []mediamodel.Subscription
	releases          []mediamodel.Release
	lastReleaseFilter mediarepo.ReleaseFilter
}

func (r *fakeRepository) CreateFeedSource(ctx context.Context, source mediamodel.FeedSource) (*mediamodel.FeedSource, error) {
	source.ID = uint(len(r.feeds) + 1)
	r.feeds = append(r.feeds, source)
	return &source, nil
}

func (r *fakeRepository) ListFeedSources(ctx context.Context) ([]mediamodel.FeedSource, error) {
	return r.feeds, nil
}

func (r *fakeRepository) GetFeedSource(ctx context.Context, id uint) (*mediamodel.FeedSource, error) {
	for _, source := range r.feeds {
		if source.ID == id {
			return &source, nil
		}
	}
	return nil, gormRecordNotFound()
}

func (r *fakeRepository) CreateSubscription(ctx context.Context, subscription mediamodel.Subscription) (*mediamodel.Subscription, error) {
	subscription.ID = uint(len(r.subscriptions) + 1)
	r.subscriptions = append(r.subscriptions, subscription)
	return &subscription, nil
}

func (r *fakeRepository) ListSubscriptions(ctx context.Context) ([]mediamodel.Subscription, error) {
	return r.subscriptions, nil
}

func (r *fakeRepository) ListReleases(ctx context.Context, filter mediarepo.ReleaseFilter) ([]mediamodel.Release, error) {
	r.lastReleaseFilter = filter
	return r.releases, nil
}

func (r *fakeRepository) GetRelease(ctx context.Context, id uint) (*mediamodel.Release, error) {
	for _, release := range r.releases {
		if release.ID == id {
			return &release, nil
		}
	}
	return nil, gormRecordNotFound()
}

type fakeService struct {
	refreshResults  []mediaservice.RefreshResult
	refreshedSource feed.Source
	refreshedInput  []byte
	dispatchResult  *mediaservice.DispatchResult
	dispatchedInput mediaservice.DispatchInput
}

func (s *fakeService) RefreshSource(ctx context.Context, source feed.Source, input []byte) ([]mediaservice.RefreshResult, error) {
	s.refreshedSource = source
	s.refreshedInput = input
	return s.refreshResults, nil
}

func (s *fakeService) DispatchDownload(ctx context.Context, input mediaservice.DispatchInput) (*mediaservice.DispatchResult, error) {
	s.dispatchedInput = input
	return s.dispatchResult, nil
}

type fakeFetcher struct {
	content []byte
	lastURL string
}

func (f *fakeFetcher) Fetch(ctx context.Context, source mediamodel.FeedSource) ([]byte, error) {
	f.lastURL = source.URL
	return f.content, nil
}

func gormRecordNotFound() error {
	return gorm.ErrRecordNotFound
}
