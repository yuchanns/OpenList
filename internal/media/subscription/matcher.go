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
