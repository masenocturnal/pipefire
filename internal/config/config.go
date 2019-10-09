package config

import (
	"fmt"
	micro "github.com/micro/go-config"
	"github.com/micro/go-config/source/env"
	"github.com/micro/go-config/source/file"
)

// WebserverConfig configuration for the webserver

// SessionConfig configuration for the session
type SFTPConnectionList struct {
	Connections   []SFTPConnection `json:"sftp_connections"`
}
// SFTP Connection is an instance of the SFTP Connection Details
type SFTPConnection struct {
	Host string `json:"host"`
	Key string `json:"key"`
	Name string `json:"name"`
	Password string `json:"password"`
	KeyPassword string `json:"key_password"`
}

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
	Database   DbConnection    `json:"database"`
	SFTPConnections SFTPConnectionList `json:"sftp_connections"`
	Background bool            `json:"background"`
	LogLevel   string          `json:"loglevel"`
}


// ReadApplicationConfig will load the application configuration from known places on the disk or environment
func ReadApplicationConfig(configFile string) (HostConfig, error) {

	conf := micro.NewConfig()
	// Load json file with encoder
	err := conf.Load(
		file.NewSource(file.WithPath(configFile)),
		// allow env overrides,
		// keys can't have _ as this is how it deals with nesting
		env.NewSource(),
	)
	var hostConfiguration HostConfig

	if err != nil {
		return hostConfiguration, err
	}

	errs := validate(conf)
	if len(errs) > 0 {
		return hostConfiguration, errs[0]
	}
	err = conf.Scan(&hostConfiguration)	

	return hostConfiguration, err
}

// Validate ensure we have some basic validation of the configuration
func validate(myconfig micro.Config) []error {
	required := [3]string{"webserver", "database", "session"}
	var errs []error

	// We need to do more error checking here but let's at least make an
	// attempt
	for _, entry := range required {
		var tmpMap map[string]string
		configValue := myconfig.Get(entry).StringMap(tmpMap)
		if configValue == nil {
			newErr := fmt.Errorf("Config is missing a definition for %s", entry)
			errs = append(errs, newErr)
		}
	}

	// check the ensure the log level works
	return errs
}
