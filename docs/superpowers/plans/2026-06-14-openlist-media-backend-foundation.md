# OpenList Media Backend Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first backend foundation for OpenList Media: feed parsing, release fingerprinting, subscription matching, and a thin OpenList downloader adapter boundary.

**Architecture:** This milestone creates focused Go packages under `internal/media` with pure-domain logic first. It does not add HTTP routes, database persistence, frontend pages, TMDB scraping, organizer behavior, or Jellyfin integration yet. Later milestones will wire these tested units into GORM, task managers, APIs, and the frontend.

**Tech Stack:** Go 1.24, standard `testing`, standard XML parsing, OpenList `internal/offline_download/tool` adapter boundary.

---

## Scope

This plan implements a working, testable backend slice:

- Parse RSS and Atom feed content into normalized media releases.
- Generate stable fingerprints for release de-duplication.
- Match releases against subscription rules using keyword, include regex, exclude regex, and size bounds.
- Convert matched releases into download requests without exposing qBittorrent or other tool-specific concepts.

This plan intentionally leaves these for later plans:

- GORM models and migration.
- API handlers under `/api/admin/media`.
- OpenList task manager integration.
- TMDB lookup and NFO/image scraping.
- Organizer and final library paths.
- Frontend pages.

## File Structure

- Create `internal/media/feed/types.go`: feed source and normalized release types.
- Create `internal/media/feed/fingerprint.go`: stable fingerprint helper.
- Create `internal/media/feed/parser.go`: RSS and Atom parser.
- Create `internal/media/feed/parser_test.go`: parser and fingerprint tests.
- Create `internal/media/subscription/types.go`: subscription rule and match result types.
- Create `internal/media/subscription/matcher.go`: matcher implementation.
- Create `internal/media/subscription/matcher_test.go`: matcher tests.
- Create `internal/media/download/types.go`: media-level download request and task reference.
- Create `internal/media/download/openlist.go`: adapter that calls an injected URL add function.
- Create `internal/media/download/openlist_test.go`: adapter tests with a real fake function.

## Verification Commands

Use local toolchain to avoid automatic `go1.24.13` download during development:

```bash
GOTOOLCHAIN=local go test ./internal/media/...
```

Expected after all tasks:

```text
ok  	github.com/OpenListTeam/OpenList/v4/internal/media/feed
ok  	github.com/OpenListTeam/OpenList/v4/internal/media/subscription
ok  	github.com/OpenListTeam/OpenList/v4/internal/media/download
```

---

### Task 1: Feed Types And Fingerprint

**Files:**
- Create: `internal/media/feed/types.go`
- Create: `internal/media/feed/fingerprint.go`
- Test: `internal/media/feed/parser_test.go`

- [ ] **Step 1: Write the failing fingerprint test**

Create `internal/media/feed/parser_test.go`:

```go
package feed

import "testing"

func TestReleaseFingerprintUsesStableIdentityFields(t *testing.T) {
	release := Release{
		SourceID:    42,
		Title:       "Example Movie 2024 2160p",
		Link:        "https://example.test/item/1",
		DownloadURL: "magnet:?xt=urn:btih:abc",
	}

	first := release.Fingerprint()
	second := release.Fingerprint()

	if first == "" {
		t.Fatal("expected fingerprint to be non-empty")
	}
	if first != second {
		t.Fatalf("expected stable fingerprint, got %q and %q", first, second)
	}
}

func TestReleaseFingerprintChangesForDifferentSource(t *testing.T) {
	left := Release{
		SourceID:    1,
		Title:       "Example Movie 2024 2160p",
		Link:        "https://example.test/item/1",
		DownloadURL: "magnet:?xt=urn:btih:abc",
	}
	right := left
	right.SourceID = 2

	if left.Fingerprint() == right.Fingerprint() {
		t.Fatal("expected source id to affect fingerprint")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
GOTOOLCHAIN=local go test ./internal/media/feed -run 'TestReleaseFingerprint' -count=1
```

Expected: FAIL because `internal/media/feed` and `Release` do not exist.

- [ ] **Step 3: Add minimal feed types**

Create `internal/media/feed/types.go`:

```go
package feed

import "time"

type SourceKind string

const (
	SourceKindRSS  SourceKind = "rss"
	SourceKindAtom SourceKind = "atom"
)

type Source struct {
	ID       uint
	Name     string
	URL      string
	Kind     SourceKind
	Enabled  bool
	ProxyURL string
}

type Release struct {
	SourceID    uint
	Title       string
	Description string
	Link        string
	DownloadURL string
	PublishedAt *time.Time
	Size        int64
}
```

