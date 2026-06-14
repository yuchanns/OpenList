package service

import (
	"context"
	"strings"
	"testing"

	"github.com/OpenListTeam/OpenList/v4/internal/media/download"
	"github.com/OpenListTeam/OpenList/v4/internal/media/feed"
	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
	"github.com/OpenListTeam/OpenList/v4/internal/media/subscription"
)

func TestRefreshSourceParsesAndMatchesReleases(t *testing.T) {
	repo := &fakeRepository{
		subscriptions: []subscription.Rule{{
			ID:         10,
			Keyword:    "Example Movie",
			Include:    "2160p",
			Downloader: "qBittorrent",
			SavePath:   "/downloads/incoming",
		}},
	}
	svc := New(repo, nil)

	results, err := svc.RefreshSource(context.Background(), feed.Source{ID: 7, Kind: feed.SourceKindRSS}, []byte(testRSS("Example Movie 2024 2160p")))
	if err != nil {
		t.Fatalf("refresh source: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	result := results[0]
	if !result.Inserted || !result.Matched {
		t.Fatalf("expected inserted matched result, got %+v", result)
	}
	if result.ReleaseID != 1 || result.SubscriptionID != 10 || result.Reason != "matched" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(repo.marked) != 1 {
		t.Fatalf("expected one marked release, got %d", len(repo.marked))
	}
	if repo.marked[0].releaseID != 1 || repo.marked[0].match.SubscriptionID != 10 {
		t.Fatalf("unexpected marked release: %+v", repo.marked[0])
	}
}

func TestRefreshSourceLeavesUnmatchedReleasePending(t *testing.T) {
	repo := &fakeRepository{
		subscriptions: []subscription.Rule{{
			ID:      20,
			Keyword: "Different Movie",
		}},
	}
	svc := New(repo, nil)

	results, err := svc.RefreshSource(context.Background(), feed.Source{ID: 7, Kind: feed.SourceKindRSS}, []byte(testRSS("Example Movie 2024 2160p")))
	if err != nil {
		t.Fatalf("refresh source: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Matched {
		t.Fatalf("expected unmatched result, got %+v", results[0])
	}
	if results[0].Reason != "no_subscription_matched" {
		t.Fatalf("unexpected reason: %q", results[0].Reason)
	}
	if len(repo.marked) != 0 {
		t.Fatalf("expected no marked releases, got %d", len(repo.marked))
	}
}

func TestRefreshSourceStopsOnInvalidSubscriptionRegex(t *testing.T) {
	repo := &fakeRepository{
		subscriptions: []subscription.Rule{{
			ID:      30,
			Keyword: "Example Movie",
			Include: "[",
		}},
	}
	svc := New(repo, nil)

	_, err := svc.RefreshSource(context.Background(), feed.Source{ID: 7, Kind: feed.SourceKindRSS}, []byte(testRSS("Example Movie 2024 2160p")))
	if err == nil {
		t.Fatal("expected invalid regex error")
	}
	if !strings.Contains(err.Error(), "build subscription matcher") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.marked) != 0 {
		t.Fatalf("expected no marked releases, got %d", len(repo.marked))
	}
}

func TestDispatchDownloadCreatesDownloaderTaskAndReference(t *testing.T) {
	repo := &fakeRepository{}
	downloader := &fakeDownloader{taskID: "task-1"}
	svc := New(repo, downloader)

	result, err := svc.DispatchDownload(context.Background(), DispatchInput{
		ReleaseID:      100,
		SubscriptionID: 200,
		DownloadURL:    "magnet:?xt=urn:btih:abc",
		DownloadPath:   "/downloads/incoming",
		DownloaderKey:  "qBittorrent",
	})
	if err != nil {
		t.Fatalf("dispatch download: %v", err)
	}
	if result.TaskID != "task-1" {
		t.Fatalf("unexpected task id: %q", result.TaskID)
	}
	if len(downloader.requests) != 1 {
		t.Fatalf("expected one downloader request, got %d", len(downloader.requests))
	}
	request := downloader.requests[0]
	if request.ReleaseID != 100 || request.SubscriptionID != 200 {
		t.Fatalf("unexpected downloader request: %+v", request)
	}
	if request.DownloadPath != "/downloads/incoming" || request.DownloaderKey != "qBittorrent" {
		t.Fatalf("unexpected downloader target: %+v", request)
	}
	if len(repo.downloadRefs) != 1 {
		t.Fatalf("expected one download ref, got %d", len(repo.downloadRefs))
	}
	ref := repo.downloadRefs[0]
	if ref.ReleaseID != 100 || ref.SubscriptionID != 200 || ref.TaskID != "task-1" {
		t.Fatalf("unexpected download ref: %+v", ref)
	}
	if ref.Status != mediamodel.DownloadStatusCreated {
		t.Fatalf("unexpected download ref status: %q", ref.Status)
	}
}

func TestDispatchDownloadRejectsUnmatchedRelease(t *testing.T) {
	repo := &fakeRepository{}
	downloader := &fakeDownloader{taskID: "task-1"}
	svc := New(repo, downloader)

	_, err := svc.DispatchDownload(context.Background(), DispatchInput{
		ReleaseID:     100,
		DownloadURL:   "magnet:?xt=urn:btih:abc",
		DownloadPath:  "/downloads/incoming",
		DownloaderKey: "qBittorrent",
	})
	if err == nil {
		t.Fatal("expected unmatched release error")
	}
	if !strings.Contains(err.Error(), "subscription id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(downloader.requests) != 0 {
		t.Fatalf("expected no downloader requests, got %d", len(downloader.requests))
	}
	if len(repo.downloadRefs) != 0 {
		t.Fatalf("expected no download refs, got %d", len(repo.downloadRefs))
	}
}

type markedRelease struct {
	releaseID uint
	match     subscription.MatchResult
}

type fakeRepository struct {
	subscriptions []subscription.Rule
	upserted      []feed.Release
	marked        []markedRelease
	downloadRefs  []mediamodel.DownloadRef
}

func (r *fakeRepository) UpsertRelease(ctx context.Context, release feed.Release) (*mediamodel.Release, bool, error) {
	r.upserted = append(r.upserted, release)
	return &mediamodel.Release{
		ID:          uint(len(r.upserted)),
		SourceID:    release.SourceID,
		Fingerprint: release.Fingerprint(),
		Title:       release.Title,
		DownloadURL: release.DownloadURL,
		Status:      mediamodel.ReleaseStatusPending,
	}, true, nil
}

func (r *fakeRepository) ListEnabledSubscriptions(ctx context.Context, sourceID uint) ([]subscription.Rule, error) {
	return r.subscriptions, nil
}

func (r *fakeRepository) MarkReleaseMatched(ctx context.Context, releaseID uint, match subscription.MatchResult) error {
	r.marked = append(r.marked, markedRelease{releaseID: releaseID, match: match})
	return nil
}

func (r *fakeRepository) CreateDownloadRef(ctx context.Context, ref mediamodel.DownloadRef) (*mediamodel.DownloadRef, error) {
	ref.ID = uint(len(r.downloadRefs) + 1)
	r.downloadRefs = append(r.downloadRefs, ref)
	return &ref, nil
}

type fakeDownloader struct {
	taskID   string
	requests []download.Request
}

func (d *fakeDownloader) Add(ctx context.Context, req download.Request) (*download.TaskRef, error) {
	d.requests = append(d.requests, req)
	return &download.TaskRef{ID: d.taskID}, nil
}

func testRSS(title string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>` + title + `</title>
      <description>Example description</description>
      <link>https://example.test/item/1</link>
      <enclosure url="magnet:?xt=urn:btih:abc" length="12345" type="application/x-bittorrent" />
    </item>
  </channel>
</rss>`
}
