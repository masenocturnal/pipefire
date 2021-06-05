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
	SrcDir                string `json:"srcDir"`
	DestDir               string `json:"destDir"`
	Enabled               bool   `json:"enabled"`
}

//Provider helper functions to encrypt/decrypt files
type Provider interface {
	EncryptFile(string, string) error
	DecryptFile(string, string) error
	GetEncryptionKey() (encryptionKey *openpgp.Entity, err error)
	GetSigningKey() (signingKey *openpgp.Entity, err error)
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

//GetEncryptionKey Returns the currently configured encryption key
func (p provider) GetEncryptionKey() (encryptionKey *openpgp.Entity, err error) {

	if len(p.config.EncryptionKey) > 0 {

		encryptionKey, err = p.keyFromFile(p.config.EncryptionKey, false)
		if err != nil {
			p.log.Errorf("Unable to load the encryption key %s ", p.config.EncryptionKey)
			return nil, err
		}
		return
	}
	// encryption key has not been specified in the configuration
	return
}

//GetEncryptionKey Returns the currently configured signing key
func (p provider) GetSigningKey() (signingKey *openpgp.Entity, err error) {

	c := p.config
	if len(c.SigningKey) > 0 {
		p.log.Debugf("Signing with %s", c.SigningKey)
		if len(c.SigningKeyPassword) > 0 {
			signingKey, err = p.decryptArmoredKey(c.SigningKey, c.SigningKeyPassword)
			if err != nil {
				return
			}

		} else {
			signingKey, err = p.keyFromFile(c.SigningKey, false)
			if err != nil {
				return
			}
		}

		p.log.Debug("Signing key loaded ")
		return
	}
	// signing key has not been specified in the configuration
	return
}

//EncryptFile provides a simple wrapper to encrypt a file
func (p provider) EncryptFile(plainTextFile string, outputFile string) (err error) {
	p.log.Debugf("Encrypting file %s", plainTextFile)
	p.log.Debugf("Output file %s", outputFile)
	p.log.Debugf("Using EncryptionKey %s ", p.config.EncryptionKey)

	// Read in public key
	recipientKey, err := p.GetEncryptionKey()
	if err != nil {
		return
	}

	signingKey, err := p.GetSigningKey()
	if err != nil {
		return
	}

	inFile, err := os.Open(plainTextFile)
	if err != nil {
		return err
	}

	outFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	p.log.Debug("Performing Encryption ")

	hints := &openpgp.FileHints{
		IsBinary: true,
		FileName: "",
	}

	// create new default
	packConfig := &packet.Config{
		DefaultHash:            crypto.SHA1,
		DefaultCompressionAlgo: packet.CompressionZLIB,
	}

	// @todo currently uses defaults, should we provide other encryption options?
	recipientKeys := []*openpgp.Entity{recipientKey}
	wc, err := openpgp.Encrypt(outFile, recipientKeys, signingKey, hints, packConfig)
	if err != nil {
		return err
	}

	bytes, err := io.Copy(wc, inFile)

	// close the encrypted text
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

	p.log.Debugf("Encrypted file to %s", plainTextFile)

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
func (p provider) keyFromFile(fileName string, subkeyWorkAround bool) (*openpgp.Entity, error) {

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
