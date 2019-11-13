package directdebit

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
)

//SftpConfig Required Params for transferring to or from an SFTP Server
type SftpConfig struct {
	RemoteDir string        `json:"remoteDir"`
	LocalDir  string        `json:"localDir"`
	Sftp      sftp.Endpoint `json:"sftp"`
	Enabled   bool          `json:"enabled"`
}

// get files from a particular endpoint
func (p ddPipeline) sftpGet(conf *SftpConfig) error {
	p.log.Infof("Begin sftpGet: %s ", conf.Sftp.Host)
	sftp, err := sftp.NewConnection("From", conf.Sftp, p.log)
	if err != nil {
		return err
	}
	defer sftp.Close()

	// grab all the files from the pickup directory
	confirmations, errors := sftp.GetDir(conf.RemoteDir, conf.LocalDir)
	if errors.Len() > 0 {
		// show all errors
		for temp := errors.Front(); temp != nil; temp = temp.Next() {
			p.log.Error(temp.Value)
		}
		return fmt.Errorf("Error getting files from %s ", conf.RemoteDir)
	}

	for temp := confirmations.Front(); temp != nil; temp = temp.Next() {
		result, _ := json.MarshalIndent(temp.Value, "", " ")
		p.log.Info(string(result))
	}

	p.log.Info("sftpGet Complete")
	return err
}

// sftpClean cleans the repote directory
func (p ddPipeline) sftpClean(conf *SftpConfig) (err error) {
	p.log.Infof("Begin sftpClean: %s", conf.Sftp.Host)
	p.log.Debugf("Cleaning remote dir: %s ", conf.RemoteDir)

	sftp, err := sftp.NewConnection(conf.Sftp.Host, conf.Sftp, p.log)
	if err != nil {
		return
	}
	defer sftp.Close()

	err = sftp.CleanDir(conf.RemoteDir)
	if err == nil {
		p.log.Infof("sftpClean Complete: Removed files from: %s ", conf.RemoteDir)
	}
	return err
}

func (p ddPipeline) sftpToSafe(conf *SftpConfig) (err error) {

	p.log.Infof("Begin sftpToSafe: %s", conf.Sftp.Host)
	p.log.Debugf("Sftp transfer from %s to %s @ %s ", conf.LocalDir, conf.RemoteDir, conf.Sftp.Host)

	// ANZ SFTP is odd and requires us to establish new connections for
	// each load
	filesInDir, err := ioutil.ReadDir(conf.LocalDir)
	if err != nil {
		return
	}

	var sb strings.Builder
	for _, file := range filesInDir {

		sftp, e := sftp.NewConnection(conf.Sftp.Host, conf.Sftp, p.log)
		if e != nil {
			p.log.Errorf("Unable to connect to %s Error: %s ", conf.Sftp.Host, e.Error())
			return e
		}
		lfp := filepath.Join(conf.LocalDir, file.Name())
		rfp := filepath.Join(conf.RemoteDir, file.Name())

		confirmation, err := sftp.SendFile(lfp, rfp)
		if err != nil {
			p.log.Errorf("Unable to Send File to %s Error: %s ", conf.Sftp.Host, err.Error())
		}
		result, _ := json.MarshalIndent(confirmation, "", " ")
		sb.WriteString(string(result))
		p.log.Info(sb.String())
		sftp.Close()
	}

	p.log.Infof("sftpTo Complete, remote %s ", conf.RemoteDir)
	return nil
}

