package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/OpenListTeam/OpenList/v4/internal/media/feed"
	"github.com/OpenListTeam/OpenList/v4/internal/media/subscription"
)

type RefreshResult struct {
	ReleaseID      uint
	Inserted       bool
	Matched        bool
	Reason         string
	SubscriptionID uint
}

type ruleMatcher struct {
	rule    subscription.Rule
	matcher *subscription.Matcher
}

func (s *Service) RefreshSource(ctx context.Context, source feed.Source, input []byte) ([]RefreshResult, error) {
	if s.repository == nil {
		return nil, errors.New("media repository is not configured")
	}
	matchers, err := s.buildMatchers(ctx, source.ID)
	if err != nil {
		return nil, err
	}
	releases, err := feed.Parse(source, input)
	if err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}

	results := make([]RefreshResult, 0, len(releases))
	for _, release := range releases {
		stored, inserted, err := s.repository.UpsertRelease(ctx, release)
		if err != nil {
			return nil, fmt.Errorf("upsert release: %w", err)
		}
		result := RefreshResult{
			ReleaseID: stored.ID,
			Inserted:  inserted,
			Reason:    "no_subscription_matched",
		}
		for _, matcher := range matchers {
			match := matcher.matcher.Match(release)
			if !match.Matched {
				continue
			}
			if err := s.repository.MarkReleaseMatched(ctx, stored.ID, match); err != nil {
				return nil, fmt.Errorf("mark release matched: %w", err)
			}
			result.Matched = true
			result.Reason = match.Reason
			result.SubscriptionID = match.SubscriptionID
			break
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) buildMatchers(ctx context.Context, sourceID uint) ([]ruleMatcher, error) {
	rules, err := s.repository.ListEnabledSubscriptions(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("list enabled subscriptions: %w", err)
	}
	matchers := make([]ruleMatcher, 0, len(rules))
	for _, rule := range rules {
		matcher, err := subscription.NewMatcher(rule)
		if err != nil {
			return nil, fmt.Errorf("build subscription matcher %d: %w", rule.ID, err)
		}
		matchers = append(matchers, ruleMatcher{rule: rule, matcher: matcher})
	}
	return matchers, nil
}
