package mqtt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/paho"
	"github.com/google/uuid"
)

// 管理 请求-响应模式：发请求，等响应，超时控制
type Requester struct {
	client  *paho.Client
	pending map[string]chan []byte // correlation_id 等待响应的 channel
	mu      sync.Mutex
	timeout time.Duration // TODO
}

// 创建请求管理器
func NewRequester(client *paho.Client, timeout time.Duration) *Requester {
	return &Requester{
		client:  client,
		pending: make(map[string]chan []byte),
		timeout: timeout,
	}
}

// Request 发送请求并同步等待响应
// reqTopic: 请求主题（robot/{id}/req）
// respTopic: 响应主题（robot/{id}/resp），会写进 ResponseTopic 属性
// payload: 请求消息内容
func (r *Requester) Request(ctx context.Context, reqTopic, respTopic string, payload []byte) ([]byte, error) {
	corrID := uuid.New().String()
	ch := make(chan []byte, 1)

	// 注册
	r.mu.Lock()
	r.pending[corrID] = ch
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.pending, corrID)
		r.mu.Unlock()
	}()

	_, err := r.client.Publish(ctx, &paho.Publish{
		Topic:   reqTopic,
		QoS:     1,
		Payload: payload,
		Properties: &paho.PublishProperties{
			ResponseTopic:   respTopic,
			CorrelationData: []byte(corrID),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("发布请求失败: %w", err)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(r.timeout):
		return nil, fmt.Errorf("请求超时（%v）", r.timeout)
	}
}

// 处理接收到的响应，在 MTQQ 回调函数使用
func (r *Requester) HandlerResponse(corrID string, payload []byte) {
	r.mu.Lock()
	ch, ok := r.pending[corrID]
	r.mu.Unlock()
	if ok {
		ch <- payload
	}
}
