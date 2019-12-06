package directdebit

import (
	"testing"
)

var configPath string = "../config/pipefired.json"

// func setup(config string) (Pipeline, error) {

// hostConfig, err := config.ReadApplicationConfig("pipefired")
// if err != nil {
// 	return err
// }

// // @todo make this dynamic
// ddConfig := hostConfig.Pipelines.DirectDebit

// // create the dd pipeline
// directDebitPipeline, err := New(&ddConfig)
// if err != nil {
// 	log.Error(err.Error())
// 	os.Exit(1)
// }
// return directDebitPipeline, err
// }

func TestRecordEncryptionCreate(t *testing.T) {
	// pipeline, err := setup(configPath)
	// if err != nil {
	// 	t.Error(err.Error())
	// }

}

func TestRecordEncryptionUpdate(t *testing.T) {
}
