package directdebit

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/masenocturnal/pipefire/internal/crypto"
)

// EncryptFilesConfig is the configuration requriements for the encryptFiles task
type EncryptFilesConfig struct {
	SrcDir    string                           `json:"srcDir"`
	OutputDir string                           `json:"outputDir"`
	Providers map[string]crypto.ProviderConfig `json:"providers"`
	Enabled   bool                             `json:"enabled"`
}

func (p pipeline) pgpCLIEncryptFilesInDir(config crypto.ProviderConfig, srcDir string, outputDir string) (errList []error) {
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

	for _, file := range files {
		srcFile := path.Join(srcDir, file.Name())
		destFile := path.Join(outputDir, file.Name())
		args := []string{
			fmt.Sprintf("-u %s", config.SigningFingerPrint),
			fmt.Sprintf("-r %s", config.FingerPrint),
			"--openpgp",
			"--sign",
			fmt.Sprintf("--output %s.gpg", destFile),
			fmt.Sprintf("--encrypt %s", srcFile),
		}
		cmd := &exec.Cmd{
			Path:   "/usr/bin/gpg2",
			Args:   args,
			Env:    nil,
			Dir:    ".",
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		err := cmd.Run()
		if err != nil {
			p.log.Errorf("Unable to execute GPG task %s ", err.Error())
			return append(errList, err)
		}
	}

	p.log.Debug("PGP Encryption Task Complete")
	return
}

func (p pipeline) encryptFiles(config EncryptFilesConfig) (errList []error) {
	p.log.Infof("Attempting to Encrypt files in %s", config.SrcDir)

	// @todo this could be cleaner there is a bit of code duplication here
	// loop through directories
	// GA goes to ANZ
	var bank string = "anz"
	p.log.Debugf("Looking in the list providers for configuration config.Providers[%s]", bank)
	if anzProviderConfig, ok := config.Providers[bank]; ok {
		anzCryptoProvider := crypto.NewProvider(anzProviderConfig, p.log)
		srcDir := filepath.Join(config.SrcDir, "GA")
		outputDir := filepath.Join(config.OutputDir, "ANZ")
		err := p.encryptFilesInDir(anzCryptoProvider, srcDir, outputDir)

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
		pxCryptoProvider := crypto.NewProvider(pxProviderConfig, p.log)
		srcDir := filepath.Join(config.SrcDir, "PX")
		outputDir := filepath.Join(config.OutputDir, "PX")

		err := p.encryptFilesInDir(pxCryptoProvider, srcDir, outputDir)
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
		//bnzCryptoProvider := crypto.NewProvider(bnzProviderConfig, p.log)
		srcDir := filepath.Join(config.SrcDir, "BNZ")
		outputDir := filepath.Join(config.OutputDir, "BNZ")
		err := p.pgpCLIEncryptFilesInDir(bnzProviderConfig, srcDir, outputDir)
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

	p.log.Debug("Encryption Task Complete")
	return
}

//encryptFilesInDir encrypt all the files in the directory with the given provider
func (p pipeline) encryptFilesInDir(cryptoProvider crypto.Provider, srcDir string, outputDir string) (errorList []error) {
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
