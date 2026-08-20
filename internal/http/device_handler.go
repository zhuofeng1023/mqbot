package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/eclipse/paho.golang/paho"
	"github.com/gin-gonic/gin"
	"github.com/zhuofeng1023/mqbot/internal/pkg/errcode"
	"github.com/zhuofeng1023/mqbot/internal/pkg/response"
	"github.com/zhuofeng1023/mqbot/protocol"
)

type moveRequest struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// listDevices 返回所有设备列表
func (s *Server) listDevices(c *gin.Context) {
	devices := s.Registry.List()
	response.Success(c, devices)
}

// getDevice 返回指定设备详情
func (s *Server) getDevice(c *gin.Context) {
	id := c.Param("id")
	device, ok := s.Registry.Get(id)
	if !ok {
		response.Fail(c, errcode.ERR_DEVICE_NOT_FOUND)
		return
	}
	response.Success(c, device)
}

// moveDevice 下发移动指令到指定设备
func (s *Server) moveDevice(c *gin.Context) {
	id := c.Param("id")

	// 检查设备是否存在
	if _, ok := s.Registry.Get(id); !ok {
		response.Fail(c, errcode.ERR_DEVICE_NOT_FOUND)
		return
	}

	// 解析参数
	var req moveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ERR_BAD_REQUEST)
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
		response.Fail(c, errcode.ERR_COMMAND_FAILED)
		return
	}
	response.Success(c, nil)
}

// stopDevice 下发停止指令到指定设备
func (s *Server) stopDevice(c *gin.Context) {
	id := c.Param("id")

	comm := protocol.NewCommandMessage(protocol.CommandBody{
		Action: protocol.ActionStop,
	})

	payload, _ := json.Marshal(comm)
	topic := fmt.Sprintf(protocol.CommandTopic, id)
	if err := s.publishToDevice(c.Request.Context(), topic, payload, 1); err != nil {
		log.Printf("[stop] 下发失败 device=%s: %v", id, err)
		response.Fail(c, errcode.ERR_COMMAND_FAILED)
		return
	}
	response.Success(c, nil)
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

// getDeviceStatus 通过 MQTT 请求-响应查询设备实时状态
func (s *Server) getDeviceStatus(ctx *gin.Context) {
	id := ctx.Param("id")

	// 构造请求
	req := protocol.NewRequestMessage(protocol.RequestBody{
		Action: protocol.ActionGetStatus,
	})
	payload, _ := json.Marshal(req)

	// 通过 MQTT 发请求等响应
	reqTopic := fmt.Sprintf(protocol.ReqTopic, id)
	respTopic := fmt.Sprintf(protocol.RespTopic, id)
	respData, err := s.Requester.Request(ctx.Request.Context(), reqTopic, respTopic, payload)
	if err != nil {
		response.Fail(ctx, errcode.ERR_DEVICE_TIMEOUT)
		return
	}

	// 解析响应
	var resp protocol.ResponseMessage
	if err := json.Unmarshal(respData, &resp); err != nil {
		response.Fail(ctx, errcode.ERR_DEVICE_RESPONSE_INVALID)
		return
	}

	response.Success(ctx, resp.Body.Data)
}

// intervalRe 轨迹聚合窗口白名单：1~5 位数字 + s/m/h 单位，防止任意字符串进 SQL
var intervalRe = regexp.MustCompile(`^[0-9]{1,5}[smh]$`)

// parseFromTo 解析查询参数 from/to（毫秒时间戳），缺省 to=now、from=now-1h
// 解析失败或范围非法时已写好错误响应，返回 ok=false
func parseFromTo(ctx *gin.Context) (from, to int64, ok bool) {
	now := time.Now().UnixMilli()

	// 解析 to，缺省为当前时间
	toStr := ctx.Query("to")
	if toStr == "" {
		to = now
	} else {
		var err error
		to, err = strconv.ParseInt(toStr, 10, 64)
		if err != nil {
			response.Fail(ctx, errcode.ERR_BAD_REQUEST)
			return 0, 0, false
		}
	}

	// 解析 from，缺省为 1 小时前
	fromStr := ctx.Query("from")
	if fromStr == "" {
		from = now - 3600*1000
	} else {
		var err error
		from, err = strconv.ParseInt(fromStr, 10, 64)
		if err != nil {
			response.Fail(ctx, errcode.ERR_BAD_REQUEST)
			return 0, 0, false
		}
	}

	// 校验时间
	if from >= to {
		response.Fail(ctx, errcode.ERR_INVALID_TIME_RANGE)
		return 0, 0, false
	}
	return from, to, true
}

func (s *Server) getDeviceHistory(ctx *gin.Context) {
	// 未启用存储组件时返回明确业务错误
	if s.Storage == nil {
		response.Fail(ctx, errcode.ERR_STORAGE_DISABLE)
		return
	}

	id := ctx.Param("id")
	if id == "" {
		response.Fail(ctx, errcode.ERR_BAD_REQUEST)
		return
	}

	from, to, ok := parseFromTo(ctx)
	if !ok {
		return
	}

	// 处理 limit 参数 (缺省 1000，上限 10000)
	limitStr := ctx.Query("limit")
	limit := int(1000)
	if limitStr != "" {
		parsedLimit, err := strconv.ParseInt(limitStr, 10, 64)
		if err != nil || parsedLimit <= 0 {
			response.Fail(ctx, errcode.ERR_BAD_REQUEST)
			return
		}
		// 范围钳制
		if parsedLimit > 10000 {
			limit = 10000
		} else {
			limit = int(parsedLimit)
		}
	}
	// 调用存储层查询数据
	points, err := s.Storage.QueryHistory(id, from, to, limit)
	if err != nil {
		log.Printf("[history] 查询失败 device=%s: %v", id, err)
		response.Fail(ctx, errcode.ERR_QUERY_FAILED)
		return
	}

	// 返回成功响应
	response.Success(ctx, gin.H{
		"id":     id,
		"points": points,
	})
}

// getDeviceTrack 查询设备轨迹（按时间窗口降采样，点太多时前端画线用）
func (s *Server) getDeviceTrack(ctx *gin.Context) {
	if s.Storage == nil {
		response.Fail(ctx, errcode.ERR_STORAGE_DISABLE)
		return
	}

	id := ctx.Param("id")

	from, to, ok := parseFromTo(ctx)
	if !ok {
		return
	}

	// interval 缺省 10s，白名单校验防注入
	interval := ctx.DefaultQuery("interval", "10s")
	if !intervalRe.MatchString(interval) {
		response.Fail(ctx, errcode.ERR_BAD_REQUEST)
		return
	}

	track, err := s.Storage.QueryTrack(id, from, to, interval)
	if err != nil {
		log.Printf("[track] 查询失败 device=%s: %v", id, err)
		response.Fail(ctx, errcode.ERR_QUERY_FAILED)
		return
	}

	response.Success(ctx, gin.H{"id": id, "points": track})
}

// 查询设备最新状态
func (s *Server) getDeviceLatest(ctx *gin.Context) {
	if s.Storage == nil {
		response.Fail(ctx, errcode.ERR_STORAGE_DISABLE)
		return
	}

	id := ctx.Param("id")

	point, err := s.Storage.QueryLatest(id)
	if err != nil {
		log.Printf("[latest] 查询失败 device=%s: %v", id, err)
		response.Fail(ctx, errcode.ERR_QUERY_FAILED)
		return
	}

	// 无数据时 point 为 nil，仍返回成功，point 字段为 null
	response.Success(ctx, gin.H{"id": id, "point": point})
}
