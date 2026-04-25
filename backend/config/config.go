package config

import "time"

type Config struct {
	Backend BackendIE `yaml:"backend" valid:"required"`
	Logger  LoggerIE  `yaml:"logger" valid:"required"`
}

type BackendIE struct {
	Username string `yaml:"username" valid:"required"`
	Password string `yaml:"password" valid:"required"`

	Port int `yaml:"port" valid:"required"`

	JWT JWTIE `yaml:"jwt" valid:"required"`

	FrontendFilePath string `yaml:"frontendFilePath" valid:"required"`

	DBPath string `yaml:"dbPath" valid:"required"`
}

type JWTIE struct {
	Secret    string        `yaml:"secret" valid:"required"`
	ExpiresIn time.Duration `yaml:"expiresIn" valid:"required"`
}

type LoggerIE struct {
	Level string `yaml:"level" valid:"required"`
}