// send files to a particular endpoint
func (p ddPipeline) sftpTo(conf *SftpConfig) (err error) {
	p.log.Infof("Begin sftpTo: %s", conf.Sftp.Host)
	p.log.Debugf("Sftp transfer from %s to %s @ %s ", conf.LocalDir, conf.RemoteDir, conf.Sftp.Host)

	// Record the files we are about to send so that we can ensure we never
	// send the same file twice
	// This is done as an atomic commit to avoid race conditions
	if err := p.recordFilesToSend(conf.LocalDir, conf.Sftp.Host); err != nil {
		return err
	}

	// establish the connection and bail if we can't get it
	sftp, err := sftp.NewConnection(conf.Sftp.Host, conf.Sftp, p.log)
	if err != nil {
		return
	}
	defer sftp.Close()

	dirList, _ := ioutil.ReadDir(conf.LocalDir)

	// we want to examine each of these files to ensure they haven't been sent before
	for _, file := range dirList {
		cur := filepath.Join(conf.LocalDir, file.Name())

		// create a synchronous transaction so that only 1 process can update the database at a time
		tx := p.transferlog.Conn.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})

		// i don't like this duplication
		fileHash, err := hashFile(cur)
		if err != nil {
			return err
		}

		val, err := p.transferlog.FileAlreadySent(tx, fileHash, conf.Sftp.Host)
		if err != nil {
			tx.Rollback()
			p.log.Errorf("Unable to confirm if file has been sent. Aborting . Err %s", err.Error())
			// def don't want to send if we don't know if it's been sent.
			return err
		}

		if val == true {
			// don't transfer the file
			p.log.Warnf("The file %s has already been sent. File will *NOT* be transferred ", cur)
			tx.Rollback()
		} else {
			startTime := time.Now()
			// attempt to transfer
			confirmation, err := sftp.SendFile(cur, conf.RemoteDir)
			if err != nil {
				p.transferlog.RecordError(tx, fileHash, conf.Sftp.Host, err.Error())
			}

			// log the confirmation
			result, _ := json.MarshalIndent(confirmation, "", " ")
			p.log.Info(string(result))

			if confirmation != nil {

				rec := &Record{
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
					CorrelationID:       p.correlationID,
				}
				if err := p.transferlog.Update(tx, rec); err != nil {
					tx.RollbackUnlessCommitted()
				}
				tx.Commit()

			} else {
				p.log.Warnf("Didn't receive file transfer confirmation for %s", cur)
			}

		}
	}

	// try and list the directory
	sftp.ListRemoteDir(conf.RemoteDir)

	p.log.Infof("sftpTo Complete, remote %s ", conf.RemoteDir)
	return nil
}

func (p ddPipeline) recordFilesToSend(localDir string, remoteHost string) error {
	// @todo validate config

	// Record the files in the database so we can
	// guard against sending them twice
	// start the transaction
	tx := p.transferlog.Conn.Begin()
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		tx.Rollback()
	// 	}
	// }()

	// list all the files ine
	filesInDir, err := ioutil.ReadDir(localDir)
	if err != nil {
		return err
	}

	if len(filesInDir) < 1 {
		p.log.Warnf("No files to send in %s", localDir)
		return fmt.Errorf("%s is empty", localDir)
	}

	hostName, _ := os.Hostname()

	for _, file := range filesInDir {
		cur := filepath.Join(localDir, file.Name())

		hash, err := hashFile(cur)
		if err != nil {
			return err
		}

		// add the record to the transferlog
		record := &Record{
			LocalFileSize: file.Size(),
			LocalFileName: file.Name(),
			LocalFilePath: cur,
			RemoteHost:    remoteHost,
			LocalHostID:   hostName,
			CorrelationID: p.correlationID,
			LocalFileHash: hash,
		}

		err = p.transferlog.Create(tx, record)
		if err != nil {
			dbErr := err.(*mysql.MySQLError)
			if dbErr != nil {
				switch dbErr.Number {
				case 1062:
					//. this is ok...if a previous attempt fails, we want to try again
					p.log.Warnf("A process has previously attempted to transfer this file: %s", cur)
					break
				default:
					tx.Rollback()
					// something is wrong, we should stop
					p.log.Error(dbErr.Error())
					return fmt.Errorf("Unexpected error trying to record files to send")
				}
			} else {
				tx.Rollback()
				p.log.Errorf("Error when trying to reserve file sending entries. Error: %s", err.Error())
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
