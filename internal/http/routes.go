package http

import (
	"github.com/gin-gonic/gin"
)

// 注册路由
func (s *Server) registerRoutes() {
	s.Router.GET(s.cfg.WebSocket.Path, s.handleWS)
	s.Router.Static("/static", "./static")
	s.Router.GET("/", func(ctx *gin.Context) {
		ctx.File("./static/index.html")
	})
	api := s.Router.Group(s.cfg.API.Prefix)
	{
		api.GET("/robots/", s.listDevices)
		api.GET("/robots/:id", s.getDevice)
		api.POST("/robots/:id/move", s.moveDevice)
		api.POST("/robots/:id/stop", s.stopDevice)
	}
}
