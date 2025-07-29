package main

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Port        int           `mapstructure:"PORT"`
	NumWorkers  int           `mapstructure:"NUM_WORKERS"`
	QueueSize   int           `mapstructure:"QUEUE_SIZE"`
	MaxRetries  int           `mapstructure:"MAX_RETRIES"`
	HttpTimeout time.Duration `mapstructure:"HTTP_TIMEOUT"`
}

func init() {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	viper.SetDefault("PORT", 8080)
	viper.SetDefault("NUM_WORKERS", 3)
	viper.SetDefault("QUEUE_SIZE", 100)
	viper.SetDefault("MAX_RETRIES", 3)
	viper.SetDefault("HTTP_TIMEOUT", "30s")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}

func LoadConfig() (Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unable to decode config: %w", err)
	}
	return cfg, nil
}
