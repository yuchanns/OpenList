package download

import (
	"context"
	"errors"
	"testing"
)

func TestOpenListDownloaderPassesRequestToAddURL(t *testing.T) {
	var captured AddURLArgs
	downloader := OpenListDownloader{
		AddURL: func(ctx context.Context, args AddURLArgs) (*TaskRef, error) {
			captured = args
			return &TaskRef{ID: "task-1"}, nil
		},
	}

	ref, err := downloader.Add(context.Background(), Request{
		URL:           "magnet:?xt=urn:btih:abc",
		DownloadPath:  "/downloads/incoming",
		DownloaderKey: "qBittorrent",
		ReleaseID:     100,
	})
	if err != nil {
		t.Fatalf("add download: %v", err)
	}
	if ref.ID != "task-1" {
		t.Fatalf("unexpected task id: %q", ref.ID)
	}
	if captured.URL != "magnet:?xt=urn:btih:abc" {
		t.Fatalf("unexpected url: %q", captured.URL)
	}
	if captured.DstDirPath != "/downloads/incoming" {
		t.Fatalf("unexpected destination: %q", captured.DstDirPath)
	}
	if captured.Tool != "qBittorrent" {
		t.Fatalf("unexpected tool: %q", captured.Tool)
	}
}

func TestOpenListDownloaderValidatesRequiredFields(t *testing.T) {
	downloader := OpenListDownloader{
		AddURL: func(ctx context.Context, args AddURLArgs) (*TaskRef, error) {
			t.Fatal("AddURL should not be called for invalid request")
			return nil, nil
		},
	}

	_, err := downloader.Add(context.Background(), Request{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestOpenListDownloaderReturnsAddURLError(t *testing.T) {
	expected := errors.New("download failed")
	downloader := OpenListDownloader{
		AddURL: func(ctx context.Context, args AddURLArgs) (*TaskRef, error) {
			return nil, expected
		},
	}

	_, err := downloader.Add(context.Background(), Request{
		URL:           "magnet:?xt=urn:btih:abc",
		DownloadPath:  "/downloads/incoming",
		DownloaderKey: "qBittorrent",
	})

	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped add url error, got %v", err)
	}
}

func TestTaskRefFromInfoUsesTaskID(t *testing.T) {
	ref, err := TaskRefFromInfo(fakeTaskInfo{id: "task-123"})
	if err != nil {
		t.Fatalf("task ref from info: %v", err)
	}
	if ref.ID != "task-123" {
		t.Fatalf("unexpected task id: %q", ref.ID)
	}
}

func TestTaskRefFromInfoRejectsEmptyTaskID(t *testing.T) {
	_, err := TaskRefFromInfo(fakeTaskInfo{})
	if err == nil {
		t.Fatal("expected empty task id error")
	}
}

type fakeTaskInfo struct {
	id string
}

func (f fakeTaskInfo) GetID() string {
	return f.id
}
