package main

import (
	"encoding/json"
	"fmt"

	"github.com/masenocturnal/pipefire/internal/crypto"
	"github.com/masenocturnal/pipefire/internal/sftp"
	log "github.com/sirupsen/logrus"
)

// get files from a particular endpoint
func sftpFromTask(conf sftp.Endpoint, correlationID string) error {

	return fmt.Errorf("Not Implemented")
}

// send files to a particular endpoint
func sftpToTask(conf sftp.Endpoint, correlationID string) (err error) {

	sftp, err := sftp.NewConnection("connection1", conf, correlationID)
	if err != nil {
		return
	}

	defer sftp.Close()

	// // Get Remote File
	// foo, err := sftp.GetFile("/home/am/positivessl.zip", "/tmp/")
	// if err != nil {
	// 	return err
	// }

	// Get Remote Dir
	status, errors := sftp.GetDir("/home/am/nocturnal.net.au", "/tmp/foobar")
	if errors != nil {
		// show all errors
		for temp := errors.Front(); temp != nil; temp = temp.Next() {
			fmt.Println(temp.Value)
		}
	}

	if errors.Len() == 0 {
		if err := sftp.CleanDir("/home/am/nocturnal.net.au"); err != nil {
			return err
		}
	}

	// result, _ := json.MarshalIndent(foo, "", " ")
	// fmt.Println(string(result))

	// confirmations, errors := sftp.SendDir("/home/andmas/tmp/RefundFiles", "/home/ubuntu/tmp")
	// if errors != nil {
	// 	// show all errors
	// 	for temp := errors.Front(); temp != nil; temp = temp.Next() {
	// 		fmt.Println(temp.Value)
	// 	}
	// }

	// // show success
	// @todo loop recursive

	for temp := status.Front(); temp != nil; temp = temp.Next() {
		result, _ := json.MarshalIndent(temp.Value, "", " ")
		log.Debug(string(result))
	}

	// if err != nil {
	// 	contextLogger.Error(err.Error())
	// 	return err
	// }

	return nil
}

func encryptTask(config crypto.ProviderConfig, correlationID string) (err error) {
	// Get Provider
	provider := crypto.NewProvider(config, correlationID)

	// @todo read from task config
	fileName := "/home/andmas/go/src/github.com/masenocturnal/pipefire/internal/crypto/test-file.txt"

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
