package media

import (
	mediamodel "github.com/OpenListTeam/OpenList/v4/internal/media/model"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

type SubscriptionRequest struct {
	Name       string `json:"name" form:"name" binding:"required"`
	Enabled    bool   `json:"enabled" form:"enabled"`
	SourceID   uint   `json:"source_id" form:"source_id"`
	Keyword    string `json:"keyword" form:"keyword"`
	Include    string `json:"include" form:"include"`
	Exclude    string `json:"exclude" form:"exclude"`
	MinSize    int64  `json:"min_size" form:"min_size"`
	MaxSize    int64  `json:"max_size" form:"max_size"`
	Downloader string `json:"downloader" form:"downloader"`
	SavePath   string `json:"save_path" form:"save_path"`
}

func (h *Handler) ListSubscriptions(c *gin.Context) {
	subscriptions, err := h.repository.ListSubscriptions(c.Request.Context())
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c, subscriptions)
}

func (h *Handler) CreateSubscription(c *gin.Context) {
	var req SubscriptionRequest
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	item, err := h.repository.CreateSubscription(c.Request.Context(), mediamodel.Subscription{
		Name:       req.Name,
		Enabled:    req.Enabled,
		SourceID:   req.SourceID,
		Keyword:    req.Keyword,
		Include:    req.Include,
		Exclude:    req.Exclude,
		MinSize:    req.MinSize,
		MaxSize:    req.MaxSize,
		Downloader: req.Downloader,
		SavePath:   req.SavePath,
	})
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c, item)
}
