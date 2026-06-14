package service

import (
	"context"
	"strings"
	"testing"

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
