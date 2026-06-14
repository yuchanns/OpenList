package download

import (
	"context"
	"errors"
	"fmt"
)

type AddURLFunc func(ctx context.Context, args AddURLArgs) (*TaskRef, error)

type TaskInfo interface {
	GetID() string
}

type OpenListDownloader struct {
	AddURL AddURLFunc
}

func (d OpenListDownloader) Add(ctx context.Context, req Request) (*TaskRef, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	if d.AddURL == nil {
		return nil, errors.New("download add url function is not configured")
	}
	ref, err := d.AddURL(ctx, AddURLArgs{
		URL:        req.URL,
		DstDirPath: req.DownloadPath,
		Tool:       req.DownloaderKey,
	})
	if err != nil {
		return nil, fmt.Errorf("add openlist download: %w", err)
	}
	return ref, nil
}

func validateRequest(req Request) error {
	if req.URL == "" {
		return errors.New("download url is required")
	}
	if req.DownloadPath == "" {
		return errors.New("download path is required")
	}
	if req.DownloaderKey == "" {
		return errors.New("downloader key is required")
	}
	return nil
}

func TaskRefFromInfo(info TaskInfo) (*TaskRef, error) {
	if info == nil {
		return nil, errors.New("download task info is required")
	}
	id := info.GetID()
	if id == "" {
		return nil, errors.New("download task id is required")
	}
	return &TaskRef{ID: id}, nil
}
