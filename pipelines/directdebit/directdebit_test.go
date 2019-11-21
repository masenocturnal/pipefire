package directdebit

func getPipeline(tasksConfig *TasksConfig) (Pipeline, error) {
	// logEntry := log.WithField("test", "test")

	ddConfig := &PipelineConfig{}
	ddConfig.Tasks = *tasksConfig

	pipeline, err := New(ddConfig)

	return pipeline, err
}
