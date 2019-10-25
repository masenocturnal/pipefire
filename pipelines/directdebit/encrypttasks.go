package directdebit

import (
	"errors"
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

	// [sysam@sec-docker-101 BNZ]$ gpg2 --verbose -u "Certegy BNZ (FTG-PROD)" -r "BNZConnect (FTG-PROD)" --openpgp --sign --output "${fileName}.gpg"  --encrypt "$fileName"
	// gpg: using PGP trust model
	// gpg: using subkey 92715549 instead of primary key 18FC3718
	// gpg: This key probably belongs to the named user
	// gpg: writing to `158042884DD.191025.011946.238011.CON.gpg'
	// gpg: ELG/AES256 encrypted for: "92715549 BNZConnect (FTG-PROD) <BNZConnect@bnz.co.nz>"
	// gpg: RSA/SHA1 signature from: "6383B673 Certegy BNZ (FTG-PROD)"

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
