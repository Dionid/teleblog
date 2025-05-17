package main

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Env              string `mapstructure:"ENV"`
	Port             int    `mapstructure:"PORT"`
	AppVersion       string `mapstructure:"APP_VERSION"`
	TelegramBotToken string `mapstructure:"TELEGRAM_BOT_TOKEN"`
	UserId           string `mapstructure:"USER_ID"`
	DisableBot       bool   `mapstructure:"DISABLE_BOT"`
}

// Call to load the variables from env
func initConfig() (*Config, error) {
	// # Read os env
	viper.AutomaticEnv()

	// # Tell viper the path/location of your env file. If it is root just add "."
	viper.AddConfigPath(".")

	viper.SetDefault("PORT", 8080)

	// # Tell viper the name of your file
	viper.SetConfigName("app")

	// # Tell viper the type of your file
	viper.SetConfigType("env")

	// # Viper reads all the variables from env file and log error if any found
	if err := viper.ReadInConfig(); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower("Not Found in")) {
			return nil, err
		}
	}

	config := &Config{}

	// # Viper unmarshals the loaded env varialbes into the struct
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	// TODO: Change somehow
	// # Load into env
	os.Setenv("APP_VERSION", config.AppVersion)

	return config, nil
}
