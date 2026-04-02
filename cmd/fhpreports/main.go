package main

import (
	"fmt"

	"fhpbioguide/pkg/api"
	"fhpbioguide/pkg/logger"

	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	viper.SetDefault("sync.triggerFile", "/tmp/fhp_sync_trigger")
	viper.SetDefault("sync.lockFile", "/tmp/fhp_sync.lock")
	viper.SetDefault("log.verbose", false)
	viper.SetDefault("log.file", "fhpreports.log")
	viper.SetDefault("log.maxSizeMB", 50)
	viper.SetDefault("log.maxBackups", 5)
	viper.SetDefault("log.maxAgeDays", 30)

	cfg := logger.Config{
		Verbose:    viper.GetBool("log.verbose"),
		File:       viper.GetString("log.file"),
		MaxSizeMB:  viper.GetInt("log.maxSizeMB"),
		MaxBackups: viper.GetInt("log.maxBackups"),
		MaxAgeDays: viper.GetInt("log.maxAgeDays"),
	}
	appLog := logger.New(cfg)

	appLog.Info("starting", "app", "fhpreports")
	appLog.Info("log output", "file", cfg.File, "maxSizeMB", cfg.MaxSizeMB, "maxBackups", cfg.MaxBackups, "maxAgeDays", cfg.MaxAgeDays, "verbose", cfg.Verbose)

	server := api.NewAPI(appLog)
	server.StartAPI()
}
