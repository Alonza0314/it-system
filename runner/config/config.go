package config

import "time"

type Config struct {
	Runner RunnerIE `yaml:"runner" valid:"required"`
	Logger LoggerIE `yaml:"logger" valid:"required"`
}

type RunnerIE struct {
	Name string `yaml:"name" valid:"required"`

	ControllerIP   string `yaml:"controller_ip" valid:"required,ip"`
	ControllerPort int    `yaml:"controller_port" valid:"required,port"`

	TokenPath string `yaml:"token_path" valid:"required"`

	HeartbeatInterval time.Duration `yaml:"heartbeat_interval" valid:"required,gt=0"`

	HttpSenderChannelSize int `yaml:"http_sender_channel_size" valid:"required,gt=0"`
}

type LoggerIE struct {
	Level string `yaml:"level" valid:"required"`
}
