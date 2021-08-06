package main

// func getPipeline(tasksConfig *config.TasksConfig) (*ddPipeline, error) {
// 	// logEntry := log.WithField("test", "test")

// 	ddConfig := &config.PipelineConfig{}
// 	ddConfig.Tasks = make([]*config.TasksConfig, 1)
// 	ddConfig.Tasks[0] = tasksConfig

// 	pipeline, err := New(ddConfig)

// 	return pipeline, err
// }

// var configPath string = "../../config/"

// func setup(t *testing.T) (*config.PipelineConfig, error) {
// 	abs, err := filepath.Abs(configPath)
// 	if err != nil {
// 		return nil, err
// 	}

// 	hostConfig, err := config.ReadApplicationConfig(abs)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ddConfig := &config.PipelineConfig{}
// 	ddConfig.Tasks = make([]*config.TasksConfig, 1)

// 	// @todo make this dynamic
// 	err = hostConfig.UnmarshalKey("pipelines.directdebit", ddConfig)
// 	if err != nil {
// 		t.Fatal(err.Error())
// 	}
// 	return ddConfig, nil
// }
