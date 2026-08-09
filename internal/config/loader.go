package config

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// LoadHub 加载 Hub 配置文件并返回 HubConfig 实例
func LoadHub() (*HubConfig, error) {
	v := viper.New()
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
func LoadRobot() (*RobotConfig, error) {
	v := viper.New()
	v.SetConfigName("robot")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("configs")

	var cfg RobotConfig
	if err := load(v, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// load 读取 viper 配置并解析到目标结构体
func load(v *viper.Viper, target any) error {
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
