package config

// 中心服务配置
type HubConfig struct {
	MQTT MQTTConfig `mapstructure:"mqtt"`
	HTTP HTTPConfig `mapstructure:"http"`
	Log  LogConfig  `mapstructure:"log"`
}

// 机器人配置
type RobotConfig struct {
	MQTT  MQTTConfig          `mapstructure:"mqtt"`
	Robot RobotBehaviorConfig `mapstructure:"robot"`
	Log   LogConfig           `mapstructure:"log"`
}

// MQTT连接配置
type MQTTConfig struct {
	Schema        string      `mapstructure:"schema" validate:"oneof=tcp ssl ws wss"`
	Host          string      `mapstructure:"host" validate:"hostname|ip"`
	Port          int         `mapstructure:"port" validate:"min=1,max=65535"`
	Auth          bool        `mapstructure:"auth"`
	UserName      string      `mapstructure:"username" validate:"required_if=Auth true"`
	Password      string      `mapstructure:"password" validate:"required_if=Auth true"`
	ClientId      string      `mapstructure:"client_id"`
	CleanStart    bool        `mapstructure:"clean_start"`
	KeepAlive     uint16      `mapstructure:"keep_alive" validate:"min=1,max=65535"`
	SessionExpiry uint32      `mapstructure:"session_expiry" validate:"min=0"`
	Will          WillConfig  `mapstructure:"will"`
	DefaultQoS    byte        `mapstructure:"default_qos" validate:"oneof=0 1 2"`
	MaxPacketSize uint32      `mapstructure:"max_packet_size" validate:"min=256"`
	Retry         RetryConfig `mapstructure:"retry"`
}

// 遗嘱消息配置（元信息，不含 topic 和 payload）
type WillConfig struct {
	Enabled bool `mapstructure:"enabled"`
	QoS     byte `mapstructure:"qos" validate:"oneof=0 1 2"`
	Retain  bool `mapstructure:"retain"`
}

// 重试配置
type RetryConfig struct {
	ConnRetryMax  int `mapstructure:"conn_retry_max" validate:"min=0"`
	ConnRetryBase int `mapstructure:"conn_retry_base" validate:"min=1"`
}

// HTTP服务配置
type HTTPConfig struct {
	Host      string          `mapstructure:"host" validate:"hostname|ip"`
	Port      int             `mapstructure:"port" validate:"min=1,max=65535"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
	CORS      CORSConfig      `mapstructure:"cors"`
	API       APIConfig       `mapstructure:"api"`
}

// WebSocket配置
type WebSocketConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	Path            string `mapstructure:"path"`
	BroadcastBuffer int    `mapstructure:"broadcast_buffer" validate:"min=0"`
}

// 跨域资源共享配置
type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// API配置
type APIConfig struct {
	Prefix  string `mapstructure:"prefix"`
	Timeout int    `mapstructure:"timeout" validate:"min=1"`
}

// 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level" validate:"oneof=debug info warn error"`
	Format string `mapstructure:"format" validate:"oneof=json text"`
	Output string `mapstructure:"output" validate:"oneof=stdout stderr file"`
}

// 机器人行为配置
type RobotBehaviorConfig struct {
	InitialSpeed   float64       `mapstructure:"initial_speed" validate:"min=0"`
	InitialBattery float64       `mapstructure:"initial_battery" validate:"min=0,max=100"`
	Report         ReportConfig  `mapstructure:"report"`
	Battery        BatteryConfig `mapstructure:"battery"`
}

type ReportConfig struct {
	BatteryThreshold  float64 `mapstructure:"battery_threshold" validate:"min=0,max=100"`
	PositionThreshold float64 `mapstructure:"position_threshold" validate:"min=0"`
	SpeedThreshold    float64 `mapstructure:"speed_threshold" validate:"min=0"`
	IntervalMs        float64 `mapstructure:"interval_ms" validate:"min=1"`
}

type BatteryConfig struct {
	ChargingRate         float64 `mapstructure:"charging_rate" validate:"min=0,max=100"`
	MovingDrain          float64 `mapstructure:"moving_drain" validate:"min=0,max=100"`
	LowBatteryThreshold  float64 `mapstructure:"low_battery_threshold" validate:"min=0,max=100"`
	FullBatteryThreshold float64 `mapstructure:"full_battery_threshold" validate:"min=0,max=100"`
}
