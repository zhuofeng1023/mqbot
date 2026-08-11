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

	"github.com/Drunk6904/mqbot/internal/config"
	"github.com/Drunk6904/mqbot/internal/http"
	"github.com/Drunk6904/mqbot/internal/hub"
	"github.com/Drunk6904/mqbot/internal/mqtt"
	"github.com/Drunk6904/mqbot/protocol"
	"github.com/eclipse/paho.golang/paho"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// 常量 ===========================================================

var server *http.Server

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

	err = mqtt.SubscribeTopic(c, fmt.Sprintf(protocol.StatusTopic, "+"), 0)
	if err != nil {
		log.Fatalf("订阅状态主题失败\n")
	}
	// 启动web服务
	server = http.NewServer(&cfg.HTTP)
	server.MqttClient = c
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
	default:
		log.Printf("未知主题：%s\n", topic)
	}
	return true, nil
}

func handStatus(pr paho.PublishReceived) {
	// 解析状态数据
	var status protocol.StatusBody
	json.Unmarshal(pr.Packet.Payload, &status)
	device := &hub.Device{
		ID:       status.ID,
		State:    status.State,
		Battery:  status.Battery,
		X:        status.X,
		Y:        status.Y,
		Speed:    status.Speed,
		LastSeen: time.Now(),
		Online:   true,
	}
	// 更新注册表
	server.Registry.Update(device)
	// 广播给ws连接
	server.Broadcast(pr.Packet.Payload)

	log.Printf("[status] %s: %s\n", device.ID, device.State)
}
