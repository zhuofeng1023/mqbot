package hub

import (
	"sync"
	"time"
)

// DeviceRegistry 线程安全的设备注册表，用于存储和管理设备信息
type DeviceRegistry struct {
	mu      sync.RWMutex
	devices map[string]*Device
}

// NewDeviceRegistry 创建并返回一个新的设备注册表实例
func NewDeviceRegistry() *DeviceRegistry {
	return &DeviceRegistry{
		devices: make(map[string]*Device),
	}
}

// Update 将设备添加或更新到注册表中
func (r *DeviceRegistry) Update(d *Device) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices[d.ID] = d
}

// Remove 从注册表中删除指定ID的设备
func (r *DeviceRegistry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.devices, id)
}

// Get 根据设备ID获取设备，返回设备指针和是否存在
func (r *DeviceRegistry) Get(id string) (*Device, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.devices[id]
	return d, ok
}

// List 返回注册表中所有设备的列表
func (r *DeviceRegistry) List() []*Device {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Device, 0, len(r.devices))
	for _, d := range r.devices {
		list = append(list, d)
	}
	return list
}

// Count 返回注册表中设备的数量
func (r *DeviceRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.devices)
}

// CheckOffline 检查并标记超时的离线设备
func (r *DeviceRegistry) CheckOffline(timeout time.Duration) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var offline []string
	now := time.Now()
	for id, d := range r.devices {
		if d.Online && now.Sub(d.LastSeen) > timeout {
			d.Online = false
			offline = append(offline, id)
		}
	}
	return offline
}

