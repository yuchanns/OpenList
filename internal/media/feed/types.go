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
