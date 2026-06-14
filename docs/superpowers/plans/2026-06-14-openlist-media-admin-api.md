# OpenList Media Admin API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the first runnable OpenList Media admin API for feed and subscription management, manual feed refresh, release listing, and matched-release download dispatch.

**Architecture:** Keep media logic in `internal/media/...` and API handling in `server/handles/media`. The only OpenList core touch points are media table registration in `internal/db/db.go` and route registration in `server/router.go`. The API composes the existing repository, service, HTTP feed fetcher, and OpenList downloader bridge.

**Tech Stack:** Go 1.24, Gin, GORM, `github.com/glebarez/sqlite` for tests, existing `server/common` response helpers, existing `internal/media` packages.

---

## Low-Intrusion Touch Points

- Modify `internal/db/db.go`: append media models to `AutoMigrate` during global DB initialization.
- Modify `server/router.go`: import `server/handles/media` and call `media.RegisterRoutes(g.Group("/media"))` inside `admin()`.

All other new code stays under:

- `internal/media/...`
- `server/handles/media/...`

## API Scope

This milestone exposes only the subscription/download chain:

- `GET /api/admin/media/feeds`
- `POST /api/admin/media/feeds`
- `GET /api/admin/media/subscriptions`
- `POST /api/admin/media/subscriptions`
- `POST /api/admin/media/feeds/:id/refresh`
- `GET /api/admin/media/releases`
- `POST /api/admin/media/releases/:id/download`

TMDB scraping, organizer, file-list context actions, and frontend pages remain out of scope for this milestone.

## File Structure

- Modify `internal/db/db.go`: include media models in startup migration.
- Create or modify `internal/db/db_test.go`: verify `db.Init` migrates media tables.
- Modify `internal/media/repository/feed.go`: add `ListFeedSources` and `GetFeedSource`.
- Modify `internal/media/repository/subscription.go`: add `ListSubscriptions`.
- Modify `internal/media/repository/release.go`: add `ListReleases`.
- Modify `internal/media/repository/repository_test.go`: cover new repository methods.
- Modify `internal/media/download/openlist.go`: expose `TaskRefFromInfo` conversion helper.
- Create `internal/media/download/tool.go`: bridge media downloader to `internal/offline_download/tool.AddURL`.
- Modify `internal/media/download/openlist_test.go`: cover task-ref conversion.
- Create `server/handles/media/fetcher.go`: HTTP feed fetcher.
- Create `server/handles/media/handler.go`: handler dependencies and real wiring.
- Create `server/handles/media/routes.go`: route registration.
- Create `server/handles/media/feed.go`: feed list/create/refresh handlers.
- Create `server/handles/media/subscription.go`: subscription list/create handlers.
- Create `server/handles/media/release.go`: release list/download handlers.
- Create `server/handles/media/handler_test.go`: handler tests with fakes.
- Modify `server/router.go`: register media admin routes.

## Verification Commands

Use Homebrew Go and a reliable Go proxy:

```bash
GOPROXY=https://goproxy.cn,direct /opt/homebrew/bin/go test ./internal/media/... ./server/handles/media ./internal/db -count=1
```

Expected:

```text
ok  	github.com/OpenListTeam/OpenList/v4/internal/media/...
ok  	github.com/OpenListTeam/OpenList/v4/server/handles/media
ok  	github.com/OpenListTeam/OpenList/v4/internal/db
```

---

### Task 1: Global Media Migration Registration

**Files:**
- Create or modify: `internal/db/db_test.go`
- Modify: `internal/db/db.go`

- [ ] **Step 1: Write failing migration integration test**

Add a test that opens an in-memory SQLite database, calls `db.Init(dB)`, and asserts `media_feed_sources`, `media_subscriptions`, `media_releases`, and `media_download_refs` exist.

- [ ] **Step 2: Run test to verify it fails**

```bash
GOPROXY=https://goproxy.cn,direct /opt/homebrew/bin/go test ./internal/db -run 'TestInitMigratesMediaTables' -count=1
```

Expected: FAIL because global DB migration does not include media models yet.

- [ ] **Step 3: Add media models to db.Init migration**

Import `internal/media/model` as `mediamodel` and append `mediamodel.Models()...` to the migration list without changing existing OpenList models.

- [ ] **Step 4: Run migration test**

```bash
GOPROXY=https://goproxy.cn,direct /opt/homebrew/bin/go test ./internal/db -run 'TestInitMigratesMediaTables' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/db/db.go internal/db/db_test.go
git commit -m "feat(media): migrate media tables on startup"
```

### Task 2: Repository API Read Methods

**Files:**
- Modify: `internal/media/repository/repository_test.go`
- Modify: `internal/media/repository/feed.go`
- Modify: `internal/media/repository/subscription.go`
- Modify: `internal/media/repository/release.go`

- [ ] **Step 1: Write failing repository API tests**

Add tests for:

