package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Drunk6904/mqbot/internal/config"
	"github.com/Drunk6904/mqbot/internal/mqtt"
	"github.com/Drunk6904/mqbot/internal/robot"
	"github.com/Drunk6904/mqbot/protocol"
	"github.com/eclipse/paho.golang/paho"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type RoBot struct {
	protocol.StatusBody
}

// 状态快照结构体
type stateSnapshot struct {
	Battery float64
	State   string
	X       float64
	Y       float64
	Speed   float64
}

var (
	selfBot       RoBot
	currentCancel context.CancelFunc
	stateMutex    sync.Mutex
	lastReported  stateSnapshot
	cfgHolder     *config.RobotConfigHolder
)

func main() {
	configPath := pflag.String("config", "configs/robot.yaml", "配置文件路径")
	_ = pflag.String("server", "", "MQTT broker 地址（覆盖配置文件）")
	_ = pflag.Int("port", 0, "MQTT broker 端口（覆盖配置文件）")
	_ = pflag.String("id", "", "客户端 ID（覆盖配置文件）")
	pflag.Parse()

	v := viper.New()
	v.BindPFlag("mqtt.host", pflag.Lookup("server"))
	v.BindPFlag("mqtt.port", pflag.Lookup("port"))
	v.BindPFlag("mqtt.client_id", pflag.Lookup("id"))

	// 加载配置（支持热更新）
	holder, err := config.LoadRobotWithWatch(v, *configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	cfgHolder = holder
	// 启动配置文件监听
	cfgHolder.Watch()

	cfg := cfgHolder.Get()

	// 从配置初始化机器人状态
	selfBot = RoBot{protocol.StatusBody{
		ID:      cfg.MQTT.ClientId,
		Speed:   cfg.Robot.InitialSpeed,
		State:   protocol.StateIdle,
		Battery: cfg.Robot.InitialBattery,
	}}
	reportInterval := time.Duration(cfg.Robot.Report.IntervalMs) * time.Millisecond

	// 创建MQTT客户端
	c, err := mqtt.NewClient(&cfg.MQTT,
		mqtt.WithHandler(handMsg),
		mqtt.WithWillTopic(fmt.Sprintf(protocol.StatusTopic, cfg.MQTT.ClientId)),
		mqtt.WithWillPayload(getStateOffline()),
	)
	
	// 错误处理
	if err != nil {
		log.Fatalf("创建MQTT客户端失败：%s\n", err)
	}

	// 注册信号处理函数，用于在程序结束时断开MQTT连接
	ic := make(chan os.Signal, 1)
	signal.Notify(ic, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ic
		if c != nil {
			err := c.Disconnect(&paho.Disconnect{ReasonCode: 0})
			if err != nil {
				log.Fatalf("断开连接时发生错误：%s", err)
			}
		}
		os.Exit(0)
	}()

	botID := selfBot.ID

	// 订阅
	err = mqtt.SubscribeTopic(c, fmt.Sprintf(protocol.TaskTopic, botID), 2)
	if err != nil {
		log.Fatalf("订阅频道发生错误：%s\n", err)
	}

	err = mqtt.SubscribeTopic(c, fmt.Sprintf(protocol.CommandTopic, botID), 2)
	if err != nil {
		log.Fatalf("订阅频道发生错误：%s\n", err)
	}

	// 发送连接建立成功消息
	sendState(c, botID)

	t := time.NewTicker(reportInterval)
	defer t.Stop()
	for range t.C {
		robotCfg := cfgHolder.Get().Robot

		stateMutex.Lock()
		if selfBot.State == protocol.StateCharging {
			selfBot.Battery += robotCfg.Battery.ChargingRate
		}
		if selfBot.Battery <= robotCfg.Battery.LowBatteryThreshold && selfBot.State != protocol.StateCharging {
			selfBot.State = protocol.StateCharging
			currentCancel()
		}
		if selfBot.State == protocol.StateCharging && selfBot.Battery > robotCfg.Battery.FullBatteryThreshold {
			selfBot.State = protocol.StateIdle
		}
		if selfBot.State == protocol.StateMoving {
			selfBot.Battery -= robotCfg.Battery.MovingDrain
		}
		stateMutex.Unlock()
		sendState(c, botID)
	}
}

func getStateOffline() []byte {
	msg, err := json.Marshal(protocol.StatusBody{
		State: protocol.StateOffline,
	})
	if err != nil {
		log.Printf("遗嘱消息建立失败：%s\n", err)
		return nil
	}
	return msg
}

func sendState(c *paho.Client, botID string) {
	stateMutex.Lock()
	if !shouldReportLocked() {
		stateMutex.Unlock()
		return
	}
	snap := selfBot.StatusBody
	updateLastReportedLocked()
	stateMutex.Unlock()

	msg, err := json.Marshal(snap)
	if err != nil {
		log.Printf("报备状态时，解析状态发生错误：%s\n", err)
		return
	}

	props := &paho.PublishProperties{}
	props.User.Add("botId", botID)
	cp := &paho.Publish{
		Topic:      fmt.Sprintf(protocol.StatusTopic, botID),
		QoS:        0,
		Payload:    msg,
		Properties: props,
	}
	if _, err = c.Publish(context.Background(), cp); err != nil {
		log.Printf("报备状态时，发布消息发生错误：%s\n", err)
	}
}

func shouldReportLocked() bool {
	reportCfg := cfgHolder.Get().Robot.Report

	// 电量变化超过阈值即上报
	if math.Abs(selfBot.Battery-lastReported.Battery) > reportCfg.BatteryThreshold {
		return true
	}

	// 状态机变化上报
	if selfBot.State != lastReported.State {
		return true
	}

	// 位置变化超过阈值
	if math.Abs(selfBot.X-lastReported.X) > reportCfg.PositionThreshold ||
		math.Abs(selfBot.Y-lastReported.Y) > reportCfg.PositionThreshold {
		return true
	}

	// 速度变化超过阈值
	if math.Abs(selfBot.Speed-lastReported.Speed) > reportCfg.SpeedThreshold {
		return true
	}

	return false
}

func updateLastReportedLocked() {
	lastReported = stateSnapshot{
		Battery: selfBot.Battery,
		State:   selfBot.State,
		X:       selfBot.X,
		Y:       selfBot.Y,
		Speed:   selfBot.Speed,
	}
}

// func calculateHash(msg []byte) string {
// 	h := fnv.New32a()
// 	h.Write(msg)
// 	hashBytes := h.Sum(nil)
// 	return hex.EncodeToString(hashBytes)
// }

// 回调函数，对接收的消息进行处理
func handMsg(pr paho.PublishReceived) (bool, error) {
	switch {
	case strings.HasSuffix(pr.Packet.Topic, "/task"):
		handTask(pr)
	case strings.HasSuffix(pr.Packet.Topic, "/command"):
		handCommand(pr)
	}
	return true, nil
}
func handTask(pr paho.PublishReceived) {
	var data protocol.TaskMessage
	err := json.Unmarshal(pr.Packet.Payload, &data)
	if err != nil {
		log.Printf("处理MQTT消息失败: %+v", err)
	}
	switch data.Body.Action {
	case protocol.ActionMoveTo:
		handleMoveTo(data)
	}
}

// handCommand 处理接收到的MQTT命令消息
func handCommand(pr paho.PublishReceived) {
	var data protocol.CommandMessage
	err := json.Unmarshal(pr.Packet.Payload, &data)
	if err != nil {
		log.Printf("处理MQTT消息(command)失败：%+v", err)
	}
	switch data.Body.Action {
	case protocol.ActionStop:
		currentCancel()
	case protocol.ActionSetSpeed:
		selfBot.Speed = protocol.FloatParam(data.Body.Params, "speed", 1.0)
	}
}

func handleMoveTo(data protocol.TaskMessage) {
	stateMutex.Lock()

	// 如果有旧的任务 结束
	if currentCancel != nil {
		currentCancel()
	}
	selfBot.TaskID = data.Body.TaskID
	stateMutex.Unlock()

	log.Printf("开始处理移动指令，当前位置: x=%.3f, y=%.3f", selfBot.X, selfBot.Y)
	// 获取x和y的字符串值
	xStr := protocol.StringParam(data.Body.Params, "x", fmt.Sprintf("%.3f", selfBot.X))
	yStr := protocol.StringParam(data.Body.Params, "y", fmt.Sprintf("%.3f", selfBot.Y))
	log.Printf("目标位置: x=%s, y=%s", xStr, yStr)

	// 将字符串转换为float64
	x, err := strconv.ParseFloat(xStr, 64)
	if err != nil {
		log.Printf("解析x坐标失败: %v", err)
		return
	}

	y, err := strconv.ParseFloat(yStr, 64)
	if err != nil {
		log.Printf("解析y坐标失败: %v", err)
		return
	}

	ctx, can := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "task_id", data.Body.TaskID)
	currentCancel = can

	// 调用MoveTo函数
	log.Printf("开始移动到目标位置: x=%.3f, y=%.3f", x, y)
	go func() {
		stateMutex.Lock()
		if selfBot.TaskID == data.Body.TaskID {
			selfBot.State = protocol.StateMoving
		}
		stateMutex.Unlock()

		robot.MoveTo(ctx, &selfBot.StatusBody, x, y)

		stateMutex.Lock()
		if selfBot.TaskID == data.Body.TaskID {
			selfBot.State = protocol.StateIdle
		}
		stateMutex.Unlock()
	}()
}
