package main

import (
	"fmt"

	"fhpbioguide/pkg/api"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	// Defaults for trigger/lock paths — must match fhpbioguide defaults.
	viper.SetDefault("sync.triggerFile", "/tmp/fhp_sync_trigger")
	viper.SetDefault("sync.lockFile", "/tmp/fhp_sync.lock")

	api := api.NewAPI()

	api.StartAPI()
}
