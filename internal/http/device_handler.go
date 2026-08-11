package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// listDevices 返回所有设备列表
func (s *Server) listDevices(c *gin.Context) {
	devices := s.Registry.List()
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "ok",
		Data: devices,
	})
}

// getDevice 返回指定设备详情
func (s *Server) getDevice(c *gin.Context) {
	id := c.Param("id")
	device, ok := s.Registry.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, Response{
			Code: 404,
			Msg:  "设备不存在",
		})
		return
	}
	c.JSON(http.StatusOK, Response{Code: 0, Msg: "ok", Data: device})
}
