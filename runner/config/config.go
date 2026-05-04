package config

type Config struct {
	Runner RunnerIE `yaml:"runner" valid:"required"`
	Logger LoggerIE `yaml:"logger" valid:"required"`
}

type RunnerIE struct {
}

type LoggerIE struct {
	Level string `yaml:"level" valid:"required"`
}
