package hub

import "time"

// Device 是 hub 侧的设备视图，在 robot 上报的 StatusBody 基础上增加管理字段（LastSeen/Online）
type Device struct {
	ID       string    `json:"id"`              // 设备 ID
	State    string    `json:"state"`           // 状态：IDLE / MOVING / CHARGING / OFFLINE
	Battery  float64   `json:"battery"`         // 电量
	X        float64   `json:"x"`               // 位置 X
	Y        float64   `json:"y"`               // 位置 Y
	Speed    float64   `json:"speed"`           // 速度
	LastSeen time.Time `json:"last_seen"`       // 最后一次上报时间
	Online   bool      `json:"online"`          // 是否在线
}

