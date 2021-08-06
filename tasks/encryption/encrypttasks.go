package encryption

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/masenocturnal/pipefire/internal/crypto"
	"github.com/masenocturnal/pipefire/internal/encryption_recorder"
	"github.com/sirupsen/logrus"
)

//GetConfig for a an appropriately shaped json configuration string return a valid ArchiveConfig
func GetConfig(jsonText string) (*crypto.EncryptFilesConfig, error) {
	config := &crypto.EncryptFilesConfig{}

	err := json.Unmarshal([]byte(jsonText), config)

	return config, err

}

func PGPCLIEncryptFilesInDir(config crypto.ProviderConfig, srcDir string, outputDir string, l *logrus.Entry) (errList []error) {

	l.Infof("Attempting to Encrypt files in %s using the CLI", srcDir)

	files, err := ioutil.ReadDir(srcDir)

	if err != nil {
		return append(errList, err)
	}

	if len(config.SigningFingerPrint) < 1 || len(config.FingerPrint) < 1 {
		x := fmt.Errorf("The fingerprint for the signing key : %s or the encryption key: %s is empty", config.SigningFingerPrint, config.FingerPrint)
		return append(errList, x)
	}

	if err := os.MkdirAll(outputDir, 0700); err != nil {
		return append(errList, err)
	}

	var cmdOut []byte

	for _, file := range files {
		srcFile := path.Join(srcDir, file.Name())
		destFile := path.Join(outputDir, file.Name()+".gpg")

		cmd := "/usr/bin/gpg2"
		args := []string{
			"-u",
			config.SigningFingerPrint,
			"-r",
			config.FingerPrint,
			"--openpgp",
			"--sign",
			"--batch",
			"--yes",
			"--output",
			destFile,
			"--encrypt",
			srcFile,
		}
		l.Info(strings.Join(args, " "))
		if cmdOut, err = exec.Command(cmd, args...).Output(); err != nil {
			x := err.(*exec.ExitError)

			l.Warn("Ensure that the GPG key is trusted otherwise you may encounter an assurance error")
			l.Errorf("Error executing command: %s Error: %s", cmd, err.Error())
			l.Errorf("Error: %s", x.Stderr)

			return append(errList, err)
		}
		out := string(cmdOut)
		l.Debug(out)

		if err != nil {
			l.Errorf("Unable to execute GPG task %s ", err.Error())
			return append(errList, err)
		}
	}

	l.Debug("PGP Encryption Task Complete")
	return
}

func PGPEncryptFilesForTransfer(config *crypto.EncryptFilesConfig, encryptionLog *encryption_recorder.EncryptionLog, correlationID string, l *logrus.Entry) (errList []error) {

	l.Infof("Attempting to Encrypt files in %s", config.SrcDir)

	for bank, providerConfig := range config.Providers {

		if providerConfig.Enabled {
			// Create the crypto provider
			encryptionProvider := crypto.NewProvider(providerConfig, l)
			srcDir := filepath.Join(config.SrcDir, providerConfig.SrcDir)

			outputDir := filepath.Join(config.OutputDir, providerConfig.DestDir)
			l.Debugf("Encrypting all files in located in %s to %s ", srcDir, outputDir)

			// encrypt files
			err := encryptFilesInDir(encryptionProvider, srcDir, outputDir, encryptionLog, correlationID, l)

			if err != nil {
				for _, e := range err {
					errList = append(errList, e)
				}
			}
		} else {
			l.Warnf("Skipping Encryption for %s ", bank)
		}
	}

	l.Debug("Encryption Task Complete")
	return
}

//encryptFilesInDir encrypt all the files in the directory with the given provider
func encryptFilesInDir(cryptoProvider crypto.Provider, srcDir string, outputDir string, encryptionLog *encryption_recorder.EncryptionLog, correlationID string, l *logrus.Entry) (errorList []error) {
	fileList, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return append(errorList, err)
	}

	err = os.MkdirAll(outputDir, 0700)
	if err != nil {
		return append(errorList, err)
	}

	if len(fileList) > 0 {
		for _, fileToEncrypt := range fileList {
			plainText := filepath.Join(srcDir, fileToEncrypt.Name())
			cryptFile := filepath.Join(outputDir, fileToEncrypt.Name()+".gpg")

			// hash file
			hash, err := crypto.HashFile(plainText)
			if err != nil {
				errorList = append(errorList, err)
				// skip this file
				break
			}
			txn := encryptionLog.Conn.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})

			// record file
			record := &encryption_recorder.EncryptionRecord{
				LocalFileName: plainText,
				LocalFilePath: plainText,
				LocalFileSize: fileToEncrypt.Size(),
				LocalFileHash: hash,
				CorrelationID: correlationID,
			}

			err = encryptionLog.Create(txn, record)
			if err != nil {
				l.Errorf("Unable to create encryption record for %s ", plainText)
				l.Debugf("Creating error %s ", err.Error())
				// try cast to mysql error
				dbErr := err.(*mysql.MySQLError)
				if dbErr != nil && dbErr.Number == 1062 {
					l.Warningf("File %s with hash %s has been processed before", plainText, hash)
					txn.Rollback()
					continue
				} else {
					errorList = append(errorList, fmt.Errorf("Unable to create record %s ", err.Error()))
					txn.Rollback()
					continue
				}
			}
			res := txn.Commit()
			if res.Error != nil {
				txn.RollbackUnlessCommitted()
			}

			txn = encryptionLog.Conn.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})

			// encrypt file
			err = cryptoProvider.EncryptFile(plainText, cryptFile)
			if err != nil {
				l.Warningf("Error encrypting file %s : %s", plainText, err.Error())
				errorList = append(errorList, err)
				txn.Rollback()
				continue
			}

			record.EncryptedFileHash, _ = crypto.HashFile(cryptFile)

			// get the encryption key
			recipientKey, err := cryptoProvider.GetEncryptionKey()
			if err != nil {
				errorList = append(errorList, err)
				txn.Rollback()
				continue
			}

			// get the signing key
			signingKey, err := cryptoProvider.GetSigningKey()

			// @todo we are assuming that the encryption key is the public key.
			// This my not be correct
			record.RecipientKey = recipientKey.PrimaryKey.KeyIdString()

			if signingKey != nil && signingKey.PrimaryKey != nil {
				signingKeyFingerprint := signingKey.PrivateKey.KeyIdString()
				if len(signingKeyFingerprint) > 0 {
					record.SigningKey = signingKeyFingerprint
				}
			}

			err = encryptionLog.Update(txn, record)
			if err != nil {
				errorList = append(errorList, err)
				txn.Rollback()
				break
			}

			// commit the transaction
			res = txn.Commit()
			if res.Error != nil {
				txn.RollbackUnlessCommitted()
			}
		}
	} else {
		l.Warnf("No files to encrypt in %s", srcDir)
	}
	return
}
