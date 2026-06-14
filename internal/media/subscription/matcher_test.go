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
