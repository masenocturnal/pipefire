package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

//HashFile calclates checksun for a file
func HashFile(fileName string) (string, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
