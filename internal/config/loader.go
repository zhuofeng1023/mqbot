package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// LoadHub 加载 Hub 配置文件并返回 HubConfig 实例
func LoadHub() (*HubConfig, error) {
	return LoadHubWithViper(viper.New())
}

// LoadHubWithViper 使用已有的 viper 实例加载配置
func LoadHubWithViper(v *viper.Viper) (*HubConfig, error) {
	setDefaultHub(v)
	v.SetConfigName("bothub")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("configs")

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
