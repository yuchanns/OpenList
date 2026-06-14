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
