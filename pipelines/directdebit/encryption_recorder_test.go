package directdebit

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestRecordEncryptionCreate(t *testing.T) {
	pipelineConfig, err := setup(t)
	if err != nil {
		t.Error(err.Error())
	}

	db, err := connectToDb(pipelineConfig.Database)
	if err != nil {
		t.Error(err.Error())
	}

	recorder := NewEncryptionRecorder(db, log.WithField("test", "true"))

}

func TestRecordEncryptionUpdate(t *testing.T) {
}