- [ ] **Step 4: Add fingerprint implementation**

Create `internal/media/feed/fingerprint.go`:

```go
package feed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

func (r Release) Fingerprint() string {
	parts := []string{
		fmt.Sprintf("%d", r.SourceID),
		normalizeFingerprintPart(r.Title),
		normalizeFingerprintPart(r.Link),
		normalizeFingerprintPart(r.DownloadURL),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func normalizeFingerprintPart(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}
```

- [ ] **Step 5: Run test to verify it passes**

Run:

```bash
GOTOOLCHAIN=local go test ./internal/media/feed -run 'TestReleaseFingerprint' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/media/feed/types.go internal/media/feed/fingerprint.go internal/media/feed/parser_test.go
git commit -m "feat(media): add release fingerprint"
```

---

### Task 2: RSS And Atom Parser

**Files:**
- Modify: `internal/media/feed/parser_test.go`
- Create: `internal/media/feed/parser.go`

- [ ] **Step 1: Add failing parser tests**

Append to `internal/media/feed/parser_test.go`:

```go
func TestParseRSSExtractsReleaseFields(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Example Movie 2024 2160p</title>
      <description>Example description</description>
      <link>https://example.test/item/1</link>
      <pubDate>Mon, 02 Jan 2006 15:04:05 GMT</pubDate>
      <enclosure url="magnet:?xt=urn:btih:abc" length="12345" type="application/x-bittorrent" />
    </item>
  </channel>
</rss>`)

	releases, err := Parse(Source{ID: 7, Kind: SourceKindRSS}, input)
	if err != nil {
		t.Fatalf("parse rss: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("expected 1 release, got %d", len(releases))
	}
	release := releases[0]
	if release.SourceID != 7 {
		t.Fatalf("expected source id 7, got %d", release.SourceID)
	}
	if release.Title != "Example Movie 2024 2160p" {
		t.Fatalf("unexpected title: %q", release.Title)
	}
	if release.Description != "Example description" {
		t.Fatalf("unexpected description: %q", release.Description)
	}
	if release.Link != "https://example.test/item/1" {
		t.Fatalf("unexpected link: %q", release.Link)
	}
	if release.DownloadURL != "magnet:?xt=urn:btih:abc" {
		t.Fatalf("unexpected download url: %q", release.DownloadURL)
	}
	if release.Size != 12345 {
		t.Fatalf("unexpected size: %d", release.Size)
	}
	if release.PublishedAt == nil {
		t.Fatal("expected published time")
	}
}

