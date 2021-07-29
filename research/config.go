package research

import "github.com/spf13/viper"

type AppConfig struct {
	Web     WebConfig     `mapstructure:"web"`
	Dropbox DropboxConfig `mapstructure:"dropbox"`
	Notion  NotionConfig  `mapstructure:"notion"`
	Redis   RedisConfig   `mapstructure:"redis"`
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

func ReadConfig() (AppConfig, error) {
	viper.SetConfigName("research-config")
	viper.AddConfigPath("./config")
	if err := viper.ReadInConfig(); err != nil {
		return AppConfig{}, err
	}
	var config AppConfig
	err := viper.UnmarshalKey("config", &config)
	return config, err
}
