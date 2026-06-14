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