func TestParseAtomExtractsReleaseFields(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>Example Show S01E01</title>
    <summary>Episode summary</summary>
    <link href="https://example.test/item/2" />
    <link rel="enclosure" href="magnet:?xt=urn:btih:def" length="67890" />
    <updated>2006-01-02T15:04:05Z</updated>
  </entry>
</feed>`)

	releases, err := Parse(Source{ID: 8, Kind: SourceKindAtom}, input)
	if err != nil {
		t.Fatalf("parse atom: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("expected 1 release, got %d", len(releases))
	}
	release := releases[0]
	if release.Title != "Example Show S01E01" {
		t.Fatalf("unexpected title: %q", release.Title)
	}
	if release.Description != "Episode summary" {
		t.Fatalf("unexpected description: %q", release.Description)
	}
	if release.Link != "https://example.test/item/2" {
		t.Fatalf("unexpected link: %q", release.Link)
	}
	if release.DownloadURL != "magnet:?xt=urn:btih:def" {
		t.Fatalf("unexpected download url: %q", release.DownloadURL)
	}
	if release.Size != 67890 {
		t.Fatalf("unexpected size: %d", release.Size)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
GOTOOLCHAIN=local go test ./internal/media/feed -run 'TestParse' -count=1
```

Expected: FAIL because `Parse` does not exist.

- [ ] **Step 3: Implement parser**

Create `internal/media/feed/parser.go`:

```go
package feed

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type rssDocument struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string       `xml:"title"`
	Description string       `xml:"description"`
	Link        string       `xml:"link"`
	PubDate     string       `xml:"pubDate"`
	Enclosure   rssEnclosure `xml:"enclosure"`
}

type rssEnclosure struct {
	URL    string `xml:"url,attr"`
	Length string `xml:"length,attr"`
}

type atomDocument struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string     `xml:"title"`
	Summary string     `xml:"summary"`
	Content string     `xml:"content"`
	Updated string     `xml:"updated"`
	Links   []atomLink `xml:"link"`
}

type atomLink struct {
	Rel    string `xml:"rel,attr"`
	Href   string `xml:"href,attr"`
	Length string `xml:"length,attr"`
}

func Parse(source Source, input []byte) ([]Release, error) {
	kind := source.Kind
	if kind == "" {
		kind = detectKind(input)
	}
	switch kind {
	case SourceKindRSS:
		return parseRSS(source, input)
	case SourceKindAtom:
		return parseAtom(source, input)
	default:
		return nil, fmt.Errorf("unsupported feed kind: %s", kind)
	}
}

func detectKind(input []byte) SourceKind {
	trimmed := bytes.TrimSpace(input)
	if bytes.Contains(trimmed, []byte("<rss")) {
		return SourceKindRSS
	}
	if bytes.Contains(trimmed, []byte("<feed")) {
		return SourceKindAtom
	}
	return ""
}

func parseRSS(source Source, input []byte) ([]Release, error) {
	var doc rssDocument
	if err := xml.Unmarshal(input, &doc); err != nil {
		return nil, err
	}
	releases := make([]Release, 0, len(doc.Channel.Items))
	for _, item := range doc.Channel.Items {
		releases = append(releases, Release{
			SourceID:    source.ID,
			Title:       cleanText(item.Title),
			Description: cleanText(item.Description),
			Link:        cleanText(item.Link),
			DownloadURL: cleanText(item.Enclosure.URL),
			PublishedAt: parseTime(item.PubDate),
			Size:        parseSize(item.Enclosure.Length),
		})
	}
	return releases, nil
}

func parseAtom(source Source, input []byte) ([]Release, error) {
	var doc atomDocument
	if err := xml.Unmarshal(input, &doc); err != nil {
		return nil, err
	}
	releases := make([]Release, 0, len(doc.Entries))
	for _, entry := range doc.Entries {
		pageURL, downloadURL, size := atomLinks(entry.Links)
		description := entry.Summary
		if description == "" {
			description = entry.Content
		}
		releases = append(releases, Release{
			SourceID:    source.ID,
			Title:       cleanText(entry.Title),
			Description: cleanText(description),
			Link:        cleanText(pageURL),
			DownloadURL: cleanText(downloadURL),
			PublishedAt: parseTime(entry.Updated),
			Size:        size,
		})
	}
	return releases, nil
}

func atomLinks(links []atomLink) (pageURL string, downloadURL string, size int64) {
	for _, link := range links {
		rel := strings.ToLower(strings.TrimSpace(link.Rel))
		if rel == "enclosure" {
			downloadURL = link.Href
			size = parseSize(link.Length)
			continue
		}
		if pageURL == "" && (rel == "" || rel == "alternate") {
			pageURL = link.Href
		}
	}
	return pageURL, downloadURL, size
}

func cleanText(value string) string {
	return strings.TrimSpace(value)
}

func parseSize(value string) int64 {
	if value == "" {
		return 0
	}
	size, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || size < 0 {
		return 0
	}
	return size
}

func parseTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	layouts := []string{time.RFC1123Z, time.RFC1123, time.RFC3339}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed
		}
	}
	return nil
}
```

- [ ] **Step 4: Run parser tests**

Run:

```bash
GOTOOLCHAIN=local go test ./internal/media/feed -run 'TestParse|TestReleaseFingerprint' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/media/feed/parser.go internal/media/feed/parser_test.go
git commit -m "feat(media): parse rss and atom releases"
```

---

### Task 3: Subscription Matcher

**Files:**
- Create: `internal/media/subscription/types.go`
- Create: `internal/media/subscription/matcher.go`
- Test: `internal/media/subscription/matcher_test.go`

- [ ] **Step 1: Write failing matcher tests**

Create `internal/media/subscription/matcher_test.go`:

```go
package subscription

import (
	"testing"

	"github.com/OpenListTeam/OpenList/v4/internal/media/feed"
)

func TestMatcherAcceptsReleaseWhenKeywordAndIncludeMatch(t *testing.T) {
	matcher, err := NewMatcher(Rule{
		ID:      10,
		Name:    "Example Movie",
		Keyword: "Example Movie",
		Include: "2160p",
	})
	if err != nil {
		t.Fatalf("new matcher: %v", err)
	}

	result := matcher.Match(feed.Release{Title: "Example Movie 2024 2160p WEB-DL"})

	if !result.Matched {
		t.Fatalf("expected match, got reason %q", result.Reason)
	}
	if result.SubscriptionID != 10 {
		t.Fatalf("unexpected subscription id: %d", result.SubscriptionID)
	}
}

func TestMatcherRejectsReleaseWhenExcludeMatches(t *testing.T) {
	matcher, err := NewMatcher(Rule{
		ID:      11,
		Keyword: "Example Movie",
		Exclude: "cam|ts",
	})
	if err != nil {
		t.Fatalf("new matcher: %v", err)
	}

	result := matcher.Match(feed.Release{Title: "Example Movie 2024 CAM"})

	if result.Matched {
		t.Fatal("expected release to be rejected")
	}
	if result.Reason != "exclude_matched" {
		t.Fatalf("unexpected reason: %q", result.Reason)
	}
}

func TestMatcherRejectsReleaseOutsideSizeRange(t *testing.T) {
	matcher, err := NewMatcher(Rule{
		ID:      12,
		Keyword: "Example Movie",
		MinSize: 100,
		MaxSize: 200,
	})
	if err != nil {
		t.Fatalf("new matcher: %v", err)
	}

	result := matcher.Match(feed.Release{Title: "Example Movie 2024", Size: 99})

	if result.Matched {
		t.Fatal("expected release to be rejected")
	}
	if result.Reason != "below_min_size" {
		t.Fatalf("unexpected reason: %q", result.Reason)
	}
}

func TestNewMatcherRejectsInvalidRegex(t *testing.T) {
	_, err := NewMatcher(Rule{Include: "["})
	if err == nil {
		t.Fatal("expected invalid regex error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
GOTOOLCHAIN=local go test ./internal/media/subscription -count=1
```

Expected: FAIL because package and matcher do not exist.

- [ ] **Step 3: Add subscription types**

Create `internal/media/subscription/types.go`:

```go
package subscription

type Rule struct {
	ID         uint
	Name       string
	Keyword    string
	Include    string
	Exclude    string
	MinSize    int64
	MaxSize    int64
	Downloader string
	SavePath   string
}

type MatchResult struct {
	Matched        bool
	Reason         string
	SubscriptionID uint
	Downloader     string
	SavePath        string
}
```

- [ ] **Step 4: Add matcher implementation**

Create `internal/media/subscription/matcher.go`:

```go
package subscription

import (
	"regexp"
	"strings"

	"github.com/OpenListTeam/OpenList/v4/internal/media/feed"
)

type Matcher struct {
	rule    Rule
	include *regexp.Regexp
	exclude *regexp.Regexp
}

func NewMatcher(rule Rule) (*Matcher, error) {
	include, err := compilePattern(rule.Include)
	if err != nil {
		return nil, err
	}
	exclude, err := compilePattern(rule.Exclude)
	if err != nil {
		return nil, err
	}
	return &Matcher{rule: rule, include: include, exclude: exclude}, nil
}

func (m *Matcher) Match(release feed.Release) MatchResult {
	text := strings.TrimSpace(release.Title + " " + release.Description)
	lowerText := strings.ToLower(text)
	keyword := strings.ToLower(strings.TrimSpace(m.rule.Keyword))
	if keyword != "" && !strings.Contains(lowerText, keyword) {
		return m.reject("keyword_not_matched")
	}
	if m.include != nil && !m.include.MatchString(text) {
		return m.reject("include_not_matched")
	}
	if m.exclude != nil && m.exclude.MatchString(text) {
		return m.reject("exclude_matched")
	}
	if m.rule.MinSize > 0 && release.Size > 0 && release.Size < m.rule.MinSize {
		return m.reject("below_min_size")
	}
	if m.rule.MaxSize > 0 && release.Size > 0 && release.Size > m.rule.MaxSize {
		return m.reject("above_max_size")
	}
	return MatchResult{
		Matched:        true,
		Reason:         "matched",
		SubscriptionID: m.rule.ID,
		Downloader:     m.rule.Downloader,
		SavePath:        m.rule.SavePath,
	}
}

func (m *Matcher) reject(reason string) MatchResult {
	return MatchResult{Matched: false, Reason: reason, SubscriptionID: m.rule.ID}
}

func compilePattern(pattern string) (*regexp.Regexp, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, nil
	}
	return regexp.Compile("(?i)" + pattern)
}
```

- [ ] **Step 5: Run matcher tests**

Run:

```bash
GOTOOLCHAIN=local go test ./internal/media/subscription -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/media/subscription
git commit -m "feat(media): match releases to subscriptions"
```

---

### Task 4: Download Adapter Boundary

**Files:**
- Create: `internal/media/download/types.go`
- Create: `internal/media/download/openlist.go`
- Test: `internal/media/download/openlist_test.go`

- [ ] **Step 1: Write failing adapter tests**

Create `internal/media/download/openlist_test.go`:

```go
package download

import (
	"context"
	"errors"
	"testing"
)

func TestOpenListDownloaderPassesRequestToAddURL(t *testing.T) {
	var captured AddURLArgs
	downloader := OpenListDownloader{
		AddURL: func(ctx context.Context, args AddURLArgs) (*TaskRef, error) {
			captured = args
			return &TaskRef{ID: "task-1"}, nil
		},
	}

	ref, err := downloader.Add(context.Background(), Request{
		URL:           "magnet:?xt=urn:btih:abc",
		DownloadPath:  "/downloads/incoming",
		DownloaderKey: "qBittorrent",
		ReleaseID:     100,
	})
	if err != nil {
		t.Fatalf("add download: %v", err)
	}
	if ref.ID != "task-1" {
		t.Fatalf("unexpected task id: %q", ref.ID)
	}
	if captured.URL != "magnet:?xt=urn:btih:abc" {
		t.Fatalf("unexpected url: %q", captured.URL)
	}
	if captured.DstDirPath != "/downloads/incoming" {
		t.Fatalf("unexpected destination: %q", captured.DstDirPath)
	}
	if captured.Tool != "qBittorrent" {
		t.Fatalf("unexpected tool: %q", captured.Tool)
	}
}

func TestOpenListDownloaderValidatesRequiredFields(t *testing.T) {
	downloader := OpenListDownloader{
		AddURL: func(ctx context.Context, args AddURLArgs) (*TaskRef, error) {
			t.Fatal("AddURL should not be called for invalid request")
			return nil, nil
		},
	}

	_, err := downloader.Add(context.Background(), Request{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestOpenListDownloaderReturnsAddURLError(t *testing.T) {
	expected := errors.New("download failed")
	downloader := OpenListDownloader{
		AddURL: func(ctx context.Context, args AddURLArgs) (*TaskRef, error) {
			return nil, expected
		},
	}

	_, err := downloader.Add(context.Background(), Request{
		URL:           "magnet:?xt=urn:btih:abc",
		DownloadPath:  "/downloads/incoming",
		DownloaderKey: "qBittorrent",
	})

	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped add url error, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
GOTOOLCHAIN=local go test ./internal/media/download -count=1
```

Expected: FAIL because package and adapter do not exist.

- [ ] **Step 3: Add download types**

Create `internal/media/download/types.go`:

```go
package download

type Request struct {
	URL            string
	DownloadPath   string
	DownloaderKey  string
	SubscriptionID uint
	ReleaseID      uint
}

type TaskRef struct {
	ID string
}

type AddURLArgs struct {
	URL        string
	DstDirPath string
	Tool       string
}
```

- [ ] **Step 4: Add adapter implementation**

Create `internal/media/download/openlist.go`:

```go
package download

import (
	"context"
	"errors"
	"fmt"
)

type AddURLFunc func(ctx context.Context, args AddURLArgs) (*TaskRef, error)

type OpenListDownloader struct {
	AddURL AddURLFunc
}

func (d OpenListDownloader) Add(ctx context.Context, req Request) (*TaskRef, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	if d.AddURL == nil {
		return nil, errors.New("download add url function is not configured")
	}
	ref, err := d.AddURL(ctx, AddURLArgs{
		URL:        req.URL,
		DstDirPath: req.DownloadPath,
		Tool:       req.DownloaderKey,
	})
	if err != nil {
		return nil, fmt.Errorf("add openlist download: %w", err)
	}
	return ref, nil
}

func validateRequest(req Request) error {
	if req.URL == "" {
		return errors.New("download url is required")
	}
	if req.DownloadPath == "" {
		return errors.New("download path is required")
	}
	if req.DownloaderKey == "" {
		return errors.New("downloader key is required")
	}
	return nil
}
```

- [ ] **Step 5: Run adapter tests**

Run:

```bash
GOTOOLCHAIN=local go test ./internal/media/download -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/media/download
git commit -m "feat(media): add download adapter boundary"
```

---

### Task 5: Run Milestone Verification

**Files:**
- No new files.

- [ ] **Step 1: Run all media tests**

Run:

```bash
GOTOOLCHAIN=local go test ./internal/media/... -count=1
```

Expected: all three media packages pass.

- [ ] **Step 2: Check worktree status**

Run:

```bash
git status --short
```

Expected: no uncommitted changes.

- [ ] **Step 3: Push branch to fork**

Run:

```bash
git push -u fork feature/openlist-media
```

Expected: branch is pushed to `https://github.com/yuchanns/OpenList`.

