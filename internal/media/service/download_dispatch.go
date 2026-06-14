package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/OpenListTeam/OpenList/v4/internal/media/download"
	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
)

type DispatchInput struct {
	ReleaseID      uint
	SubscriptionID uint
	DownloadURL    string
	DownloadPath   string
	DownloaderKey  string
}

type DispatchResult struct {
	TaskID string
}

func (s *Service) DispatchDownload(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
	if err := validateDispatchInput(input); err != nil {
		return nil, err
	}
	if s.downloader == nil {
		return nil, errors.New("media downloader is not configured")
	}
	if s.repository == nil {
		return nil, errors.New("media repository is not configured")
	}

	taskRef, err := s.downloader.Add(ctx, download.Request{
		URL:            input.DownloadURL,
		DownloadPath:   input.DownloadPath,
		DownloaderKey:  input.DownloaderKey,
		SubscriptionID: input.SubscriptionID,
		ReleaseID:      input.ReleaseID,
	})
	if err != nil {
		return nil, fmt.Errorf("add download: %w", err)
	}
	if taskRef == nil || taskRef.ID == "" {
		return nil, errors.New("download task id is required")
	}
	if _, err := s.repository.CreateDownloadRef(ctx, mediamodel.DownloadRef{
		ReleaseID:      input.ReleaseID,
		SubscriptionID: input.SubscriptionID,
		TaskID:         taskRef.ID,
		DownloadPath:   input.DownloadPath,
		DownloaderKey:  input.DownloaderKey,
		Status:         mediamodel.DownloadStatusCreated,
	}); err != nil {
		return nil, fmt.Errorf("create download ref: %w", err)
	}
	return &DispatchResult{TaskID: taskRef.ID}, nil
}

func validateDispatchInput(input DispatchInput) error {
	if input.ReleaseID == 0 {
		return errors.New("release id is required")
	}
	if input.SubscriptionID == 0 {
		return errors.New("subscription id is required")
	}
	if input.DownloadURL == "" {
		return errors.New("download url is required")
	}
	if input.DownloadPath == "" {
		return errors.New("download path is required")
	}
	if input.DownloaderKey == "" {
		return errors.New("downloader key is required")
	}
	return nil
}
