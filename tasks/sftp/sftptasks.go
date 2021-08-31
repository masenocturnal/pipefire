package sftp

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/masenocturnal/pipefire/internal/sftp"
	"github.com/masenocturnal/pipefire/internal/transfer_recorder"
	"github.com/sirupsen/logrus"
)

func GetConfig(jsonText string) (*sftp.SftpConfig, error) {
	sftpConfig := &sftp.SftpConfig{}

	err := json.Unmarshal([]byte(jsonText), sftpConfig)

	return sftpConfig, err

}

// get files from a particular endpoint
func SFTPGet(conf *sftp.SftpConfig, filesToTransfer *[]sftp.TransferFiles, l *logrus.Entry) error {
	l.Infof("Begin sftpGet: %s ", conf.Sftp.Host)
	sftp, err := sftp.NewConnection("From", conf.Sftp, l)
	if err != nil {
		return err
	}
	defer sftp.Close()

	// grab all the files from the pickup directory
	confirmations, errors := sftp.GetDir(conf.RemoteDir, conf.LocalDir, filesToTransfer)
	if errors.Len() > 0 {
		// show all errors
		for temp := errors.Front(); temp != nil; temp = temp.Next() {
			l.Error(temp.Value)
		}
		return fmt.Errorf("Error getting files from %s ", conf.RemoteDir)
	}

	for temp := confirmations.Front(); temp != nil; temp = temp.Next() {
		result, _ := json.MarshalIndent(temp.Value, "", " ")
		l.Info(string(result))
	}

	l.Info("sftpGet Complete")
	return err
}

// sftpClean cleans the repote directory
func SFTPClean(conf *sftp.SftpConfig, filesToTransfer *[]sftp.TransferFiles, l *logrus.Entry) (err error) {
	l.Infof("Begin sftpClean: %s", conf.Sftp.Host)
	l.Debugf("Cleaning remote dir: %s ", conf.RemoteDir)

	sftp, err := sftp.NewConnection(conf.Sftp.Host, conf.Sftp, l)
	if err != nil {
		return err
	}
	defer sftp.Close()

	err = sftp.CleanDir(conf.RemoteDir, filesToTransfer)
	if err == nil {
		l.Infof("sftpClean Complete: Removed files from: %s ", conf.RemoteDir)
	}
	return err
}

func SFTPToSafe(conf *sftp.SftpConfig, l *logrus.Entry) (err error) {

	l.Infof("Begin sftpToSafe: %s", conf.Sftp.Host)
	l.Debugf("Sftp transfer from %s to %s @ %s ", conf.LocalDir, conf.RemoteDir, conf.Sftp.Host)

	// ANZ SFTP is odd and requires us to establish new connections for
	// each load
	filesInDir, err := ioutil.ReadDir(conf.LocalDir)
	if err != nil {
		return
	}

	var sb strings.Builder
	for _, file := range filesInDir {

		sftp, e := sftp.NewConnection(conf.Sftp.Host, conf.Sftp, l)
		if e != nil {
			l.Errorf("Unable to connect to %s Error: %s ", conf.Sftp.Host, e.Error())
			return e
		}
		lfp := filepath.Join(conf.LocalDir, file.Name())
		rfp := filepath.Join(conf.RemoteDir, file.Name())

		confirmation, err := sftp.SendFile(lfp, rfp)
		if err != nil {
			l.Errorf("Unable to Send File to %s Error: %s ", conf.Sftp.Host, err.Error())
		}
		result, _ := json.MarshalIndent(confirmation, "", " ")
		sb.WriteString(string(result))
		l.Info(sb.String())
		sftp.Close()
	}

	l.Infof("sftpTo Complete, remote %s ", conf.RemoteDir)
	return nil
}

