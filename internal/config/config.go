package config

import (
	"github.com/masenocturnal/pipefire/pipelines/directdebit"
	"github.com/spf13/viper"
)

// @todo load dynamically

//Pipelines top level =pipeline configuration
type Pipelines struct {
	DirectDebit directdebit.Config `json:"directdebit"`
}

// HostConfig data structure that represent a valid configuration file
type HostConfig struct {
	LogLevel   string    `json:"loglevel"`
	Background bool      `json:"background"`
	Pipelines  Pipelines `json:"piplines"`
	// Sftp       map[string]sftp.Endpoint         `json:"sftp"`
	// Crypto     map[string]crypto.ProviderConfig `json:"crypto"`
}

// ReadApplicationConfig will load the application configuration from known places on the disk or environment
func ReadApplicationConfig(configName string) (*HostConfig, error) {

	// conf := micro.NewConfig()
	conf := viper.New()
	conf.SetConfigName(configName)
	conf.AddConfigPath("/etc/pipefire/")
	conf.AddConfigPath("../config/")
	conf.AutomaticEnv()

	err := conf.ReadInConfig()
	hostConfig := &HostConfig{}
	conf.Unmarshal(hostConfig)

	// @todo validation
	return hostConfig, err
}
