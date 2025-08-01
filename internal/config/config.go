package config

import (
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	Env         string `mapstructure:"env"`
	StoragePath string `mapstructure:"storage_path"`
	HTTPServer  `mapstructure:"http_server"`
}

type HTTPServer struct {
	Address     string        `mapstructure:"address"`
	Timeout     time.Duration `mapstructure:"timeout"`
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`
}

func Load() Config {
	viper.AddConfigPath("./config")
	viper.SetConfigName("config.example")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(err)
	}

	return cfg
}
