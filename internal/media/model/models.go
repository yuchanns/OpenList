package model

import "time"

type FeedSource struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Name          string     `json:"name" gorm:"not null"`
	URL           string     `json:"url" gorm:"not null"`
	Kind          string     `json:"kind" gorm:"not null"`
	Enabled       bool       `json:"enabled" gorm:"index;not null;default:true"`
	ProxyURL      string     `json:"proxy_url"`
	LastCheckedAt *time.Time `json:"last_checked_at"`
	LastError     string     `json:"last_error" gorm:"type:text"`
}

func (FeedSource) TableName() string {
	return "media_feed_sources"
}

type Subscription struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Name       string    `json:"name" gorm:"not null"`
	Enabled    bool      `json:"enabled" gorm:"index;not null;default:true"`
	SourceID   uint      `json:"source_id" gorm:"index"`
	Keyword    string    `json:"keyword"`
	Include    string    `json:"include"`
	Exclude    string    `json:"exclude"`
	MinSize    int64     `json:"min_size"`
	MaxSize    int64     `json:"max_size"`
	Downloader string    `json:"downloader"`
	SavePath   string    `json:"save_path"`
}

func (Subscription) TableName() string {
	return "media_subscriptions"
}

type Release struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	SourceID       uint       `json:"source_id" gorm:"index;not null"`
	SubscriptionID *uint      `json:"subscription_id" gorm:"index"`
	Fingerprint    string     `json:"fingerprint" gorm:"uniqueIndex;not null"`
	Title          string     `json:"title" gorm:"not null"`
	Description    string     `json:"description" gorm:"type:text"`
	Link           string     `json:"link"`
	DownloadURL    string     `json:"download_url"`
	PublishedAt    *time.Time `json:"published_at"`
	Size           int64      `json:"size"`
	Status         string     `json:"status" gorm:"index;not null"`
	MatchReason    string     `json:"match_reason"`
	Downloader     string     `json:"downloader"`
	SavePath       string     `json:"save_path"`
}

func (Release) TableName() string {
	return "media_releases"
}

type DownloadRef struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ReleaseID      uint      `json:"release_id" gorm:"uniqueIndex;not null"`
	SubscriptionID uint      `json:"subscription_id" gorm:"index;not null"`
	TaskID         string    `json:"task_id" gorm:"index;not null"`
	DownloadPath   string    `json:"download_path" gorm:"not null"`
	DownloaderKey  string    `json:"downloader_key" gorm:"not null"`
	Status         string    `json:"status" gorm:"index;not null"`
}

func (DownloadRef) TableName() string {
	return "media_download_refs"
}

func Models() []interface{} {
	return []interface{}{
		new(FeedSource),
		new(Subscription),
		new(Release),
		new(DownloadRef),
	}
}
