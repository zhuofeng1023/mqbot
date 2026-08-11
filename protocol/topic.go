package protocol

import "strings"

const (
	StatusTopic  = "robot/%s/status"
	TaskTopic    = "robot/%s/task"
	CommandTopic = "robot/%s/command"
)

// DeviceIDFromTopic 从 "robot/<id>/<action>" 格式的 topic 提取设备 ID
func DeviceIDFromTopic(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 {
		return ""
	}
	return parts[1]
}
