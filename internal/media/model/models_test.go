package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestModelsMigrateExpectedTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	if err := db.AutoMigrate(Models()...); err != nil {
		t.Fatalf("migrate media models: %v", err)
	}

	for _, table := range []string{
		"media_feed_sources",
		"media_subscriptions",
		"media_releases",
		"media_download_refs",
	} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s to exist", table)
		}
	}
}

func TestReleaseFingerprintIsUnique(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(Models()...); err != nil {
		t.Fatalf("migrate media models: %v", err)
	}

	first := Release{SourceID: 1, Fingerprint: "same", Title: "First", Status: ReleaseStatusPending}
	second := Release{SourceID: 1, Fingerprint: "same", Title: "Second", Status: ReleaseStatusPending}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first release: %v", err)
	}
	if err := db.Create(&second).Error; err == nil {
		t.Fatal("expected duplicate fingerprint to fail")
	}
}
