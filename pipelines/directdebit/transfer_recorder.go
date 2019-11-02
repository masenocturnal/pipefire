package directdebit

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

// TransferRecorder provides a mechanism to update the transfer status
type TransferRecorder interface {
	Create(txn *gorm.DB, rec *Record) error
	FileAlreadySent(txn *gorm.DB, localFileHash string, remoteHost string) (bool, error)
	// Close() error
}

//TableName sets the table name to TransferRecord
func (Record) TableName() string {
	return "TransferRecord"
}

//Record Maps to a row in the FileTransfers table
type Record struct {
	gorm.Model
	LocalFileName       string
	LocalFilePath       string
	LocalFileSize       int64
	RemoteFileName      string
	RemoteFilePath      string
	RemoteFileSize      int64
	RecipientName       string
	SenderName          string
	LocalFileHash       string
	TransferredFileHash string
	LocalHostID         string
	RemoteHost          string
	TransferStart       time.Time
	TransferEnd         time.Time
	TransferErrors      string
	CorrelationID       string
}

//TransferLog Stores a database log
type TransferLog struct {
	Conn *gorm.DB
	log  *log.Entry
}

// NewRecorder provides a service which records transfer records in the database
func NewRecorder(Conn *gorm.DB, log *log.Entry) *TransferLog {

	transferLog := &TransferLog{
		Conn: Conn,
		log:  log,
	}

	return transferLog
}

//Create  Creates a TransferRecord
func (t TransferLog) Create(txn *gorm.DB, rec *Record) error {
	if txn == nil {
		return fmt.Errorf("Create must be performed in a transaction")
	}

	if err := txn.Create(rec).Error; err != nil {
		return err
	}
	return nil
}

//GetRecordByHash Returns a record by hash
func (t TransferLog) GetRecordByHash(hash string) {

}

//GetRecordByFileName Returns a record based on the file name
func (t TransferLog) GetRecordByFileName(fileName string) {

}

//FileAlreadySent Determines if a file has been
func (t TransferLog) FileAlreadySent(txn *gorm.DB, hash string, remoteHost string) (bool, error) {
	var rec Record
	err := txn.Where("local_file_hash = ? and remote_host = ? and deleted_at IS NULL", hash, remoteHost).First(&rec).Error
	t.log.Debugf("remote File size %d", rec.RemoteFileSize)
	t.log.Debugf("remote FileName %s", rec.RemoteFileName)
	t.log.Debugf("remote Hash %s", rec.TransferredFileHash)
	return (rec.RemoteFileSize > 0 || rec.RemoteFileName != "" || rec.TransferredFileHash != ""), err
}

//RecordError Updates the transfer record in the database to record the error message
func (t TransferLog) RecordError(txn *gorm.DB, hash string, remoteHost string, errorMsg string) error {
	var rec Record
	if err := txn.Model(&rec).Update("TransferErrors", errorMsg).Error; err != nil {
		t.log.Error(err.Error())
		return err
	}
	return nil
}

// Update updates the record
func (t TransferLog) Update(txn *gorm.DB, rec *Record) error {

	if err := txn.Model(rec).Updates(rec).Error; err != nil {
		t.log.Error(err.Error())
		return err
	}
	return nil
}
