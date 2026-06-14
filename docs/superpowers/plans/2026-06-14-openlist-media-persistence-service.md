# OpenList Media Persistence Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the second backend milestone for OpenList Media: persistent media models, repository operations, feed refresh matching, and download dispatch orchestration.

**Architecture:** Keep media logic isolated under `internal/media/...`. The repository owns GORM persistence and exposes domain-oriented methods; the service layer composes feed parsing, subscription matching, and the existing downloader boundary. This milestone does not modify OpenList core files, does not add HTTP routes, and does not start background schedulers.

**Tech Stack:** Go 1.24, GORM, `github.com/glebarez/sqlite` for repository tests, standard `testing`, existing `internal/media/feed`, `internal/media/subscription`, and `internal/media/download` packages.

---

## Low-Intrusion Boundaries

本阶段新增文件只允许落在 `internal/media/...`。OpenList 原模块触点为：无。

后续 API 或调度器阶段如果需要接入 OpenList core，必须在新计划中单独列出触点，并优先选择 route registration 或 service bootstrap 这类薄入口。

## File Structure

- Create `internal/media/model/models.go`: GORM models and `Models()` migration list.
- Create `internal/media/model/status.go`: release and download status constants.
- Create `internal/media/model/models_test.go`: migration and table-name tests.
- Create `internal/media/repository/repository.go`: GORM repository constructor and migration.
- Create `internal/media/repository/feed.go`: feed source persistence.
- Create `internal/media/repository/subscription.go`: subscription persistence and domain conversion.
- Create `internal/media/repository/release.go`: release upsert, match state, lookup.
- Create `internal/media/repository/download.go`: download reference persistence.
- Create `internal/media/repository/repository_test.go`: repository integration tests with in-memory SQLite.
- Create `internal/media/service/service.go`: service dependencies and constructor.
- Create `internal/media/service/feed_refresh.go`: parse, upsert, and match feed releases.
- Create `internal/media/service/download_dispatch.go`: dispatch matched releases to downloader.
- Create `internal/media/service/service_test.go`: service tests with fakes.

## Verification Commands

Use Homebrew Go explicitly to avoid the older `/usr/local/go` toolchain path:

```bash
/opt/homebrew/bin/go test ./internal/media/... -count=1
```

Expected after all tasks:

```text
ok  	github.com/OpenListTeam/OpenList/v4/internal/media/download
ok  	github.com/OpenListTeam/OpenList/v4/internal/media/feed
ok  	github.com/OpenListTeam/OpenList/v4/internal/media/model
ok  	github.com/OpenListTeam/OpenList/v4/internal/media/repository
ok  	github.com/OpenListTeam/OpenList/v4/internal/media/service
ok  	github.com/OpenListTeam/OpenList/v4/internal/media/subscription
```

---

### Task 1: Media Models And Migration Helper

**Files:**
- Create: `internal/media/model/models_test.go`
- Create: `internal/media/model/status.go`
- Create: `internal/media/model/models.go`

- [ ] **Step 1: Write failing model tests**

Create `internal/media/model/models_test.go`:

```go
package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestModelsMigrateExpectedTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	if err := db.AutoMigrate(Models()...); err != nil {
		t.Fatalf("migrate media models: %v", err)
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

func TestReleaseFingerprintIsUnique(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(Models()...); err != nil {
		t.Fatalf("migrate media models: %v", err)
	}

	first := Release{SourceID: 1, Fingerprint: "same", Title: "First", Status: ReleaseStatusPending}
	second := Release{SourceID: 1, Fingerprint: "same", Title: "Second", Status: ReleaseStatusPending}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first release: %v", err)
	}
	if err := db.Create(&second).Error; err == nil {
		t.Fatal("expected duplicate fingerprint to fail")
	}
}
```

- [ ] **Step 2: Run model tests to verify they fail**

```bash
/opt/homebrew/bin/go test ./internal/media/model -count=1
```

Expected: FAIL because `internal/media/model` does not exist.

- [ ] **Step 3: Implement statuses and models**

Create `internal/media/model/status.go` with these constants: `ReleaseStatusPending`, `ReleaseStatusMatched`, `ReleaseStatusIgnored`, `DownloadStatusCreated`, and `DownloadStatusFailed`.

Create `internal/media/model/models.go` with these structs and explicit table names:

- `FeedSource` -> `media_feed_sources`
- `Subscription` -> `media_subscriptions`
- `Release` -> `media_releases`
- `DownloadRef` -> `media_download_refs`

`Release.Fingerprint` must use a unique index. `Models() []interface{}` must return pointers to all four structs.

- [ ] **Step 4: Run model tests**

```bash
/opt/homebrew/bin/go test ./internal/media/model -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/media/model
git commit -m "feat(media): add persistence models"
```

### Task 2: Media Repository

**Files:**
- Create: `internal/media/repository/repository_test.go`
- Create: `internal/media/repository/repository.go`
- Create: `internal/media/repository/feed.go`
- Create: `internal/media/repository/subscription.go`
- Create: `internal/media/repository/release.go`
- Create: `internal/media/repository/download.go`

- [ ] **Step 1: Write failing repository tests**

Create `internal/media/repository/repository_test.go` with these concrete tests:

- `TestRepositoryMigratesMediaTables`: `repo.Migrate(ctx)` creates all media tables.
- `TestRepositoryListsEnabledSourcesAndSubscriptions`: disabled feed sources and disabled subscriptions are filtered out.
- `TestRepositoryUpsertsReleaseByFingerprint`: the same fingerprint updates the existing release instead of creating a duplicate row.
- `TestRepositoryMarksReleaseMatched`: matched release stores subscription ID, match reason, downloader, and save path.
- `TestRepositoryCreatesDownloadRef`: download reference stores release ID, subscription ID, task ID, downloader, path, and created status.

