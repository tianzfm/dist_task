package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	App      AppConfig      `toml:"app"`
	Database DatabaseConfig `toml:"database"`
	RocketMQ RocketMQConfig `toml:"rocketmq"`
	Log      LogConfig      `toml:"log"`
	Retry    RetryConfig    `toml:"retry"`
}

type AppConfig struct {
	Name string `toml:"app_name"`
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type DatabaseConfig struct {
	Host            string `toml:"host"`
	Port            int    `toml:"port"`
	Username        string `toml:"username"`
	Password        string `toml:"password"`
	Name            string `toml:"name"`
	MaxOpenConns    int    `toml:"max_open_conns"`
	MaxIdleConns    int    `toml:"max_idle_conns"`
	ConnMaxLifetime int    `toml:"conn_max_lifetime"`
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.Username, d.Password, d.Host, d.Port, d.Name)
}

type RocketMQConfig struct {
	NameServer    string `toml:"namesrv"`
	ProducerGroup string `toml:"producer_group"`
	ConsumerGroup string `toml:"consumer_group"`
}

type LogConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
	Output string `toml:"output"`
}

type RetryConfig struct {
	DefaultMaxAttempts int `toml:"default_max_attempts"`
	DefaultInterval    int `toml:"default_interval"`
}

var GlobalConfig *Config

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file failed: %w", err)
	}

	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, fmt.Errorf("parse config file failed: %w", err)
	}

	GlobalConfig = &cfg
	return &cfg, nil
}

func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(err)
	}
	return cfg
}
