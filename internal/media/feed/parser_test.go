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
