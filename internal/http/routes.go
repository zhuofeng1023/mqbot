package http

import (
	"time"

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
		api.GET("/devices/:id/status", s.getDeviceStatus)
		api.GET("/metrics", s.getMetrics)
	}
}

// getMetrics 返回服务运行指标
func (s *Server) getMetrics(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"devices_online":      len(s.wsConns), // 简化：用 WS 连接数
		"msgs_received_total": s.MsgCount.Load(),
		"ws_clients":          s.WsClients.Load(),
		"uptime_seconds":      int(time.Since(s.startTime).Seconds()),
	})
}
