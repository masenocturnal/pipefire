package directdebit

import (
	"github.com/google/uuid"
	"github.com/masenocturnal/pipefire/internal/crypto"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordEncryptionCreate(t *testing.T) {
	pipelineConfig, err := setup(t)
	if err != nil {
		t.Fatal(err.Error())
	}

	db, err := connectToDb(pipelineConfig.Database)
	if err != nil {
		t.Error(err.Error())
	}

	recorder := NewEncryptionRecorder(db, log.WithField("test", "true"))

	fname := "./testdata/encryption_recorder/record_create1.txt"
	f, err := os.Stat(fname)

	if err != nil {
		t.Error(err.Error())
	}

	absPath, _ := filepath.Abs(fname)
	hash, _ := hashFile(absPath)
	guid := uuid.New()

	txn := db.Begin()
	// create a new record
	rec := &EncryptionRecord{
		LocalFileName: f.Name(),
		LocalFilePath: absPath,
		LocalFileHash: hash,
		LocalFileSize: f.Size(),
		CorrelationID: guid.String(),
	}
	err = recorder.Create(txn, rec)
	if err != nil {
		t.Error(err.Error())
		txn.RollbackUnlessCommitted()
	}
	txn.Commit()

}

func TestRecordEncryptionUpdate(t *testing.T) {
	pipelineConfig, err := setup(t)
	if err != nil {
		t.Fatal(err.Error())
	}

	db, err := connectToDb(pipelineConfig.Database)
	if err != nil {
		t.Error(err.Error())
	}
	logger := log.WithField("test", "true")
	recorder := NewEncryptionRecorder(db, logger)

	fname := "./testdata/encryption_recorder/record_create1.txt"
	absPath, _ := filepath.Abs(fname)
	hash, _ := hashFile(absPath)
	rec, err := recorder.GetByHash(hash)
	if err != nil {
		t.Fatal(err)
	}

	//encrypt the file
	ANZconf := pipelineConfig.Tasks.EncryptFiles.Providers["anz"]
	provider := crypto.NewProvider(ANZconf, logger)

	encryptedFile, err := ioutil.TempFile("/tmp", "pipefire_test")
	if err != nil {
		t.Fatal(err.Error())
	}
	err = provider.EncryptFile(fname, encryptedFile.Name())

	if err != nil {
		t.Error(err)
		t.Fail()
	}
	efPath := encryptedFile.Name()
	_, err = os.Stat(efPath)
	if err != nil {
		t.Error(err)
		t.Fail()
	}

	encHash, _ := hashFile(efPath)
	rec.RecipientKey = ANZconf.EncryptionKey
	rec.EncryptedFileHash = encHash
	txn := db.Begin()
	recorder.Update(txn, rec)
	if x := txn.Commit(); x.Error != nil {
		t.Error(x.Error)
		t.Fail()
	}

}
