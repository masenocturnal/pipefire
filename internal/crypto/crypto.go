package crypto

import (
	"io"
	"os"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

//Provider is an instance of the SFTP Connection Details
type Provider struct {
	PublicKey          string `json:"PublicKey"`
	FingerPrint        string `json:"FingerPrint"`
	PrivateKey         string `json:"PrivateKey"`
	PrivateKeyPassword string `json:"PrivateKeyPassword"`
}

// change as required
const pubKey = "/tmp/pubKey.asc"
const fileToEnc = "/tmp/data.txt"

//NewRrovider returns a Crypto Provider
func NewProvider() {

}

//EncryptFile provides a simple wrapper to encrypt a file
func (p Provider) EncryptFile(filePath string, encryptedFile string) error {
	//recip []*openpgp.Entity, signer *openpgp.Entity, r io.Reader, w io.Writer
	wc, err := openpgp.Encrypt(w, recip, signer, &openpgp.FileHints{IsBinary: true}, nil)
	if err != nil {
		return err
	}
	if _, err := io.Copy(wc, r); err != nil {
		return err
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
