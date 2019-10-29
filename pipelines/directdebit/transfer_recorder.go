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
	FileSent(hash string, remoteHost string) (bool, error)
	// Close() error
}

//Record Maps to a row in the FileTransfers table
type Record struct {
	LocalFileName      string
	LocalFilePath      string
	LocalFileSize      uint
	RemoteFileName     string
	RemoteFilePath     string
	RemoteFileSize     uint
	RecipientName      string
	SenderName         string
	LocalFileHash      string
	TransferedFileHash string
	LocalHostID        string
	RemoteHost         string
	TransferStart      time.Time
	TransferEnd        time.Time
	TransferErrors     string
	CorrelationID      string
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
		txn.Rollback()
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

//FileSent Determines if a file has been
func (t TransferLog) FileSent(hash string, remoteHost string) (bool, error) {

	return true, nil
}

// //Close Closes the underlying connection
// func (t TransferLog) Close() (err error) {
// 	if err = t.Conn.Close(); err != nil {
// 		t.log.Error("Error closing database connection: %s", err.Error())
// 	}
// 	return err
// }
