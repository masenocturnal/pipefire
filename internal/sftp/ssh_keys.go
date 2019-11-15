package sftp

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ScaleFT/sshkeys"
	"golang.org/x/crypto/ssh"
)

func getPrivateKeyAuthentication(keyPath string, keyPassword string) (ssh.AuthMethod, error) {

	if !keyExists(keyPath) {
		return nil, fmt.Errorf("File: %s doesn't exist ", keyPath)
	}

	keyInBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	var signer ssh.Signer
	if len(keyPassword) > 0 {
		// signer, err = ssh.ParsePrivateKeyWithPassphrase(keyInBytes, []byte(keyPassword))
		signer, err = sshkeys.ParseEncryptedPrivateKey(keyInBytes, []byte(keyPassword))
	} else {
		signer, err = ssh.ParsePrivateKey(keyInBytes)
	}
	if err != nil {
		e := fmt.Errorf("Unable to decrypt private key. This could be because it is a new openssh-v1 encrypted key which is not currently supported. %s", err.Error())
		return nil, e
	}
	return ssh.PublicKeys(signer), err
}

func keyExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