- `ListFeedSources(ctx)` returning enabled and disabled feeds in ID order.
- `GetFeedSource(ctx, id)` returning one feed source.
- `ListSubscriptions(ctx)` returning enabled and disabled subscriptions in ID order.
- `ListReleases(ctx, filter)` returning releases filtered by status and ordered by newest first.

- [ ] **Step 2: Run repository tests to verify they fail**

```bash
GOPROXY=https://goproxy.cn,direct /opt/homebrew/bin/go test ./internal/media/repository -run 'TestRepositoryListsAll|TestRepositoryGetsFeedSource|TestRepositoryListsReleases' -count=1
```

Expected: FAIL because these repository methods do not exist.

- [ ] **Step 3: Implement repository API methods**

Add:

```go
type ReleaseFilter struct {
	Status string
	Limit  int
}

func (r *Repository) ListFeedSources(ctx context.Context) ([]mediamodel.FeedSource, error)
func (r *Repository) GetFeedSource(ctx context.Context, id uint) (*mediamodel.FeedSource, error)
func (r *Repository) ListSubscriptions(ctx context.Context) ([]mediamodel.Subscription, error)
func (r *Repository) ListReleases(ctx context.Context, filter ReleaseFilter) ([]mediamodel.Release, error)
```

- [ ] **Step 4: Run repository tests**

```bash
GOPROXY=https://goproxy.cn,direct /opt/homebrew/bin/go test ./internal/media/repository -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/media/repository
git commit -m "feat(media): expose admin repository queries"
```

### Task 3: OpenList Downloader Bridge

**Files:**
- Modify: `internal/media/download/openlist_test.go`
- Modify: `internal/media/download/openlist.go`
- Create: `internal/media/download/tool.go`

- [ ] **Step 1: Write failing task-ref conversion test**

Add a test that passes a fake task info with `GetID() string` and expects a `TaskRef` with that ID.

- [ ] **Step 2: Run download tests to verify they fail**

```bash
GOPROXY=https://goproxy.cn,direct /opt/homebrew/bin/go test ./internal/media/download -run 'TestTaskRefFromInfo' -count=1
```

Expected: FAIL because the conversion helper does not exist.

- [ ] **Step 3: Implement downloader bridge**

Add a `TaskRefFromInfo` helper and `NewOpenListToolDownloader()` that calls `tool.AddURL` with the media request fields, then converts the returned task info into `TaskRef`.

- [ ] **Step 4: Run download tests**

```bash
GOPROXY=https://goproxy.cn,direct /opt/homebrew/bin/go test ./internal/media/download -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/media/download
git commit -m "feat(media): bridge openlist downloader"
```

### Task 4: Media Admin Handlers

**Files:**
- Create: `server/handles/media/handler_test.go`
- Create: `server/handles/media/fetcher.go`
- Create: `server/handles/media/handler.go`
- Create: `server/handles/media/routes.go`
- Create: `server/handles/media/feed.go`
- Create: `server/handles/media/subscription.go`
- Create: `server/handles/media/release.go`

- [ ] **Step 1: Write failing handler tests**

Add Gin handler tests with fake repository, fake service, and fake fetcher for:

- `POST /feeds` creates a feed source.
- `GET /feeds` lists feed sources.
- `POST /subscriptions` creates a subscription.
- `GET /subscriptions` lists subscriptions.
- `POST /feeds/:id/refresh` fetches feed content and calls `RefreshSource`.
- `GET /releases` lists releases.
- `POST /releases/:id/download` dispatches a matched release.

- [ ] **Step 2: Run handler tests to verify they fail**

```bash
GOPROXY=https://goproxy.cn,direct /opt/homebrew/bin/go test ./server/handles/media -count=1
```

Expected: FAIL because media handler package does not exist.

- [ ] **Step 3: Implement handlers and route registration**

Implement `NewHandler`, `RegisterRoutes`, request structs, `HTTPFetcher`, and real wiring through `repository.New(db.GetDb())`, `service.New(repo, download.NewOpenListToolDownloader())`.

- [ ] **Step 4: Run handler tests**

```bash
GOPROXY=https://goproxy.cn,direct /opt/homebrew/bin/go test ./server/handles/media -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/handles/media
git commit -m "feat(media): add admin api handlers"
```

### Task 5: Admin Route Registration

**Files:**
- Modify: `server/router.go`

- [ ] **Step 1: Register media routes in admin router**

Add one import and one call in `admin()`:

```go
mediahandles.RegisterRoutes(g.Group("/media"))
```

- [ ] **Step 2: Run package tests**

```bash
GOPROXY=https://goproxy.cn,direct /opt/homebrew/bin/go test ./server/handles/media ./internal/media/... ./internal/db -count=1
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add server/router.go
git commit -m "feat(media): register admin routes"
```

### Task 6: Milestone Verification

**Files:**
- No new files.

- [ ] **Step 1: Run targeted verification**

```bash
GOPROXY=https://goproxy.cn,direct /opt/homebrew/bin/go test ./internal/media/... ./server/handles/media ./internal/db -count=1
```

Expected: all targeted packages pass.

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
