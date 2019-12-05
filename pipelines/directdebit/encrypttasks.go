package directdebit

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/masenocturnal/pipefire/internal/crypto"
)

// EncryptFilesConfig is the configuration requriements for the encryptFiles task
type EncryptFilesConfig struct {
	SrcDir    string                           `json:"srcDir"`
	OutputDir string                           `json:"outputDir"`
	Providers map[string]crypto.ProviderConfig `json:"providers"`
	Enabled   bool                             `json:"enabled"`
}

func (p ddPipeline) pgpCLIEncryptFilesInDir(config crypto.ProviderConfig, srcDir string, outputDir string) (errList []error) {
	p.log.Infof("Attempting to Encrypt files in %s using the CLI", srcDir)
	//gpg2 -u "Certegy BNZ (FTG-PROD)" -r "BNZConnect (FTG-PROD)" --openpgp --sign --output "./BNZ_SEND/${fileName}.gpg"  --encrypt "$fileName"

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
		p.log.Info(strings.Join(args, " "))
		if cmdOut, err = exec.Command(cmd, args...).Output(); err != nil {
			x := err.(*exec.ExitError)

			p.log.Warn("Ensure that the GPG key is trusted otherwise you may encounter an assurance error")
			p.log.Errorf("Error executing command: %s Error: %s", cmd, err.Error())
			p.log.Errorf("Error: %s", x.Stderr)

			return append(errList, err)
		}
		out := string(cmdOut)
		p.log.Debug(out)

		if err != nil {
			p.log.Errorf("Unable to execute GPG task %s ", err.Error())
			return append(errList, err)
		}
	}

	p.log.Debug("PGP Encryption Task Complete")
	return
}

func (p ddPipeline) pgpEncryptFilesForBank(config *EncryptFilesConfig) (errList []error) {
	p.log.Infof("Attempting to Encrypt files in %s", config.SrcDir)

	for bank, providerConfig := range config.Providers {

		if providerConfig.Enabled {
			// Create the crypto provider
			encryptionProvider := crypto.NewProvider(providerConfig, p.log)
			srcDir := filepath.Join(config.SrcDir, providerConfig.SrcDir)

			outputDir := filepath.Join(config.OutputDir, providerConfig.DestDir)
			p.log.Debugf("Encrypting all files in located in %s to %s ", srcDir, outputDir)

			// encrypt files
			err := p.encryptFilesInDir(encryptionProvider, srcDir, outputDir)

			if err != nil {
				for _, e := range err {
					errList = append(errList, e)
				}
			}
		} else {
			p.log.Warnf("Skipping Encryption for %s ", bank)
		}
	}

	p.log.Debug("Encryption Task Complete")
	return
}

//encryptFilesInDir encrypt all the files in the directory with the given provider
func (p ddPipeline) encryptFilesInDir(cryptoProvider crypto.Provider, srcDir string, outputDir string) (errorList []error) {
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
			f := filepath.Join(srcDir, fileToEncrypt.Name())
			o := filepath.Join(outputDir, fileToEncrypt.Name())

			// hash file
			hash, err := crypto.HashFile(f)
			if err != nil {
				errorList = append(errorList, err)
			}
			txn := p.encryptionLog.Conn.BeginTx(context.Background, &sql.TxOptions{Isolation: sql.LevelSerializable})
			// record file
			record := &EncryptionRecord{
				LocalFileName: f,
				LocalFilePath: f,
				LocalFileSize: fileToEncrypt.Size(),
				LocalFileHash: hash,
				CorrelationID: p.correlationID,
			}
			txn = p.
				p.encryptionLog.Create()
			// encrypt file
			err = cryptoProvider.EncryptFile(f, o+".gpg")
			if err != nil {
				p.log.Warningf("Error encrypting file %s : %s", f, err.Error())
				errorList = append(errorList, err)
			}
		}
	} else {
		p.log.Warnf("No files to encrypt in %s", srcDir)
	}
	return
}
