package mqtt

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/zhuofeng1023/mqbot/internal/config"
)

// Publisher 抽象 MQTT 发布能力：*paho.Client 与 autopaho 连接管理器都满足该接口，
type Publisher interface {
	Publish(ctx context.Context, p *paho.Publish) (*paho.PublishResponse, error)
}

type clientOptions struct {
	handler     func(paho.PublishReceived) (bool, error)
	willTopic   string
	willPayload []byte
	subscriptions []paho.SubscribeOptions // 连接成功后需恢复的订阅
}

// Option 配置 MQTT 客户端的选项函数
type Option func(*clientOptions)

// WithHandler 设置消息回调
func WithHandler(h func(paho.PublishReceived) (bool, error)) Option {
	return func(co *clientOptions) {
		co.handler = h
	}
}

// WithWillTopic 设置遗嘱消息主题
func WithWillTopic(topic string) Option {
	return func(co *clientOptions) {
		co.willTopic = topic
	}
}

// WithWillPayload 设置遗嘱消息内容
func WithWillPayload(wp []byte) Option {
	return func(co *clientOptions) {
		co.willPayload = wp
	}
}

// WithSubscriptions 声明订阅列表：每次连接（含重连）成功后自动恢复
func WithSubscriptions(topic string, qos byte) Option {
	return func(co *clientOptions) {
		co.subscriptions = append(co.subscriptions, paho.SubscribeOptions{Topic: topic, QoS: qos})
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
		if cfg.Will.Enabled && options.willTopic != "" {
			cp.WillMessage = &paho.WillMessage{
				Topic:   options.willTopic,
				QoS:     cfg.Will.QoS,
				Retain:  cfg.Will.Retain,
				Payload: options.willPayload,
			}
		}
		if cfg.SessionExpiry > 0 {
			cp.Properties.SessionExpiryInterval = &cfg.SessionExpiry
		}
		expiry := uint32(3600)
		cp.WillProperties = &paho.WillProperties{
			MessageExpiry: &expiry,
		}
		conn, err := net.Dial(cfg.Schema, host)
		if err != nil {
			log.Printf("连接 %s 失败 (尝试 %d/%d): %s\n", host, i+1, connRetryMax, err)
			continue
		}

		// 收到消息时调用 handler；未设置则留空切片，消息会被忽略而非 panic
		var handlers []func(paho.PublishReceived) (bool, error)
		if options.handler != nil {
			handlers = []func(paho.PublishReceived) (bool, error){options.handler}
		}

		client = paho.NewClient(paho.ClientConfig{
			ClientID:          cfg.ClientId,
			Conn:              conn,
			OnPublishReceived: handlers,
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

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		ps, err := c.Subscribe(ctx, &paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{Topic: topic, QoS: qos},
			},
		})
		cancel()

		if err != nil {
			log.Printf("订阅 %s 失败 (尝试 %d/%d): %v", topic, i, max, err)
		} else if len(ps.Reasons) == 0 {
			log.Printf("订阅 %s 失败 (尝试 %d/%d): 未收到响应", topic, i, max)
		} else if ps.Reasons[0] < 0x80 {
			log.Printf("订阅 %s 成功 (尝试 %d/%d)", topic, i, max)
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
			return fmt.Errorf("订阅 %s 失败: 已达到最大重试次数 %d", topic, max)
		}
		i++

		<-t.C
	}
}

// NewAutoClient 创建带自动重连的 MQTT 客户端（autopaho 实现），供 hub 使用。
// 断线后自动重连并在每次连接成功时恢复 WithSubscriptions 声明的订阅，
// 解决 *paho.Client 单连接断开后 "no connection available" 永久失效的问题。
func NewAutoClient(ctx context.Context, cfg *config.MQTTConfig, opts ...Option) (*autopaho.ConnectionManager, error) {
	options := &clientOptions{}
	for _, opt := range opts {
		opt(options)
	}

	serverURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", cfg.Schema, cfg.Host, cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("解析 broker 地址失败: %w", err)
	}

	// 收到消息时调用 handler
	var handlers []func(paho.PublishReceived) (bool, error)
	if options.handler != nil {
		handlers = []func(paho.PublishReceived) (bool, error){options.handler}
	}

	// 重连退避：以配置的基准间隔固定等待（后续可改为指数退避）
	retryBase := time.Duration(cfg.Retry.ConnRetryBase) * time.Second
	if retryBase <= 0 {
		retryBase = 2 * time.Second
	}

	// 每次连接成功（含重连）后自动恢复订阅
	subs := options.subscriptions
	onConnectionUp := func(cm *autopaho.ConnectionManager, connack *paho.Connack) {
		log.Printf("MQTT 已连接 (server=%s)", serverURL.Host)
		if len(subs) == 0 {
			return
		}
		subCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := cm.Subscribe(subCtx, &paho.Subscribe{Subscriptions: subs}); err != nil {
			log.Printf("恢复 %d 个订阅失败: %v", len(subs), err)
		}
	}

	cc := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{serverURL},
		KeepAlive:                     cfg.KeepAlive,
		CleanStartOnInitialConnection: cfg.CleanStart,
		SessionExpiryInterval:         cfg.SessionExpiry,
		ConnectTimeout:                10 * time.Second,
		ReconnectBackoff:              func(attempt int) time.Duration { return retryBase },
		OnConnectionUp:                onConnectionUp,
		OnConnectError:                func(err error) { log.Printf("MQTT 连接失败，将继续重试: %v", err) },
		ConnectUsername:               cfg.UserName,
		ConnectPassword:               []byte(cfg.Password),
		WillMessage:                   willMessage(cfg, options),
	}
	// 未开启认证时清空用户名密码
	if !cfg.Auth {
		cc.ConnectUsername = ""
		cc.ConnectPassword = nil
	}
	// 内嵌的 paho.ClientConfig：客户端标识与消息回调
	cc.ClientID = cfg.ClientId
	cc.OnPublishReceived = handlers
	// 最大包限制是 CONNECT 属性，通过定制连接包下发
	if cfg.MaxPacketSize > 0 {
		maxPacket := cfg.MaxPacketSize
		cc.ConnectPacketBuilder = func(c *paho.Connect, u *url.URL) (*paho.Connect, error) {
			if c.Properties == nil {
				c.Properties = &paho.ConnectProperties{}
			}
			c.Properties.MaximumPacketSize = &maxPacket
			return c, nil
		}
	}

	cm, err := autopaho.NewConnection(ctx, cc)
	if err != nil {
		return nil, fmt.Errorf("创建 MQTT 连接失败: %w", err)
	}

	// 首次连接成功才算就绪（后续断线由 autopaho 自动恢复）
	awaitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := cm.AwaitConnection(awaitCtx); err != nil {
		return nil, fmt.Errorf("等待 MQTT 首次连接失败: %w", err)
	}

	return cm, nil
}

// willMessage 按配置组装遗嘱消息（未启用时返回 nil）
func willMessage(cfg *config.MQTTConfig, options *clientOptions) *paho.WillMessage {
	if !cfg.Will.Enabled || options.willTopic == "" {
		return nil
	}
	return &paho.WillMessage{
		Topic:   options.willTopic,
		QoS:     cfg.Will.QoS,
		Retain:  cfg.Will.Retain,
		Payload: options.willPayload,
	}
}

