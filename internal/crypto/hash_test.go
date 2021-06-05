package crypto

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestHashFile(t *testing.T) {
	// generate a random file with known contents
	d, err := ioutil.TempDir("/tmp/", "pipefire_test")
	if err != nil {
		t.Errorf(err.Error())
	}

	f, err := ioutil.TempFile(d, "test_hash")
	if err != nil {
		t.Errorf(err.Error())
	}
	defer f.Close()
	n := f.Name()
	fmt.Printf("File is %s ", n)

	f.WriteString("Hello World")

	checkSum, err := HashFile(f.Name())
	if err != nil {
		t.Errorf(err.Error())
	}
	expectedHash := "a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e"
	if expectedHash != checkSum {
		t.Errorf("Hash %s does not match expected hash of %s ", checkSum, expectedHash)
	}

}
