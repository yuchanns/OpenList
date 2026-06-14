package media

import (
	"errors"
	"strconv"

	mediarepo "github.com/OpenListTeam/OpenList/v4/internal/media/repository"
	mediaservice "github.com/OpenListTeam/OpenList/v4/internal/media/service"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

func (h *Handler) ListReleases(c *gin.Context) {
	limit, err := parseOptionalInt(c.Query("limit"))
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	releases, err := h.repository.ListReleases(c.Request.Context(), mediarepo.ReleaseFilter{
		Status: c.Query("status"),
		Limit:  limit,
	})
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c, releases)
}

func (h *Handler) DownloadRelease(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	release, err := h.repository.GetRelease(c.Request.Context(), uint(id))
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	if release.SubscriptionID == nil || *release.SubscriptionID == 0 {
		common.ErrorResp(c, errors.New("release is not matched to a subscription"), 400)
		return
	}
	result, err := h.service.DispatchDownload(c.Request.Context(), mediaservice.DispatchInput{
		ReleaseID:      release.ID,
		SubscriptionID: *release.SubscriptionID,
		DownloadURL:    release.DownloadURL,
		DownloadPath:   release.SavePath,
		DownloaderKey:  release.Downloader,
	})
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c, result)
}

func parseOptionalInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}
