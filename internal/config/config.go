package config

import (
	"github.com/masenocturnal/pipefire/internal/crypto"
	"github.com/masenocturnal/pipefire/internal/sftp"
	"github.com/spf13/viper"
)

// DbConnection stores connection information for the database
type DbConnection struct {
	// @todo pull from config
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Name     string `json:"name"`
	Timeout  string `json:"timeout"`
}

// HostConfig data structure that represent a valid configuration file
type HostConfig struct {
	LogLevel   string                           `json:"loglevel"`
	Background bool                             `json:"background"`
	Database   DbConnection                     `json:"database"`
	Sftp       map[string]sftp.Endpoint         `json:"sftp"`
	Crypto     map[string]crypto.ProviderConfig `json:"crypto"`
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
