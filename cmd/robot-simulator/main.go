package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/eclipse/paho.golang/paho"
	"github.com/spf13/pflag"
	"github.com/zhuofeng1023/mqbot/protocol"
)

func main() {
	count := pflag.Int("robots", 100, "模拟机器人数量")
	rate := pflag.Float64("rate", 2, "每个机器人上报频率(Hz)")
	host := pflag.String("host", "127.0.0.1", "MQTT broker 地址")
	port := pflag.Int("port", 1883, "MQTT broker 端口")
	duration := pflag.Int("duration", 60, "压测时长(秒)")
	maxConns := pflag.Int("max-conns", 50, "最大并发连接数(防止瞬间打满Broker)")
	pflag.Parse()

	sim := &Simulator{
		count:    *count,
		rate:     *rate,
		host:     *host,
		port:     *port,
		duration: time.Duration(*duration) * time.Second,
		maxConns: *maxConns,
	}

	fmt.Printf("启动压测: %d 个机器人, %.1f Hz/个, 总预期 %.0f msg/s, 最大并发连接 %d, 持续 %v\n",
		*count, *rate, float64(*count)**rate, *maxConns, sim.duration)

	sim.Run()
}

// Simulator 压测模拟器
type Simulator struct {
	count    int
	rate     float64
	host     string
	port     int
	duration time.Duration
	maxConns int

	sentCount      atomic.Int64
	connectedCount atomic.Int64
}

func (s *Simulator) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 优雅退出
	ic := make(chan os.Signal, 1)
	signal.Notify(ic, os.Interrupt, syscall.SIGTERM)

	// 统计 ticker：每秒打印一次
	statTicker := time.NewTicker(1 * time.Second)
	defer statTicker.Stop()

	var lastSent int64

	// 启动统计协程
	go func() {
		for range statTicker.C {
			sent := s.sentCount.Load()
			connected := s.connectedCount.Load()
			rate := sent - lastSent
			lastSent = sent
			fmt.Printf("[stats] 已连接: %d/%d, 总发送: %d, 当前速率: %d msg/s\n", connected, s.count, sent, rate)
		}
	}()

	// 启动 N 个模拟机器人
	var wg sync.WaitGroup
	interval := time.Duration(float64(time.Second) / s.rate)

	// 信号量控制并发连接数，避免 100 个 goroutine 瞬间发起连接导致 Broker 拒绝
	sem := make(chan struct{}, s.maxConns)

	for i := 0; i < s.count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量

			s.runRobot(ctx, id, interval)
		}(i)
	}

	// 等待结束
	select {
	case <-time.After(s.duration):
		fmt.Println("\n压测时长结束")
	case <-ic:
		fmt.Println("\n收到退出信号")
	}
	cancel()
	wg.Wait()

	// 输出汇总
	fmt.Printf("\n===== 压测汇总 =====\n")
	fmt.Printf("成功连接: %d/%d\n", s.connectedCount.Load(), s.count)
	fmt.Printf("总发送消息: %d\n", s.sentCount.Load())
}

// runRobot 模拟单个机器人的状态上报
func (s *Simulator) runRobot(ctx context.Context, id int, interval time.Duration) {
	botID := fmt.Sprintf("sim_%04d", id)

	// 1. TCP 连接
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", s.host, s.port))
	if err != nil {
		fmt.Printf("机器人 %s TCP连接失败: %v\n", botID, err)
		return
	}
	defer conn.Close()

	client := paho.NewClient(paho.ClientConfig{
		ClientID: botID,
		Conn:     conn,
	})

	// 2. MQTT 连接 (增加超时)
	connCtx, connCancel := context.WithTimeout(ctx, 5*time.Second)
	defer connCancel()

	ack, err := client.Connect(connCtx, &paho.Connect{
		ClientID:   botID,
		KeepAlive:  30,
		CleanStart: true,
		UsernameFlag: true,
		PasswordFlag: true, 
		Username:   "test001",
		Password:   []byte("test001"),
	})
	if err != nil {
		fmt.Printf("机器人 %s MQTT连接失败: %v\n", botID, err)
		return
	}

	// 3. 检查 Broker 是否拒绝连接 (打印详细原因)
	if ack.ReasonCode >= 0x80 {
		reason := "unknown"
		if ack.Properties != nil && ack.Properties.ReasonString != "" {
			reason = ack.Properties.ReasonString
		}
		fmt.Printf("机器人 %s 被Broker拒绝: code=0x%02X reason=%s\n", botID, ack.ReasonCode, reason)
		return
	}

	s.connectedCount.Add(1)
	defer client.Disconnect(&paho.Disconnect{ReasonCode: 0})

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 随机初始位置
	x := rand.Float64() * 100
	y := rand.Float64() * 100

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 模拟位置变化
			x += rand.Float64()*2 - 1
			y += rand.Float64()*2 - 1

			sendTime := time.Now()

			body := protocol.StatusBody{
				ID:      botID,
				X:       x,
				Y:       y,
				Battery: 80,
				State:   protocol.StateMoving,
				Speed:   1.0,
			}
			// 在 header 里塞发送时间戳，用于测延迟

			payload, _ := json.Marshal(body)

			// 4. 发布消息 (增加超时防阻塞，关闭 Retain 减轻 Broker 压力)
			pubCtx, pubCancel := context.WithTimeout(ctx, 2*time.Second)
			client.Publish(pubCtx, &paho.Publish{
				Topic:   fmt.Sprintf(protocol.StatusTopic, botID),
				QoS:     1,
				Payload: payload,
				Retain:  false,
				Properties: &paho.PublishProperties{
					User: paho.UserProperties{
						paho.UserProperty{Key: "send_ts", Value: fmt.Sprintf("%d", sendTime.UnixNano())},
					},
				},
			})
			pubCancel()

			s.sentCount.Add(1)
		}
	}
}