// send files to a particular endpoint
func SFTPTo(conf *sftp.SftpConfig, transferLog *transfer_recorder.TransferLog, correlationID string, l *logrus.Entry) (err error) {
	l.Infof("Begin sftpTo: %s", conf.Sftp.Host)
	l.Debugf("Sftp transfer from %s to %s @ %s ", conf.LocalDir, conf.RemoteDir, conf.Sftp.Host)

	// Record the files we are about to send so that we can ensure we never
	// send the same file twice
	// This is done as an atomic commit to avoid race conditions
	if err := recordFilesToSend(conf.LocalDir, conf.Sftp.Host, transferLog, correlationID, l); err != nil {
		return err
	}

	// establish the connection and bail if we can't get it
	sftp, err := sftp.NewConnection(conf.Sftp.Host, conf.Sftp, l)
	if err != nil {
		return
	}
	defer sftp.Close()

	dirList, _ := ioutil.ReadDir(conf.LocalDir)

	if transferLog == nil || transferLog.Conn == nil {
		return fmt.Errorf("Transfer log is unavailable, aborting")
	}

	// we want to examine each of these files to ensure they haven't been sent before
	for _, file := range dirList {
		cur := filepath.Join(conf.LocalDir, file.Name())

		// create a synchronous transaction so that only 1 process can update the database at a time
		tx := transferLog.Conn.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})

		fileHash, err := hashFile(cur)
		if err != nil {
			return err
		}

		val, err := transferLog.FileAlreadySent(tx, fileHash, conf.Sftp.Host)
		if err != nil {
			tx.Rollback()
			l.Errorf("Unable to confirm if file has been sent. Aborting . Err %s", err.Error())
			// def don't want to send if we don't know if it's been sent.
			return err
		}

		if val == true {
			// don't transfer the file
			l.Warnf("The file %s has already been sent. File will *NOT* be transferred ", cur)
			tx.Rollback()
		} else {
			startTime := time.Now()
			// attempt to transfer
			confirmation, err := sftp.SendFile(cur, conf.RemoteDir)
			if err != nil {
				rec := &transfer_recorder.TransferRecord{
					RemoteHost: conf.Sftp.Host,
					// @todo see if this is populated TransferredFileHash: confirmation.TransferredHash,
					TransferStart:  startTime,
					TransferEnd:    time.Now(),
					LocalFileHash:  fileHash,
					CorrelationID:  correlationID,
					TransferErrors: err.Error(),
				}
				transferLog.RecordError(tx, rec)
			}

			// log the confirmation
			result, _ := json.MarshalIndent(confirmation, "", " ")
			l.Info(string(result))

			if confirmation != nil {

				rec := &transfer_recorder.TransferRecord{
					RemoteFileName:      confirmation.RemoteFileName,
					RemoteFilePath:      confirmation.RemotePath,
					RemoteFileSize:      confirmation.RemoteSize,
					RemoteHost:          conf.Sftp.Host,
					RecipientName:       "",
					SenderName:          "",
					TransferredFileHash: confirmation.TransferredHash,
					TransferStart:       startTime,
					TransferEnd:         time.Now(),
					LocalFileHash:       confirmation.LocalHash,
					CorrelationID:       correlationID,
				}
				if err := transferLog.Update(tx, rec); err != nil {
					tx.RollbackUnlessCommitted()
				}
				tx.Commit()

			} else {
				l.Warnf("Didn't receive file transfer confirmation for %s", cur)
			}

		}
	}

	// try and list the directory
	sftp.ListRemoteDir(conf.RemoteDir)

	l.Infof("sftpTo Complete, remote %s ", conf.RemoteDir)
	return nil
}

func recordFilesToSend(localDir string, remoteHost string, transferLog *transfer_recorder.TransferLog, correlationID string, l *logrus.Entry) error {
	// @todo validate config

	// Record the files in the database so we can
	// guard against sending them twice
	// start the transaction
	tx := transferLog.Conn.Begin()

	// list all the files ine
	filesInDir, err := ioutil.ReadDir(localDir)
	if err != nil {
		return err
	}

	if len(filesInDir) < 1 {
		l.Warnf("No files to send in %s", localDir)
		// it's not an error ...this can happen in multiple runs but there is
		// no sense doing anything more
		return nil
	}

	hostName, _ := os.Hostname()

	for _, file := range filesInDir {
		cur := filepath.Join(localDir, file.Name())

		hash, err := hashFile(cur)
		if err != nil {
			return err
		}

		// add the record to the transferlog
		record := &transfer_recorder.TransferRecord{
			LocalFileSize: file.Size(),
			LocalFileName: file.Name(),
			LocalFilePath: cur,
			RemoteHost:    remoteHost,
			LocalHostID:   hostName,
			CorrelationID: correlationID,
			LocalFileHash: hash,
		}

		err = transferLog.Create(tx, record)
		if err != nil {
			dbErr := err.(*mysql.MySQLError)
			if dbErr != nil {
				switch dbErr.Number {
				case 1062:
					//. this is ok...if a previous attempt fails, we want to try again
					l.Warnf("A process has previously attempted to transfer this file: %s", cur)
					break
				default:
					tx.Rollback()
					// something is wrong, we should stop
					l.Error(dbErr.Error())
					return fmt.Errorf("Unexpected error trying to record files to send")
				}
			} else {
				tx.Rollback()
				l.Errorf("Error when trying to reserve file sending entries. Error: %s", err.Error())
				return err
			}
		}
	}
	tx.Commit()
	return nil
}

// @ turn into a lib
func hashFile(filePath string) (string, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("Can't hash %s bailing out. Error :  %s", filePath, err.Error())
	}

	// @todo inject hashwriter to support other hash algorithms
	hashWriter := sha256.New()
	// calculate local checksum
	_, err = hashWriter.Write(data)
	return hex.EncodeToString(hashWriter.Sum(nil)), err
}
