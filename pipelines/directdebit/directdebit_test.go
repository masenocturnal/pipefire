package directdebit

import (
	"path/filepath"
	"testing"

	"github.com/masenocturnal/pipefire/internal/config"
)

func getPipeline(tasksConfig *TasksConfig) (Pipeline, error) {
	// logEntry := log.WithField("test", "test")

	ddConfig := &PipelineConfig{}
	ddConfig.Tasks = tasksConfig

	pipeline, err := New(ddConfig)

	return pipeline, err
}

var configPath string = "../../config/"

func setup(t *testing.T) (*PipelineConfig, error) {
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}

	hostConfig, err := config.ReadApplicationConfig(abs)
	if err != nil {
		return nil, err
	}

	ddConfig := &PipelineConfig{}

	// @todo make this dynamic
	err = hostConfig.UnmarshalKey("pipelines.directdebit", ddConfig)
	if err != nil {
		t.Fatal(err.Error())
	}
	return ddConfig, nil
}
