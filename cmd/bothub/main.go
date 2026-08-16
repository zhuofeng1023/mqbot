package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/eclipse/paho.golang/paho"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/zhuofeng1023/mqbot/internal/config"
	"github.com/zhuofeng1023/mqbot/internal/http"
	"github.com/zhuofeng1023/mqbot/internal/hub"
	"github.com/zhuofeng1023/mqbot/internal/mqtt"
	"github.com/zhuofeng1023/mqbot/protocol"
)

// 常量 ===========================================================

var server *http.Server
var requester *mqtt.Requester

// Main ==========================================================

func main() {
	// 解析命令行参数
	configPath := pflag.String("config", "configs/bothub.yaml", "配置文件路径")
	_ = pflag.String("mqtt-host", "", "MQTT broker 地址（覆盖配置文件）")
	_ = pflag.Int("mqtt-port", 0, "MQTT broker 端口（覆盖配置文件）")
	_ = pflag.Int("http-port", 0, "HTTP 服务端口（覆盖配置文件）")
	pflag.Parse()

	// 绑定 flag 到 viper
	v := viper.New()
	v.BindPFlag("mqtt.host", pflag.Lookup("mqtt-host"))
	v.BindPFlag("mqtt.port", pflag.Lookup("mqtt-port"))
	v.BindPFlag("http.port", pflag.Lookup("http-port"))

	// 加载配置
	cfg, err := config.LoadHub(v, *configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// mqtt 服务
	c, err := mqtt.NewClient(&cfg.MQTT, mqtt.WithHandler(MsgHandler))

	if err != nil {
		log.Fatalf("创建 mqtt 客户端失败：%v\n", err)
	}

	// 创建请求管理器
	requester = mqtt.NewRequester(c, 5*time.Second)

	err = mqtt.SubscribeTopic(c, fmt.Sprintf(protocol.RespTopic, "+"), 1)
	if err != nil {
		log.Fatalf("订阅响应主题失败：%v\n", err)
	}

	err = mqtt.SubscribeTopic(c, fmt.Sprintf(protocol.StatusTopic, "+"), 0)
	if err != nil {
		log.Fatalf("订阅状态主题失败\n")
	}
	// 启动web服务
	server = http.NewServer(&cfg.HTTP)
	server.MqttClient = c
	server.Requester = requester
	go func() {
		if err = server.Start(); err != nil {
			log.Fatalf("启动 web服务失败：%v\n", err)
		}
	}()

	// 停止
	ic := make(chan os.Signal, 1)
	signal.Notify(ic, os.Interrupt, syscall.SIGTERM)
	<-ic
	if c != nil {
		err := c.Disconnect(&paho.Disconnect{ReasonCode: 0})
		if err != nil {
			log.Fatalf("发生错误: %s\n", err)
		}
	}
	os.Exit(0)
}

// MQTT 服务相关 =================================================

// mqtt 消息处理函数
func MsgHandler(pr paho.PublishReceived) (bool, error) {
	topic := pr.Packet.Topic
	switch {
	case strings.HasSuffix(topic, "/status"):
		handStatus(pr)
	case strings.HasSuffix(topic, "/resp"):
		handResponse(pr)
	default:
		log.Printf("未知主题：%s\n", topic)
	}
	return true, nil
}

func handResponse(pr paho.PublishReceived) {
	corrData := string(pr.Packet.Properties.CorrelationData)
	requester.HandlerResponse(corrData, pr.Packet.Payload)
}

func handStatus(pr paho.PublishReceived) {
	if server == nil {
		// 等待初始化完成
		deadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond) // 每 10ms 检查一次
			if server != nil {
				break // 初始化完成了，跳出循环继续执行
			}
		}
	}
	if server == nil {
		log.Printf("[warn] Server 未初始化，丢弃状态消息。Topic: %s", pr.Packet.Topic)
		return
	}
	id := protocol.DeviceIDFromTopic(pr.Packet.Topic)

	// 遗嘱消息：robot 下线，payload 只有 state 没有 id
	var status protocol.StatusBody
	json.Unmarshal(pr.Packet.Payload, &status)

	if status.State == protocol.StateOffline {
		server.Registry.Remove(id)
		// 构造带 id 的离线通知
		notify, _ := json.Marshal(protocol.StatusBody{ID: id, State: protocol.StateOffline})
		server.Broadcast(notify)
		log.Printf("[offline] %s 遗嘱下线", id)
		return
	}

	// 正常状态上报
	status.ID = id // robot 上报的 payload 不带 id，用 topic 取到的补上
	device := &hub.Device{
		ID:       id,
		State:    status.State,
		Battery:  status.Battery,
		X:        status.X,
		Y:        status.Y,
		Speed:    status.Speed,
		LastSeen: time.Now(),
		Online:   true,
	}
	server.Registry.Update(device)
	// 广播补全 id 后的状态，前端靠 id 匹配设备
	notify, _ := json.Marshal(status)
	server.Broadcast(notify)
	log.Printf("[status] %s: %s\n", id, device.State)
}
