package config

func DefaultHubConfig() HubConfig {
	return HubConfig{
		Log:  defaultLogConfig(),
		HTTP: defaultHTTPConfig(),
		MQTT: defaultMQTTConfig(),
	}
}

func DefaultRobotConfig() RobotConfig {
	return RobotConfig{
		Log:   defaultLogConfig(),
		MQTT:  defaultMQTTConfig(),
		Robot: defaultRobotBehaviorConfig(),
	}
}

func defaultMQTTConfig() MQTTConfig {
	return MQTTConfig{
		Schema:        "tcp",
		Host:          "127.0.0.1",
		Port:          1883,
		Auth:          false,
		UserName:      "",
		Password:      make([]byte, 0),
		ClientId:      "",
		CleanStart:    true,
		KeepAlive:     60,
		SessionExpiry: 3600,
		Will: WillConfig{
			Enabled: false,
			QoS:     0,
			Retain:  false,
		},
		DefaultQoS:    0,
		MaxPacketSize: 1024*1024,
		Retry: RetryConfig{
			ConnRetryMax:  3,
			ConnRetryBase: 1,
		},
	}
}

func defaultHTTPConfig() HTTPConfig {
	return HTTPConfig{
		Host: "127.0.0.1",
		Port: 8080,
		WebSocket: WebSocketConfig{
			Enabled:         false,
			Path:            "/ws",
			BroadcastBuffer: 1024,
		},
		CORS: CORSConfig{
			Enabled:        false,
			AllowedOrigins: []string{},
		},
		API: APIConfig{
			Prefix:  "/api/v1",
			Timeout: 30,
		},
	}
}

func defaultLogConfig() LogConfig {
	return LogConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}
}

func defaultRobotBehaviorConfig() RobotBehaviorConfig {
	return RobotBehaviorConfig{
		InitialSpeed:   1.0,
		InitialBattery: 100.0,
		Report: ReportConfig{
			BatteryThreshold:  0.1,
			PositionThreshold: 0.1,
			SpeedThreshold:    0.1,
			IntervalMs:        1000.0,
		},
		Battery: BatteryConfig{
			ChargingRate:         1.0,
			MovingDrain:          0.1,
			LowBatteryThreshold:  20.0,
			FullBatteryThreshold: 100.0,
		},
	}
}
