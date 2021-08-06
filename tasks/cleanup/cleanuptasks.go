package cleanup

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

// CleanUpConfig defines the configuration for the cleanup task
type CleanUpConfig struct {
	Paths   []string `json:"paths"`
	Enabled bool     `json:"enabled"`
}

//GetConfig for a an appropriately shaped json configuration string return a valid ArchiveConfig
func GetConfig(jsonText string) (*CleanUpConfig, error) {
	config := &CleanUpConfig{}

	err := json.Unmarshal([]byte(jsonText), config)

	return config, err

}

//cleanDirtyFiles removes all files from the directory
func CleanDirtyFiles(config *CleanUpConfig, l *logrus.Entry) (errorList []error) {
	if len(config.Paths) > 0 {
		for _, file := range config.Paths {
			if err := os.RemoveAll(file); err != nil {
				errorList = append(errorList, fmt.Errorf("Can't remove %s : %s ", file, err.Error()))
			}
		}
		return
	}
	l.Warn("No paths to remove")
	return
}
