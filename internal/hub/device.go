package hub

import "time"

type Device struct {
    ID        string    // 设备 ID
    State     string    // 状态：idle / moving / charging / offline
    Battery   float64   // 电量
    X         float64   // 位置 X
    Y         float64   // 位置 Y
    Speed     float64   // 速度
    LastSeen  time.Time // 最后一次上报时间
    Online    bool      // 是否在线
}