- [ ] **Step 2: Run repository tests to verify they fail**

```bash
/opt/homebrew/bin/go test ./internal/media/repository -count=1
```

Expected: FAIL because `internal/media/repository` does not exist.

- [ ] **Step 3: Implement repository**

Add this repository API:

```go
func New(db *gorm.DB) *Repository
func (r *Repository) Migrate(ctx context.Context) error
func (r *Repository) CreateFeedSource(ctx context.Context, source model.FeedSource) (*model.FeedSource, error)
func (r *Repository) ListEnabledFeedSources(ctx context.Context) ([]feed.Source, error)
func (r *Repository) CreateSubscription(ctx context.Context, subscription model.Subscription) (*model.Subscription, error)
func (r *Repository) ListEnabledSubscriptions(ctx context.Context, sourceID uint) ([]subscription.Rule, error)
func (r *Repository) UpsertRelease(ctx context.Context, release feed.Release) (*model.Release, bool, error)
func (r *Repository) GetRelease(ctx context.Context, id uint) (*model.Release, error)
func (r *Repository) MarkReleaseMatched(ctx context.Context, releaseID uint, match subscription.MatchResult) error
func (r *Repository) CreateDownloadRef(ctx context.Context, ref model.DownloadRef) (*model.DownloadRef, error)
```

The boolean from `UpsertRelease` is `true` only when a row was inserted.

- [ ] **Step 4: Run repository tests**

```bash
/opt/homebrew/bin/go test ./internal/media/repository -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/media/repository
git commit -m "feat(media): add persistence repository"
```

### Task 3: Feed Refresh Service

**Files:**
- Create: `internal/media/service/service_test.go`
- Create: `internal/media/service/service.go`
- Create: `internal/media/service/feed_refresh.go`

- [ ] **Step 1: Write failing service refresh tests**

Create `internal/media/service/service_test.go` with these concrete tests:

- `TestRefreshSourceParsesAndMatchesReleases`: RSS input with one item is parsed, upserted, matched against one enabled rule, and marked matched.
- `TestRefreshSourceLeavesUnmatchedReleasePending`: RSS input with no matching rule is upserted and returned as unmatched without calling `MarkReleaseMatched`.
- `TestRefreshSourceStopsOnInvalidSubscriptionRegex`: invalid include or exclude regex returns an error before marking a release.

- [ ] **Step 2: Run service tests to verify they fail**

```bash
/opt/homebrew/bin/go test ./internal/media/service -run 'TestRefreshSource' -count=1
```

Expected: FAIL because `internal/media/service` does not exist.

- [ ] **Step 3: Implement refresh service**

Add this service API:

```go
type RefreshResult struct {
	ReleaseID      uint
	Inserted       bool
	Matched        bool
	Reason         string
	SubscriptionID uint
}

func (s *Service) RefreshSource(ctx context.Context, source feed.Source, input []byte) ([]RefreshResult, error)
```

`RefreshSource` parses input, upserts each release, loads enabled subscriptions for the feed source, applies matchers in repository order, and marks only the first matching subscription.

- [ ] **Step 4: Run service refresh tests**

```bash
/opt/homebrew/bin/go test ./internal/media/service -run 'TestRefreshSource' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/media/service
git commit -m "feat(media): add feed refresh service"
```

### Task 4: Download Dispatch Service

**Files:**
- Modify: `internal/media/service/service_test.go`
- Create: `internal/media/service/download_dispatch.go`

- [ ] **Step 1: Write failing download dispatch tests**

Append these concrete tests to `internal/media/service/service_test.go`:

- `TestDispatchDownloadCreatesDownloaderTaskAndReference`: matched release input calls the downloader once and persists the returned task reference.
- `TestDispatchDownloadRejectsUnmatchedRelease`: release without a subscription ID returns an error and does not call the downloader.

- [ ] **Step 2: Run dispatch tests to verify they fail**

```bash
/opt/homebrew/bin/go test ./internal/media/service -run 'TestDispatchDownload' -count=1
```

Expected: FAIL because `DispatchDownload` does not exist.

- [ ] **Step 3: Implement dispatch service**

Add this service API:

```go
type DispatchInput struct {
	ReleaseID      uint
	SubscriptionID uint
	DownloadURL     string
	DownloadPath    string
	DownloaderKey   string
}

type DispatchResult struct {
	TaskID string
}

func (s *Service) DispatchDownload(ctx context.Context, input DispatchInput) (*DispatchResult, error)
```

The service must reject empty subscription IDs before calling the downloader. The persisted download reference must use `DownloadStatusCreated`.

- [ ] **Step 4: Run service tests**

```bash
/opt/homebrew/bin/go test ./internal/media/service -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/media/service
git commit -m "feat(media): dispatch matched downloads"
```

### Task 5: Milestone Verification

**Files:**
- No new files.

- [ ] **Step 1: Run all media tests**

```bash
/opt/homebrew/bin/go test ./internal/media/... -count=1
```

Expected: all media packages pass.

- [ ] **Step 2: Check worktree status**

```bash
git status --short --branch
```

Expected: clean branch, ahead of `fork/feature/openlist-media` until pushed.

- [ ] **Step 3: Push branch**

```bash
git push
```

Expected: branch updates `https://github.com/yuchanns/OpenList` `feature/openlist-media`.
