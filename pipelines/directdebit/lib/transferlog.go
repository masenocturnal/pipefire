package database

import (
	"database/sql"

	"time"

	_ "github.com/samonzeweb/godb/adapters/mysql"

	log "github.com/sirupsen/logrus"
)

// // DbConfig stores connection information for the database
// type DbConfig struct {
// 	// @todo pull from config
// 	Username string `json:"username"`
// 	Password string `json:"password"`
// 	Host     string `json:"host"`
// 	Name     string `json:"name"`
// 	Timeout  string `json:"timeout"`
// }

// TransferLog provides a mechanism to update the transfer status
type TransferRecorder interface {
	CreateXferRecord() error
	FileSent(hash string) error
	Close() error
}

//XferRecord Maps to a row in the FileTransfers table
type XferRecord struct {
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
}

//TransferLog Stores a database log
type TransferLog struct {
	Conn *sql.DB
	txn  *sql.Tx
	log  *log.Entry
}

// New provides a connection to the database
func New(Conn *sql.DB, log *log.Entry) TransferRecorder {

	transferLog := &transferLog{
		Conn: Conn,
		log:  log,
	}

	return transferLog
}

func (t transferLog) CreateXferRecord() error {

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
