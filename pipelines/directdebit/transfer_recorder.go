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
	LocalFileHash       string `gorm:"primary_key"`
	TransferredFileHash string
	LocalHostID         string
	RemoteHost          string `gorm:"primary_key"`
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
	var myCount []int = make([]int, 1)
	sql := fmt.Sprintf(`SELECT count(id) as noRecords
		FROM %s
		WHERE local_file_hash = ? and remote_host = ? and deleted_at IS NULL
		AND (
			remote_file_name <> '' 
			AND  remote_file_size > 0 
			AND (
				transferred_file_hash IS NOT NULL 
				OR transferred_file_hash <> ''
			)
		)`, rec.TableName())

	err := txn.Raw(sql, hash, remoteHost).Pluck("noRecords", &myCount).Error
	if err == nil && len(myCount) == 1 {
		t.log.Debugf("%d records found", myCount[0])
		y := (myCount[0] > 0)
		return y, err
	}
	return false, err
}

//RecordError Updates the transfer record in the database to record the error message
func (t TransferLog) RecordError(txn *gorm.DB, rec *Record) error {

	sql := "local_file_hash = ? and remote_host = ? and deleted_at IS NULL"

	result := txn.
		Model(rec).
		Where(sql, rec.LocalFileHash, rec.RemoteHost, rec.CorrelationID).
		UpdateColumns(Record{
			TransferEnd:    rec.TransferEnd,
			TransferErrors: rec.TransferErrors,
		})
	if err := result.Error; err != nil {

		t.log.Error(err.Error())
		return err
	}
	t.log.Debugf("Rows Updated %d ", result.RowsAffected)

	return nil
}

// Update updates the record
func (t TransferLog) Update(txn *gorm.DB, rec *Record) error {

	sql := "local_file_hash = ? AND remote_host = ? AND correlation_id = ?"
	result := txn.
		Model(rec).
		Where(sql, rec.LocalFileHash, rec.RemoteHost, rec.CorrelationID).
		UpdateColumns(Record{
			RemoteFileName:      rec.RemoteFileName,
			RemoteFilePath:      rec.RemoteFilePath,
			RemoteFileSize:      rec.RemoteFileSize,
			TransferredFileHash: rec.TransferredFileHash,
			TransferStart:       rec.TransferStart,
			TransferEnd:         rec.TransferEnd,
		})
	if err := result.Error; err != nil {

		t.log.Error(err.Error())
		return err
	}
	t.log.Debugf("Rows Updated %d ", result.RowsAffected)
	return nil
}
