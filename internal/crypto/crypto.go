package crypto

import (
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

//ProviderConfig is an instance of the SFTP Connection Details
type ProviderConfig struct {
	EncryptionKey string `json:"encryptionKey"`
	SigningKey    string `json:"signingKey"`
	FingerPrint   string `json:"fingerprint"`
	KeyPassword   string `json:"keyPassword"`
	DecryptionKey string `json:"decryptionKey"`
}

//Provider helper functions to encrypt/decrypt files
type Provider interface {
	EncryptFile(string, string) error
	DecryptFile(string, string) error
}

type provider struct {
	config ProviderConfig
	log    *log.Entry
}

//NewProvider returns a Crypto Provider
func NewProvider(config ProviderConfig, correlationID string) Provider {

	provider := &provider{
		config: config,
		log:    log.WithField("correlationId", correlationID),
	}
	return provider
}

func (p provider) DecryptFile(encryptedFile, outputFile string) error {

	return nil
}

//EncryptFile provides a simple wrapper to encrypt a file
func (p provider) EncryptFile(plainTextFile string, outputFile string) (err error) {
	log.Infof("Encrypting file %s to %s using EncryptionKey %s ", plainTextFile, outputFile, p.config.EncryptionKey)

	// Read in public key
	recipientKey, err := keyFromFile(p.config.EncryptionKey)
	if err != nil {
		return
	}
	log.Debug("Key found and loaded successfully")
	recipientKeys := []*openpgp.Entity{recipientKey}

	var signingKey *openpgp.Entity = nil
	if len(p.config.SigningKey) > 0 {
		log.Debugf("Signing with %s", p.config.SigningKey)

		signingKey, err = keyFromFile(p.config.SigningKey)
		if err != nil {
			return err
		}
		log.Debug("Signing key loaded ")
	}

	hints := &openpgp.FileHints{IsBinary: true}
	inFile, err := os.Open(plainTextFile)
	if err != nil {
		return
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		return
	}

	log.Debug("Performing Encryption ")
	// @todo currently uses defaults, provide other encryption options
	wc, err := openpgp.Encrypt(outFile, recipientKeys, signingKey, hints, nil)

	bytes, err := io.Copy(wc, inFile)
	if err != nil {
		return
	}

	s, err := os.Stat(plainTextFile)
	if err != nil {
		return err
	}

	log.Debug("Comparing Encrypted bytes with bytes written to disk")
	if s.Size() != bytes {
		return fmt.Errorf("File size of : %d does not equal the %d bytes encrypted", s.Size(), bytes)
	}
	return wc.Close()
}

//keyFromFile load from fil
func keyFromFile(fileName string) (*openpgp.Entity, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	block, err := armor.Decode(f)
	if err != nil {
		return nil, err
	}
	return openpgp.ReadEntity(packet.NewReader(block.Body))
}
