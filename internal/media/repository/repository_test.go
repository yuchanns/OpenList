package repository_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/OpenListTeam/OpenList/v4/internal/media/feed"
	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
	"github.com/OpenListTeam/OpenList/v4/internal/media/repository"
	"github.com/OpenListTeam/OpenList/v4/internal/media/subscription"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRepositoryMigratesMediaTables(t *testing.T) {
	repo, db := newRepository(t)
	if err := repo.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	for _, table := range []string{
		"media_feed_sources",
		"media_subscriptions",
		"media_releases",
		"media_download_refs",
	} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s to exist", table)
		}
	}
}

func TestRepositoryListsEnabledSourcesAndSubscriptions(t *testing.T) {
	ctx := context.Background()
	repo, _ := newMigratedRepository(t)

	source, err := repo.CreateFeedSource(ctx, mediamodel.FeedSource{
		Name:    "Custom RSS",
		URL:     "https://example.test/rss",
		Kind:    string(feed.SourceKindRSS),
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("create enabled source: %v", err)
	}
	if _, err := repo.CreateFeedSource(ctx, mediamodel.FeedSource{
		Name:    "Disabled RSS",
		URL:     "https://example.test/disabled",
		Kind:    string(feed.SourceKindRSS),
		Enabled: false,
	}); err != nil {
		t.Fatalf("create disabled source: %v", err)
	}
	if _, err := repo.CreateSubscription(ctx, mediamodel.Subscription{
		Name:       "Enabled Subscription",
		Enabled:    true,
		SourceID:   source.ID,
		Keyword:    "Example",
		Include:    "2160p",
		Downloader: "qBittorrent",
		SavePath:   "/downloads/incoming",
	}); err != nil {
		t.Fatalf("create enabled subscription: %v", err)
	}
	if _, err := repo.CreateSubscription(ctx, mediamodel.Subscription{
		Name:     "Disabled Subscription",
		Enabled:  false,
		SourceID: source.ID,
		Keyword:  "Ignored",
	}); err != nil {
		t.Fatalf("create disabled subscription: %v", err)
	}

	sources, err := repo.ListEnabledFeedSources(ctx)
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 enabled source, got %d", len(sources))
	}
	if sources[0].ID != source.ID || sources[0].Kind != feed.SourceKindRSS {
		t.Fatalf("unexpected source: %+v", sources[0])
	}

	rules, err := repo.ListEnabledSubscriptions(ctx, source.ID)
	if err != nil {
		t.Fatalf("list subscriptions: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 enabled subscription, got %d", len(rules))
	}
	if rules[0].Keyword != "Example" || rules[0].Downloader != "qBittorrent" {
		t.Fatalf("unexpected subscription rule: %+v", rules[0])
	}
}

func TestRepositoryUpsertsReleaseByFingerprint(t *testing.T) {
	ctx := context.Background()
	repo, _ := newMigratedRepository(t)
	input := feed.Release{
		SourceID:    10,
		Title:       "Example Movie 2024 2160p",
		Description: "First description",
		Link:        "https://example.test/item/1",
		DownloadURL: "magnet:?xt=urn:btih:abc",
		Size:        123,
	}

	first, inserted, err := repo.UpsertRelease(ctx, input)
	if err != nil {
		t.Fatalf("upsert first release: %v", err)
	}
	if !inserted {
		t.Fatal("expected first upsert to insert")
	}

	input.Description = "Updated description"
	second, inserted, err := repo.UpsertRelease(ctx, input)
	if err != nil {
		t.Fatalf("upsert second release: %v", err)
	}
	if inserted {
		t.Fatal("expected second upsert to update")
	}
	if second.ID != first.ID {
		t.Fatalf("expected same release id, got %d and %d", first.ID, second.ID)
	}
	if second.Description != "Updated description" {
		t.Fatalf("expected updated description, got %q", second.Description)
	}
}

func TestRepositoryMarksReleaseMatched(t *testing.T) {
	ctx := context.Background()
	repo, _ := newMigratedRepository(t)
	release, _, err := repo.UpsertRelease(ctx, feed.Release{
		SourceID:    11,
		Title:       "Example Movie 2024 2160p",
		DownloadURL: "magnet:?xt=urn:btih:def",
	})
	if err != nil {
		t.Fatalf("upsert release: %v", err)
	}

	err = repo.MarkReleaseMatched(ctx, release.ID, subscription.MatchResult{
		Matched:        true,
		Reason:         "matched",
		SubscriptionID: 99,
		Downloader:     "qBittorrent",
		SavePath:       "/downloads/incoming",
	})
	if err != nil {
		t.Fatalf("mark release matched: %v", err)
	}

	updated, err := repo.GetRelease(ctx, release.ID)
	if err != nil {
		t.Fatalf("get release: %v", err)
	}
	if updated.Status != mediamodel.ReleaseStatusMatched {
		t.Fatalf("expected matched status, got %q", updated.Status)
	}
	if updated.SubscriptionID == nil || *updated.SubscriptionID != 99 {
		t.Fatalf("unexpected subscription id: %v", updated.SubscriptionID)
	}
	if updated.MatchReason != "matched" || updated.Downloader != "qBittorrent" || updated.SavePath != "/downloads/incoming" {
		t.Fatalf("unexpected matched release: %+v", updated)
	}
}

func TestRepositoryCreatesDownloadRef(t *testing.T) {
	ctx := context.Background()
	repo, _ := newMigratedRepository(t)

	ref, err := repo.CreateDownloadRef(ctx, mediamodel.DownloadRef{
		ReleaseID:      20,
		SubscriptionID: 30,
		TaskID:         "task-1",
		DownloadPath:   "/downloads/incoming",
		DownloaderKey:  "qBittorrent",
		Status:         mediamodel.DownloadStatusCreated,
	})
	if err != nil {
		t.Fatalf("create download ref: %v", err)
	}
	if ref.ID == 0 {
		t.Fatal("expected download ref id")
	}
	if ref.Status != mediamodel.DownloadStatusCreated {
		t.Fatalf("unexpected status: %q", ref.Status)
	}
}

func newMigratedRepository(t *testing.T) (*repository.Repository, *gorm.DB) {
	t.Helper()
	repo, db := newRepository(t)
	if err := repo.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return repo, db
}

func newRepository(t *testing.T) (*repository.Repository, *gorm.DB) {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return repository.New(db), db
}
