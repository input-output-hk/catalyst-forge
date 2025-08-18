package main

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// initViper configures Viper for config file, env, and key formatting.
func initViper(cfgFile string) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("foundry-api")
		viper.SetConfigType("toml")
		viper.AddConfigPath("/etc/foundry")
		viper.AddConfigPath("/etc")
		viper.AddConfigPath("$HOME/.config/foundry")
		viper.AddConfigPath(".")
	}

	viper.SetEnvPrefix("")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// Ensure critical keys resolve from environment during Unmarshal
	_ = viper.BindEnv("auth.bootstraptoken")
	_ = viper.BindEnv("auth.rbacseeddefaults")

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// loadConfigFromViper unmarshals Viper state into cfg and validates it.
func loadConfigFromViper() error {
	if err := viper.Unmarshal(&cfg, viper.DecodeHook(mapstructure.StringToTimeDurationHookFunc())); err != nil {
		return err
	}
	return cfg.Validate()
}
