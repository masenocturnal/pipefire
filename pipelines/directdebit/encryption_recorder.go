package directdebit

import (
	"fmt"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

// EncryptionRecorder provides a mechanism to update the transfer status
type EncryptionRecorder interface {
	Create(txn *gorm.DB, rec *EncryptionRecord) error
	FileAlreadySent(txn *gorm.DB, localFileHash string, remoteHost string) (bool, error)
	// Close() error
}

//TableName sets the table name to TransferRecord
func (EncryptionRecord) TableName() string {
	return "EncryptionRecord"
}

//EncryptionRecord Maps to a row in the FileTransfers table
type EncryptionRecord struct {
	gorm.Model
	LocalFileName     string
	LocalFilePath     string
	LocalFileSize     int64
	RecipientKey      string
	SigningKey        string
	LocalFileHash     string `gorm:"primary_key"`
	EncryptedFileHash string
	CorrelationID     string
}

//EncryptionLog Stores a database log
type EncryptionLog struct {
	Conn *gorm.DB
	log  *log.Entry
}

// NewEncryptionRecorder provides a service which records transfer records in the database
func NewEncryptionRecorder(Conn *gorm.DB, log *log.Entry) *TransferLog {

	transferLog := &TransferLog{
		Conn: Conn,
		log:  log,
	}

	return transferLog
}

//Create  Creates a TransferRecord
func (t EncryptionLog) Create(txn *gorm.DB, rec *EncryptionRecord) error {
	if txn == nil {
		return fmt.Errorf("Create must be performed in a transaction")
	}

	if err := txn.Create(rec).Error; err != nil {
		return err
	}
	return nil
}
