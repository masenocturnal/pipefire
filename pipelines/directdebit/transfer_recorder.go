package directdebit

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

// TransferRecorder provides a mechanism to update the transfer status
type TransferRecorder interface {
	Create(txn *gorm.DB, rec *TransferRecord) error
	FileAlreadySent(txn *gorm.DB, localFileHash string, remoteHost string) (bool, error)
	// Close() error
}

//TableName sets the table name to TransferRecord
func (TransferRecord) TableName() string {
	return "TransferRecord"
}

//TransferRecord Maps to a row in the FileTransfers table
type TransferRecord struct {
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

// NewTransferRecorder provides a service which records transfer records in the database
func NewTransferRecorder(Conn *gorm.DB, log *log.Entry) *TransferLog {

	transferLog := &TransferLog{
		Conn: Conn,
		log:  log,
	}

	return transferLog
}

//Create  Creates a TransferRecord
func (t TransferLog) Create(txn *gorm.DB, rec *TransferRecord) error {
	if txn == nil {
		return fmt.Errorf("Create must be performed in a transaction")
	}

	if err := txn.Create(rec).Error; err != nil {
		return err
	}
	return nil
}

//RecordError Updates the transfer record in the database to record the error message
func (t TransferLog) RecordError(txn *gorm.DB, rec *TransferRecord) error {

	sql := "local_file_hash = ? and remote_host = ? and correlation_id = ?"

	result := txn.
		Model(rec).
		Where(sql, rec.LocalFileHash, rec.RemoteHost, rec.CorrelationID).
		UpdateColumns(TransferRecord{
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

//AvailableToSend Represents a row of files
//which have been encrypted and are available to send
type AvailableToSend struct {
	plaintextFileHash   string
	fileToTransfer      string
	fileToTransferHash  string
	remoteHost          string
	transferredFileHash string
	remoteFileSize      int
}

//FileAlreadySent Determines if a file has been
func (t TransferLog) FileAlreadySent(txn *gorm.DB, hash string, remoteHost string) (bool, error) {

	//tr := &TransferRecord{}
	// er := &EncryptionRecord{}

	sql := `
	SELECT 
		er.local_file_hash as plaintext_file_hash
		,tr.local_file_name as file_to_transfer
		,tr.local_file_hash as file_to_transfer_hash
		,tr.remote_host
		,tr.transferred_file_hash    
		,tr.remote_file_size
	FROM 
		EncryptionRecord er   
		LEFT JOIN TransferRecord tr ON tr.local_file_hash = er.encrypted_file_hash
	WHERE 
		er.local_file_hash IS NOT NULL
		AND tr.local_file_hash = ?
		AND er.encrypted_file_hash IS NOT NULL
		AND tr.deleted_at IS NULL 
		AND er.deleted_at IS NULL`

	res := txn.Raw(sql, hash)
	if res.Error != nil {
		t.log.Error(res.Error.Error())
		return false, fmt.Errorf("Unable to confirm that file has not been sent previously")
	}

	rows, err := res.Rows()
	if err != nil {
		t.log.Error(err.Error())
		return false, fmt.Errorf("Unable to confirm that file has not been sent previously")
	}
	defer rows.Close()

	rows.Next()

	var row AvailableToSend
	err = rows.Scan(
		&row.plaintextFileHash,
		&row.fileToTransfer,
		&row.plaintextFileHash,
		&row.remoteHost,
		&row.transferredFileHash,
		&row.remoteFileSize,
	)
	if err != nil {
		// handle this error
		t.log.Error(err.Error())
	}

	if (row.transferredFileHash != "" || row.remoteFileSize > 0) && row.remoteHost == remoteHost {
		// It looks like the file has been sent
		t.log.Warnf("File has been sent previously")

		return true, nil
	}

	// File has NOT been sent before
	return false, err
}

// Update updates the record
func (t TransferLog) Update(txn *gorm.DB, rec *TransferRecord) error {

	sql := "local_file_hash = ? AND remote_host = ? AND correlation_id = ?"
	result := txn.
		Model(rec).
		Where(sql, rec.LocalFileHash, rec.RemoteHost, rec.CorrelationID).
		UpdateColumns(TransferRecord{
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
