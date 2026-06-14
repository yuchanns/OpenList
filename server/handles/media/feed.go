package media

import (
	"strconv"

	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

type FeedSourceRequest struct {
	Name     string `json:"name" form:"name" binding:"required"`
	URL      string `json:"url" form:"url" binding:"required"`
	Kind     string `json:"kind" form:"kind" binding:"required"`
	Enabled  bool   `json:"enabled" form:"enabled"`
	ProxyURL string `json:"proxy_url" form:"proxy_url"`
}

func (h *Handler) ListFeeds(c *gin.Context) {
	sources, err := h.repository.ListFeedSources(c.Request.Context())
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c, sources)
}

func (h *Handler) CreateFeed(c *gin.Context) {
	var req FeedSourceRequest
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	source, err := h.repository.CreateFeedSource(c.Request.Context(), mediamodel.FeedSource{
		Name:     req.Name,
		URL:      req.URL,
		Kind:     req.Kind,
		Enabled:  req.Enabled,
		ProxyURL: req.ProxyURL,
	})
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c, source)
}

func (h *Handler) RefreshFeed(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	source, err := h.repository.GetFeedSource(c.Request.Context(), uint(id))
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	content, err := h.fetcher.Fetch(c.Request.Context(), *source)
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	results, err := h.service.RefreshSource(c.Request.Context(), feedSourceFromModel(*source), content)
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c, results)
}
