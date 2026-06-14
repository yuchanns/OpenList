package db

import (
	"testing"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestInitMigratesMediaTables(t *testing.T) {
	dB, err := gorm.Open(sqlite.Open("file:media_migration?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	conf.Conf = conf.DefaultConfig("data")

	Init(dB)

	for _, table := range []string{
		"media_feed_sources",
		"media_subscriptions",
		"media_releases",
		"media_download_refs",
	} {
		if !dB.Migrator().HasTable(table) {
			t.Fatalf("expected table %s to exist", table)
		}
	}
}
