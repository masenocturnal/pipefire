package config

import (
	"github.com/spf13/viper"
)

// HostConfig data structure that represent a valid configuration file
type HostConfig struct {
	LogLevel   string            `json:"loglevel"`
	Background bool              `json:"background"`
	Pipelines  map[string]string `json:"pipelines"`
}
type includeFile string

// ReadApplicationConfig will load the application configuration from known places on the disk or environment
func ReadApplicationConfig(paths ...string) (*viper.Viper, error) {

	conf := viper.New()
	conf.SetConfigName("pipefired")
	//conf.Set("Verbose", true)
	if len(paths) > 0 {
		for _, path := range paths {
			conf.AddConfigPath(path)
		}
	} else {
		conf.AddConfigPath("/etc/pipefire/")
		conf.AddConfigPath("../config/")
		conf.AddConfigPath("./")
	}
	conf.AutomaticEnv()

	err := conf.ReadInConfig()

	if err != nil {
		return nil, err
	}
	// hostConfig := &HostConfig{}
	// err = conf.Unmarshal(hostConfig)
	// conf.Debug()

	// @todo validation
	return conf, err
}
