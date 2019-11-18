package directdebit

import log "github.com/sirupsen/logrus"

func getPipeline(tasksConfig *TasksConfig) (Pipeline, error) {
	logEntry := log.WithField("test", "test")

	ddConfig := &PipelineConfig{}
	ddConfig.Tasks = *tasksConfig

	pipeline, err := New(ddConfig, logEntry)

	return pipeline, err
}
