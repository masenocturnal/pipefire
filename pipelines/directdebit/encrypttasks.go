package directdebit

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/masenocturnal/pipefire/internal/crypto"
)

// EncryptFilesConfig is the configuration requriements for the encryptFiles task
type EncryptFilesConfig struct {
	SrcDir    string                           `json:"srcDir"`
	Providers map[string]crypto.ProviderConfig `json:"providers"`
}

func (p pipeline) encryptFiles(config EncryptFilesConfig) (errList []error) {
	p.log.Infof("Attempting to Encrypt files in %s", config.SrcDir)

	// @todo this could be cleaner there is a bit of code duplication here
	// loop through directories
	// GA goes to ANZ
	var bank string = "anz"
	p.log.Debugf("Looking in the list providers for configuration config.Providers[%s]", bank)
	if anzProviderConfig, ok := config.Providers[bank]; ok {

		anzCryptoProvider := crypto.NewProvider(anzProviderConfig, p.correlationID)
		err := p.encryptFilesInDir(anzCryptoProvider, filepath.Join(config.SrcDir, "GA"))

		if err != nil {
			for _, e := range err {
				errList = append(errList, e)
			}
		}
	} else {
		msg := fmt.Sprintf("Encryption configuration not found. Task encryptFiles Task requires a provider configured for %s", bank)
		p.log.Error(msg)
		errList = append(errList, errors.New(msg))
	}

	bank = "px"
	p.log.Debugf("Looking in the list providers for configuration config.Providers[%s]", bank)
	if pxProviderConfig, ok := config.Providers[bank]; ok {
		pxCryptoProvider := crypto.NewProvider(pxProviderConfig, p.correlationID)
		err := p.encryptFilesInDir(pxCryptoProvider, filepath.Join(config.SrcDir, "PX"))
		if err != nil {
			for _, e := range err {
				errList = append(errList, e)
			}
		}
	} else {
		msg := fmt.Sprintf("Encryption configuration not found. Task encryptFiles Task requires a provider configured for %s", bank)
		p.log.Error(msg)
		errList = append(errList, errors.New(msg))
	}

	bank = "bnz"
	p.log.Debugf("Looking in the list providers for configuration config.Providers[%s]", bank)
	if bnzProviderConfig, ok := config.Providers[bank]; ok {
		bnzCryptoProvider := crypto.NewProvider(bnzProviderConfig, p.correlationID)
		err := p.encryptFilesInDir(bnzCryptoProvider, filepath.Join(config.SrcDir, "BNZ"))
		if err != nil {
			for _, e := range err {
				errList = append(errList, e)
			}
		}
	} else {
		msg := fmt.Sprintf("Encryption configuration not found. Task encryptFiles Task requires a provider configured for %s", bank)
		p.log.Error(msg)
		errList = append(errList, errors.New(msg))
	}

	p.log.Info("Decryption Task Complete")
	return
}

//encryptFilesInDir encrypt all the files in the directory with the given provider
func (p pipeline) encryptFilesInDir(cryptoProvider crypto.Provider, dir string) (errorList []error) {
	fileList, err := ioutil.ReadDir(dir)
	if err != nil {
		return append(errorList, err)
	}

	if len(fileList) > 0 {
		for _, fileToEncrypt := range fileList {
			f := filepath.Join(dir, fileToEncrypt.Name())
			err = cryptoProvider.EncryptFile(f, f+".gpg")
			if err != nil {
				p.log.Warningf("Error encrypting file %s : %s", f, err.Error())
				errorList = append(errorList, err)
			}
		}
	} else {
		p.log.Warnf("No files to encrypt in %s", dir)
	}
	return
}
