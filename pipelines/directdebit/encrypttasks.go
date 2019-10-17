package directdebit

import (
	"io/ioutil"

	"github.com/masenocturnal/pipefire/internal/crypto"
)

// EncryptFilesConfig is the configuration requriements for the encryptFiles task
type EncryptFilesConfig struct {
	SrcDir    string                           `json:"srcDir"`
	Providers map[string]crypto.ProviderConfig `json:"providers"`
}

func (p pipeline) encryptFiles(config EncryptFilesConfig) (err error) {
	// Get Crypto Provider
	anzConfig := config.Providers["ANZ"]
	provider := crypto.NewProvider(anzConfig, p.correlationID)

	// loop through directories
	dir, err := ioutil.ReadDir(config.SrcDir)
	_ = dir
	fileName := config.SrcDir
	err = provider.EncryptFile(fileName, fileName+".gpg")
	if err != nil {
		return
	}

	// f, err := os.Open(fileToEnc)
	// if err != nil {
	// 	return
	// }
	// defer f.Close()

	// dst, err := os.Create(fileToEnc + ".gpg")
	// if err != nil {
	// 	return
	// }
	// defer dst.Close()
	// encrypt([]*openpgp.Entity{recipient}, nil, f, dst)
	return err
}
