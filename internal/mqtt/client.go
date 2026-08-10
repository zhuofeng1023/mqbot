package mqtt

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Drunk6904/mqbot/internal/config"
	"github.com/eclipse/paho.golang/paho"
)

type clientOptions struct {
	handler     func(paho.PublishReceived) (bool, error)
	willPayload []byte
}

type Option func(*clientOptions)

func WithHandler(h func(paho.PublishReceived) (bool, error)) Option {
	return func(co *clientOptions) {
		co.handler = h
	}
}

// WithWillPayload 设置遗嘱消息内容
func WithWillPayload(wp []byte) Option {
	return func(co *clientOptions) {
		co.willPayload = wp
	}
}

// NewClient 创建 MQTT 客户端
func NewClient(cfg *config.MQTTConfig, opts ...Option) (*paho.Client, error) {
	var client *paho.Client
	host := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	// 初始化默认选项
	options := &clientOptions{}
	// 依次应用所有传入的选项
	for _, opt := range opts {
		opt(options)
	}

	connRetryMax := cfg.Retry.ConnRetryMax
	for i := 0; i <= connRetryMax; i++ {
		// 如果是最后一次重试，返回错误(0,1,2 完毕后 3在这里直接返回 不会进行第四次连接)
		if i == connRetryMax {
			return nil, fmt.Errorf("连接失败，已达到最大重试次数 %d", connRetryMax)
		}

		cp := &paho.Connect{
			Username:     cfg.UserName,
			Password:     []byte(cfg.Password),
			UsernameFlag: cfg.Auth,
			PasswordFlag: cfg.Auth,

			ClientID:   cfg.ClientId,
			KeepAlive:  cfg.KeepAlive,
			CleanStart: cfg.CleanStart,

			Properties: &paho.ConnectProperties{
				MaximumPacketSize: &cfg.MaxPacketSize,
			},
		}
		if cfg.Will.Topic != "" {
			cp.WillMessage = &paho.WillMessage{
				Topic:   cfg.Will.Topic,
				QoS:     cfg.Will.QoS,
				Retain:  cfg.Will.Retain,
				Payload: options.willPayload,
			}
		}
		if cfg.SessionExpiry > 0 {
			cp.Properties.SessionExpiryInterval = &cfg.SessionExpiry
		}

		conn, err := net.Dial(cfg.Schema, host)
		if err != nil {
			log.Printf("连接 %s 失败 (尝试 %d/%d): %s\n", host, i+1, connRetryMax, err)
			continue
		}

		client = paho.NewClient(paho.ClientConfig{
			ClientID:          cfg.ClientId,
			Conn:              conn,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){options.handler},
		})

		cxt, cancel := context.WithTimeout(context.Background(), time.Second*10)
		ack, err := client.Connect(cxt, cp)
		cancel()

		if err == nil && ack.ReasonCode < 0x80 {
			log.Printf("连接成功 (尝试 %d/%d)\n", i+1, connRetryMax)
			break
		} else if err != nil {
			log.Printf("连接失败(%s) (尝试 %d/%d): %s\n", host, i+1, connRetryMax, err)
		} else if ack.ReasonCode >= 0x80 {
			log.Printf("连接失败 (尝试 %d/%d): 服务器返回错误码 %d - %s\n", i+1, connRetryMax, ack.ReasonCode, ack.Properties.ReasonString)
		}

		// 关闭 TCP 连接
		conn.Close()
		// 等待后再重试，防止触发限流
		if i < connRetryMax-1 {
			waitTime := time.Duration(cfg.Retry.ConnRetryBase) * time.Second
			if waitTime == 0 {
				waitTime = 1 * time.Second
			}
			log.Printf("等待 %v 后重试...\n", waitTime)
			time.Sleep(waitTime)
		}
	}
	return client, nil
}

// SubscribeTopic 订阅 MQTT 主题
func SubscribeTopic(c *paho.Client, topic string, qos byte) error {

	t := time.NewTicker(3 * time.Second)
	defer t.Stop()

	max := 3
	i := 1

	for range t.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		ps, err := c.Subscribe(ctx, &paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{Topic: topic, QoS: qos},
			},
		})

		if err != nil {
			log.Printf("订阅 %s 失败 (尝试 %d/%d): %v", topic, i, max, err)
		} else if len(ps.Reasons) == 0 {
			log.Printf("订阅 %s 失败 (尝试 %d/%d): 未收到响应", topic, i, max)
		} else if ps.Reasons[0] < 0x80 {
			log.Printf("订阅 %s 成功 (尝试 %d/%d)", topic, i, max)
			cancel()
			return nil
		} else {
			reason := "未知错误"
			if ps.Properties != nil && ps.Properties.ReasonString != "" {
				reason = ps.Properties.ReasonString
			}
			log.Printf("订阅 %s 被拒绝 (尝试 %d/%d): [0x%02X] %s", topic, i, max, ps.Reasons[0], reason)
			err = fmt.Errorf("reason code 0x%02X", ps.Reasons[0])
		}

		if i >= max {
			cancel()
			return fmt.Errorf("订阅 %s 失败: 已达到最大重试次数 %d", topic, max)
		}
		cancel()
		i++
	}
	return fmt.Errorf("订阅 %s 失败: ticker已停止", topic)
}
