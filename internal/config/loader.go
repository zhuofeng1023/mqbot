package config

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// LoadHub 加载 Hub 配置文件并返回 HubConfig 实例
func LoadHub(v *viper.Viper, configPath string) (*HubConfig, error) {
	if v == nil {
		v = viper.New()
	}
	if configPath == "" {
		configPath = "configs/bothub.yaml"
	}
	return LoadHubWithViper(v, configPath)
}

// LoadHubWithViper 使用已有的 viper 实例加载配置
func LoadHubWithViper(v *viper.Viper, configPath string) (*HubConfig, error) {
	setDefaultHub(v)
	v.SetConfigFile(configPath)

	var cfg HubConfig
	if err := load(v, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadRobot 加载 Robot 配置文件并返回 RobotConfig 实例
func LoadRobot(v *viper.Viper, configPath string) (*RobotConfig, error) {
	if configPath == "" {
		configPath = "configs/robot.yaml"
	}
	return LoadRobotWithViper(v, configPath)
}

// LoadRobotWithViper 使用已有的 viper 实例加载配置
func LoadRobotWithViper(v *viper.Viper, configPath string) (*RobotConfig, error) {
	setDefaultRobot(v)
	v.SetConfigFile(configPath)

	var cfg RobotConfig
	if err := load(v, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// load 读取 viper 配置并解析到目标结构体
func load(v *viper.Viper, target any) error {
	_ = godotenv.Load()

	v.SetEnvPrefix("MQBOT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := v.Unmarshal(target); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	if err := validate.Struct(target); err != nil {
		return fmt.Errorf("校验配置失败: %w", err)
	}

	return nil
}

func setDefaultHub(v *viper.Viper) {
	setDefaultMQTT(v)
	setDefaultHTTP(v)
	setDefaultLog(v)
}

func setDefaultRobot(v *viper.Viper) {
	setDefaultMQTT(v)
	setDefaultLog(v)
	setDefaultRobotBehavior(v)
}

func setDefaultMQTT(v *viper.Viper) {
	v.SetDefault("mqtt.schema", "tcp")
	v.SetDefault("mqtt.host", "127.0.0.1")
	v.SetDefault("mqtt.port", 1883)
	v.SetDefault("mqtt.auth", false)
	v.SetDefault("mqtt.username", "")
	v.SetDefault("mqtt.password", "")
	v.SetDefault("mqtt.client_id", "")
	v.SetDefault("mqtt.clean_start", true)
	v.SetDefault("mqtt.keep_alive", 60)
	v.SetDefault("mqtt.session_expiry", 3600)
	v.SetDefault("mqtt.will.enabled", false)
	v.SetDefault("mqtt.will.qos", 0)
	v.SetDefault("mqtt.will.retain", false)
	v.SetDefault("mqtt.default_qos", 0)
	v.SetDefault("mqtt.max_packet_size", 1024*1024)
	v.SetDefault("mqtt.retry.conn_retry_max", 3)
	v.SetDefault("mqtt.retry.conn_retry_base", 1)
}

func setDefaultHTTP(v *viper.Viper) {
	v.SetDefault("http.host", "127.0.0.1")
	v.SetDefault("http.port", 8080)
	v.SetDefault("http.websocket.enabled", false)
	v.SetDefault("http.websocket.path", "/ws")
	v.SetDefault("http.websocket.broadcast_buffer", 1024)
	v.SetDefault("http.cors.enabled", false)
	v.SetDefault("http.cors.allowed_origins", []string{})
	v.SetDefault("http.api.prefix", "/api/v1")
	v.SetDefault("http.api.timeout", 30)
}

func setDefaultLog(v *viper.Viper) {
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("log.output", "stdout")
}

func setDefaultRobotBehavior(v *viper.Viper) {
	v.SetDefault("robot.initial_speed", 1.0)
	v.SetDefault("robot.initial_battery", 100.0)
	v.SetDefault("robot.report.battery_threshold", 0.1)
	v.SetDefault("robot.report.position_threshold", 0.1)
	v.SetDefault("robot.report.speed_threshold", 0.1)
	v.SetDefault("robot.report.interval_ms", 1000.0)
	v.SetDefault("robot.battery.charging_rate", 1.0)
	v.SetDefault("robot.battery.moving_drain", 0.1)
	v.SetDefault("robot.battery.low_battery_threshold", 20.0)
	v.SetDefault("robot.battery.full_battery_threshold", 100.0)
}

// RobotConfigHolder 是线程安全的 RobotConfig 容器，支持热更新。
type RobotConfigHolder struct {
	mu            sync.RWMutex
	cfg           *RobotConfig
	v             *viper.Viper
	path          string
	debounceTimer *time.Timer
}

// Get 返回当前配置的快照（只读使用）。
func (h *RobotConfigHolder) Get() *RobotConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.cfg
}

// Watch 启动配置文件热更新监听。
// 只热更可热更字段（log / robot.report / robot.battery），
// 不可热更字段（mqtt / robot.initial_*）的变更会打 warning 提示需重启。
func (h *RobotConfigHolder) Watch() {
	h.v.WatchConfig()
	h.v.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("[config] 检测到配置文件变更: %s", e.Name)

		// 防抖：短时间内多次事件合并为一次热更
		if h.debounceTimer != nil {
			h.debounceTimer.Stop()
		}
		h.debounceTimer = time.AfterFunc(200*time.Millisecond, func() {
			h.reload()
		})
	})
}

// reload 重新加载配置并合并可热更字段。
func (h *RobotConfigHolder) reload() {
	var newCfg RobotConfig
	if err := h.v.Unmarshal(&newCfg); err != nil {
		log.Printf("[config] 热更失败：解析配置错误: %v", err)
		return
	}
	if err := validate.Struct(&newCfg); err != nil {
		log.Printf("[config] 热更失败：校验不通过: %v", err)
		return
	}

	h.mu.Lock()
	oldCfg := h.cfg

	// 检测不可热更字段是否变化，变化了打 warning
	needRestart := false
	if oldCfg.MQTT != newCfg.MQTT {
		needRestart = true
		log.Printf("[config] 警告: mqtt 配置变更需重启生效")
	}
	if oldCfg.Robot.InitialSpeed != newCfg.Robot.InitialSpeed ||
		oldCfg.Robot.InitialBattery != newCfg.Robot.InitialBattery {
		needRestart = true
		log.Printf("[config] 警告: robot.initial_* 配置变更需重启生效")
	}

	// 只合并可热更字段
	h.cfg.Log = newCfg.Log
	h.cfg.Robot.Report = newCfg.Robot.Report
	h.cfg.Robot.Battery = newCfg.Robot.Battery

	h.mu.Unlock()

	if needRestart {
		log.Printf("[config] 部分配置已热更，需重启的字段未生效")
	} else {
		log.Printf("[config] 配置热更完成")
	}
}

// LoadRobotWithWatch 加载 robot 配置并返回支持热更新的 holder。
func LoadRobotWithWatch(v *viper.Viper, configPath string) (*RobotConfigHolder, error) {
	cfg, err := LoadRobot(v, configPath)
	if err != nil {
		return nil, err
	}
	holder := &RobotConfigHolder{
		cfg:  cfg,
		v:    v,
		path: configPath,
	}
	return holder, nil
}
