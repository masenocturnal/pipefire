package config

import (
	"github.com/masenocturnal/pipefire/pipelines/directdebit"
	"github.com/spf13/viper"
)

// @todo load dynamically

//Pipelines top level =pipeline configuration
type Pipelines struct {
	DirectDebit directdebit.PipelineConfig `json:"directdebit"`
}

// HostConfig data structure that represent a valid configuration file
type HostConfig struct {
	LogLevel   string    `json:"loglevel"`
	Background bool      `json:"background"`
	Pipelines  Pipelines `json:"piplines"`
}

// ReadApplicationConfig will load the application configuration from known places on the disk or environment
func ReadApplicationConfig(configName string) (*HostConfig, error) {

	// conf := micro.NewConfig()
	conf := viper.New()
	conf.SetConfigName(configName)
	conf.AddConfigPath("/etc/pipefire/")
	conf.AddConfigPath("../config/")
	conf.AddConfigPath("./")
	conf.AutomaticEnv()

	err := conf.ReadInConfig()
	if err != nil {
		return nil, err
	}
	hostConfig := &HostConfig{}
	err = conf.Unmarshal(hostConfig)

	// @todo validation
	return hostConfig, err
}
