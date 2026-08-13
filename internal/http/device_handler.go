package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/eclipse/paho.golang/paho"
	"github.com/gin-gonic/gin"
	"github.com/zhuofeng1023/mqbot/protocol"
)

type moveRequest struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

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

func (s *Server) moveDevice(c *gin.Context) {
	id := c.Param("id")

	// 检查设备是否存在
	if _, ok := s.Registry.Get(id); !ok {
		c.JSON(http.StatusNotFound, Response{Code: 404, Msg: "设备不存在"})
		return
	}

	// 解析参数
	var req moveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: "参数错误"})
		return
	}

	// 组装 MQTT 消息
	task := protocol.NewTaskMessage(protocol.TaskBody{
		Action: protocol.ActionMoveTo,
		Params: map[string]any{
			"x": strconv.FormatFloat(req.X, 'f', -1, 64),
			"y": strconv.FormatFloat(req.Y, 'f', -1, 64),
		},
	})

	payload, _ := json.Marshal(task)
	topic := fmt.Sprintf(protocol.TaskTopic, id)
	if err := s.publishToDevice(c.Request.Context(), topic, payload, 1); err != nil {
		log.Printf("[move] 下发失败 device=%s: %v", id, err)
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "指令下发失败"})
		return
	}
	c.JSON(http.StatusOK, Response{Code: 0, Msg: "指令已下发"})
}

func (s *Server) stopDevice(c *gin.Context) {
	id := c.Param("id")

	comm := protocol.NewCommandMessage(protocol.CommandBody{
		Action: protocol.ActionStop,
	})

	payload, _ := json.Marshal(comm)
	topic := fmt.Sprintf(protocol.CommandTopic, id)
	if err := s.publishToDevice(c.Request.Context(), topic, payload, 1); err != nil {
		log.Printf("[stop] 下发失败 device=%s: %v", id, err)
		c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: "指令下发失败"})
		return
	}
	c.JSON(http.StatusOK, Response{Code: 0, Msg: "指令已下发"})
}

// publishToDevice 向指定 topic 发布消息，校验 broker 的 ack
func (s *Server) publishToDevice(ctx context.Context, topic string, payload []byte, qos byte) error {
	pr, err := s.MqttClient.Publish(ctx, &paho.Publish{
		Topic:   topic,
		QoS:     qos,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("发布失败: %w", err)
	}
	if pr != nil && pr.ReasonCode >= 0x80 {
		return fmt.Errorf("发布被拒绝: code=%d", pr.ReasonCode)
	}
	return nil
}
