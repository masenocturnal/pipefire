package sftp

import (
	"fmt"
	"io/ioutil"
	"os"

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
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyInBytes, []byte(keyPassword))
	} else {
		signer, err = ssh.ParsePrivateKey(keyInBytes)
	}
	if err != nil {
		return nil, err
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
