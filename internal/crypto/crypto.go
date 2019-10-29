package crypto

import (
	"crypto"
	"errors"
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
	"golang.org/x/crypto/ripemd160"
)

func init() {
	// we need to register this for the ANZ key.
	// this is deprecated but hopefully people make newer keys
	crypto.RegisterHash(crypto.RIPEMD160, ripemd160.New)
}

//ProviderConfig is an instance of the SFTP Connection Details
type ProviderConfig struct {
	EncryptionKey         string `json:"encryptionKey"`
	SigningKey            string `json:"signingKey"`
	SigningKeyPassword    string `json:"signingKeyPassword"`
	SigningFingerPrint    string `json:"signingFingerPrint"`
	FingerPrint           string `json:"fingerprint"`
	EncryptionKeyPassword string `json:"encryptionKeyPassword"`
	DecryptionKey         string `json:"decryptionKey"`
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
func NewProvider(config ProviderConfig, log *log.Entry) Provider {

	provider := &provider{
		config: config,
		log:    log,
	}
	return provider
}

func (p provider) DecryptFile(encryptedFile, outputFile string) error {

	return nil
}

//EncryptFile provides a simple wrapper to encrypt a file
func (p provider) EncryptFile(plainTextFile string, outputFile string) (err error) {
	p.log.Debugf("Encrypting file %s", plainTextFile)
	p.log.Debugf("Output file %s", outputFile)
	p.log.Debugf("Using EncryptionKey %s ", p.config.EncryptionKey)

	// Read in public key
	recipientKey, err := p.keyFromFile(p.config.EncryptionKey)
	if err != nil {
		return
	}
	p.log.Debug("Key found and loaded successfully")
	recipientKeys := []*openpgp.Entity{recipientKey}

	var signingKey *openpgp.Entity = nil
	if len(p.config.SigningKey) > 0 {
		p.log.Debugf("Signing with %s", p.config.SigningKey)
		if len(p.config.SigningKeyPassword) > 0 {
			signingKey, err = p.decryptArmoredKey(p.config.SigningKey, p.config.SigningKeyPassword)
		} else {

			signingKey, err = p.keyFromFile(p.config.SigningKey)
			if err != nil {
				return err
			}
		}

		p.log.Debug("Signing key loaded ")
	}

	// do we need to do this ?
	// compressed, err := gzip.NewWriterLevel(plain, gzip.BestCompression)
	// kingpin.FatalIfError(err, "Invalid compression level")

	// n, err := io.Copy(compressed, os.Stdin)

	inFile, err := os.Open(plainTextFile)
	if err != nil {
		return err
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	p.log.Debug("Performing Encryption ")
	//config := &packet.Config{}
	hints := &openpgp.FileHints{
		IsBinary: true,
	}
	// @todo currently uses defaults, provide other encryption options
	wc, err := openpgp.Encrypt(outFile, recipientKeys, signingKey, hints, nil)
	if err != nil {
		return err
	}

	bytes, err := io.Copy(wc, inFile)
	if err != nil {
		p.log.Errorf("Error Copying : %s", err.Error())
		return err
	}

	err = wc.Close()
	if err != nil {
		p.log.Errorf("Error Closing pgp writer : %s", err.Error())
		return err
	}

	s, err := os.Stat(plainTextFile)
	if err != nil {
		return err
	}

	p.log.Debug("Comparing Encrypted bytes with bytes written to disk")
	if s.Size() != bytes {
		return fmt.Errorf("File size of : %d does not equal the %d bytes encrypted", s.Size(), bytes)
	}

	p.log.Debugf("Decrypted file to %s", plainTextFile)

	return
}

func (p provider) decryptArmoredKey(fileName string, password string) (*openpgp.Entity, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	entitylist, err := openpgp.ReadArmoredKeyRing(f)
	if err != nil {
		return nil, err
	}
	if len(entitylist) != 1 {
		return nil, errors.New("The encrypted key contains more entities than expected. Feature request ?")
	}
	entity := entitylist[0]
	p.log.Debug("Private key from armored string:", entity.Identities)

	// Decrypt private key using passphrase
	passphrase := []byte(password)
	if entity.PrivateKey != nil && entity.PrivateKey.Encrypted {
		p.log.Debug("Decrypting private key using passphrase")
		err := entity.PrivateKey.Decrypt(passphrase)
		if err != nil {
			return nil, errors.New("failed to decrypt key: " + err.Error())
		}
	}
	return entity, err
}

//keyFromFile load from fil
func (p provider) keyFromFile(fileName string) (*openpgp.Entity, error) {

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	block, err := armor.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("Unable to read the Signing Key. Make sure it's ASCII Armoured (not binary): %s", err.Error())
	}

	return openpgp.ReadEntity(packet.NewReader(block.Body))
}
