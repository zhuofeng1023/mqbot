package config

// 中心服务配置
type HubConfig struct {
	MQTT MQTTConfig `mapstructure:"mqtt" validate:"required"`
	HTTP HTTPConfig `mapstructure:"http" validate:"required"`
	Log  LogConfig  `mapstructure:"log" validate:"required"`
}

// 机器人配置
type RobotConfig struct {
	MQTT  MQTTConfig          `mapstructure:"mqtt" validate:"required"`
	Robot RobotBehaviorConfig `mapstructure:"robot" validate:"required"`
	Log   LogConfig           `mapstructure:"log" validate:"required"`
}

// MQTT连接配置
type MQTTConfig struct {
	Schema        string      `mapstructure:"schema" validate:"required,oneof=tcp ssl ws wss"`
	Host          string      `mapstructure:"host" validate:"required,hostname|ip"`
	Port          int         `mapstructure:"port" validate:"required,min=1,max=65535"`
	Auth          bool        `mapstructure:"auth" validate:""`
	UserName      string      `mapstructure:"username" validate:"required_if=Auth true"`
	Password      []byte      `mapstructure:"password" validate:"required_if=Auth true"`
	ClientId      string      `mapstructure:"client_id" validate:"required"`
	CleanStart    bool        `mapstructure:"clean_start" validate:""`
	KeepAlive     uint16      `mapstructure:"keep_alive" validate:"min=1,max=65535"`
	SessionExpiry uint32      `mapstructure:"session_expiry" validate:"min=0"`
	Will          WillMessage `mapstructure:"will" validate:"required"`
	DefaultQoS    byte        `mapstructure:"default_qos" validate:"oneof=0 1 2"`
	MaxPacketSize uint32      `mapstructure:"max_packet_size" validate:"min=256"`
	Retry         RetryConfig `mapstructure:"retry" validate:"required"`
}

// 遗嘱消息配置
type WillMessage struct {
	Enabled bool `mapstructure:"enabled" validate:""`
	QoS     byte `mapstructure:"qos" validate:"oneof=0 1 2"`
	Retain  bool `mapstructure:"retain" validate:""`
}

// 重试配置
type RetryConfig struct {
	ConnRetryMax  int `mapstructure:"conn_retry_max" validate:"required,min=0"`
	ConnRetryBase int `mapstructure:"conn_retry_base" validate:"required,min=1"`
}

// HTTP服务配置
type HTTPConfig struct {
	Host      string          `mapstructure:"host" validate:"required,hostname|ip"`
	Port      int             `mapstructure:"port" validate:"required,min=1,max=65535"`
	WebSocket WebSocketConfig `mapstructure:"websocket" validate:"required"`
	CORS      CORSConfig      `mapstructure:"cors" validate:"required"`
	API       APIConfig       `mapstructure:"api" validate:"required"`
}

// WebSocket配置
type WebSocketConfig struct {
	Enabled         bool   `mapstructure:"enabled" validate:""`
	Path            string `mapstructure:"path" validate:"required"`
	BroadcastBuffer int    `mapstructure:"broadcast_buffer" validate:"min=0"`
}

// 跨域资源共享配置
type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled" validate:""`
	AllowedOrigins []string `mapstructure:"allowed_origins" validate:"dive,url"`
}

// API配置
type APIConfig struct {
	Prefix  string `mapstructure:"prefix" validate:"required"`
	Timeout int    `mapstructure:"timeout" validate:"required,min=1"`
}

// 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level" validate:"required,oneof=debug info warn error"`
	Format string `mapstructure:"format" validate:"required,oneof=json text"`
	Output string `mapstructure:"output" validate:"required,oneof=stdout stderr file"`
}

// 机器人行为配置
type RobotBehaviorConfig struct {
	InitialSpeed   float64       `mapstructure:"initial_speed" validate:"required,min=0"`
	InitialBattery float64       `mapstructure:"initial_battery" validate:"required,min=0,max=100"`
	Report         ReportConfig  `mapstructure:"report" validate:"required"`
	Battery        BatteryConfig `mapstructure:"battery" validate:"required"`
}

type ReportConfig struct {
	BatteryThreshold  float64 `mapstructure:"battery_threshold" validate:"required,min=0,max=100"`
	PositionThreshold float64 `mapstructure:"position_threshold" validate:"required,min=0"`
	SpeedThreshold    float64 `mapstructure:"speed_threshold" validate:"required,min=0"`
	IntervalMs        float64 `mapstructure:"interval_ms" validate:"required,min=1"`
}

type BatteryConfig struct {
	ChargingRate         float64 `mapstructure:"charging_rate" validate:"required,min=0,max=100"`
	MovingDrain          float64 `mapstructure:"moving_drain" validate:"required,min=0,max=100"`
	LowBatteryThreshold  float64 `mapstructure:"low_battery_threshold" validate:"required,min=0,max=100"`
	FullBatteryThreshold float64 `mapstructure:"full_battery_threshold" validate:"required,min=0,max=100"`
}
