package directdebit

import (
	"fmt"
	"os"
)

// CleanUpConfig defines the configuration for the cleanup task
type CleanUpConfig struct {
	Paths []string `json:"paths"`
}

//cleanDirtyFiles removes all files from the directory
func (p ddPipeline) cleanDirtyFiles(config *CleanUpConfig) (errorList []error) {
	if len(config.Paths) > 0 {
		for _, file := range config.Paths {
			if err := os.RemoveAll(file); err != nil {
				errorList = append(errorList, fmt.Errorf("Can't remove %s : %s ", file, err.Error()))
			}
		}
		return
	}
	p.log.Warn("No paths to remove")
	return
}
