package protocol

import "strings"

// MQTT 主题命名约定：robot/<设备ID>/<动作>，如 robot/bot_001/status
const (
	StatusTopic  = "robot/%s/status"  // 状态上报
	TaskTopic    = "robot/%s/task"    // 任务下发
	CommandTopic = "robot/%s/command" // 实时指令
	ReqTopic     = "robot/%s/req"     // 请求-响应：请求主题
	RespTopic    = "robot/%s/resp"    // 请求-响应：响应主题
)

// DeviceIDFromTopic 从 "robot/<id>/<action>" 格式的 topic 提取设备 ID
func DeviceIDFromTopic(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 {
		return ""
	}
	return parts[1]
}
