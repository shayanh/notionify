package notionify

import (
	"time"

	"github.com/spf13/viper"
)

type RootConfig struct {
	Web       WebConfig       `mapstructure:"web"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Research  ResearchConfig  `mapstructure:"research"`
	Recurring RecurringConfig `mapstructure:"recurring"`
}

type ResearchConfig struct {
	Dropbox DropboxConfig `mapstructure:"dropbox"`
	Notion  NotionConfig  `mapstructure:"notion"`
}

type RecurringConfig struct {
	Interval time.Duration `mapstructure:"interval"`
	Notion   NotionConfig  `mapstructure:"notion"`
}

type WebConfig struct {
	Addr string `mapstructure:"addr"`
}

type DropboxConfig struct {
	Token      string `mapstructure:"token"`
	RootFolder string `mapstructure:"rootFolder"`
}

type NotionConfig struct {
	Token      string `mapstructure:"token"`
	DatabaseID string `mapstructure:"databaseID"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func ReadConfig() (RootConfig, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return RootConfig{}, err
	}
	var config RootConfig
	err := viper.UnmarshalKey("config", &config)
	return config, err
}
