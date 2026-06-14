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
