package media

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, handlers ...*Handler) {
	handler := NewDefaultHandler()
	if len(handlers) > 0 && handlers[0] != nil {
		handler = handlers[0]
	}

	g.GET("/feeds", handler.ListFeeds)
	g.POST("/feeds", handler.CreateFeed)
	g.POST("/feeds/:id/refresh", handler.RefreshFeed)
	g.GET("/subscriptions", handler.ListSubscriptions)
	g.POST("/subscriptions", handler.CreateSubscription)
	g.GET("/releases", handler.ListReleases)
	g.POST("/releases/:id/download", handler.DownloadRelease)
}
