package database

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

// DbConfig stores connection information for the database
type DbConfig struct {
	// @todo pull from config
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Name     string `json:"name"`
	Timeout  string `json:"timeout"`
}

// TransferLog provides a mechanism to update the transfer status
type TransferLog interface {
	CreateXferRecord() error
	FileSent(hash string) error
	Close() error
}

//XferRecord Maps to a row in the FileTransfers table
type XferRecord struct {
	process_start` DATETIME NOT NULL COMMENT 'Date and time transfer process was started',
	process_errors` TEXT COMMENT 'Any errors detected in processing the file',
	process_end` DATETIME COMMENT 'Date and time transfer process ended',
	file_name` TINYTEXT NOT NULL COMMENT 'Name of the file being transferred',
	file_recipient` TINYTEXT NOT NULL COMMENT 'Place that the file is being transferred to',
	file_key` TEXT COMMENT 'Fingerprint of the key used to encrypt the file',
	file_sender` TINYTEXT NOT NULL COMMENT 'Name of the machine that is sending the file',
	hash_plaintext` VARCHAR(254) NOT NULL COMMENT 'Hash of the file on disk before encryption',
	hash_ciphertext` TEXT COMMENT 'Hash of the file on disk after encryption',
	hash_remote` TEXT COMMENT 'Hash of the file after upload to recipient',
		 
}

type transferLog struct {
	Conn *sql.DB
	log  *log.Entry
}

// NewTransferLog provides a connection to the database
func NewTransferLog(Conn *sql.DB, log *log.Entry) TransferLog {

	transferLog := &transferLog{
		Conn: Conn,
		log:  log,
	}

	return transferLog
}

func (t transferLog) CreateXferRecord() error {

	
		
	
	
	~                                                                                                                                                                                                                                                                              
	~                                                 
	return nil
}

func (t transferLog) GetRecordByHash() {

}
func (t transferLog) GetRecordByFileName() {

}

func (t transferLog) FileSent(hash string) error {

	return nil
}

func (t transferLog) Close() (err error) {
	if err = t.Conn.Close(); err != nil {
		t.log.Error("Error closing database connection: %s", err.Error())
	}
	return err
}
