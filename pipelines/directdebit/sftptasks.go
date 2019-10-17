package directdebit

import (
	"fmt"

	"github.com/masenocturnal/pipefire/internal/sftp"
)

//SftpConfig Required Params for transferring to or from an SFTP Server
type SftpConfig struct {
	RemoteDir string        `json:"remoteDir"`
	LocalDir  string        `json:"localDir"`
	Sftp      sftp.Endpoint `json:"sftp"`
}

// get files from a particular endpoint
func (p pipeline) sftpGet(conf SftpConfig) error {
	p.log.Info("Get files from BFP")
	sftp, err := sftp.NewConnection("From", conf.Sftp, p.correlationID)
	if err != nil {
		return err
	}
	defer sftp.Close()

	// grab all the files from the pickup directory
	confirmations, errors := sftp.GetDir(conf.RemoteDir, conf.LocalDir)

	if errors.Len() > 0 {
		// show all errors
		for temp := errors.Front(); temp != nil; temp = temp.Next() {
			p.log.Error(temp.Value)
		}
		return fmt.Errorf("Error getting files from %s ", conf.RemoteDir)
	}

	for temp := confirmations.Front(); temp != nil; temp = temp.Next() {
		p.log.Info(temp.Value)
	}
	p.log.Info("Complete")
	return err
}

// send files to a particular endpoint
func (p pipeline) sftpTo(conf SftpConfig) (err error) {

	sftp, err := sftp.NewConnection("To", conf.Sftp, p.correlationID)
	if err != nil {
		return
	}

	defer sftp.Close()

	// // Get Remote Dir
	// status, errors := sftp.GetDir("/home/am/nocturnal.net.au", "/tmp/foobar")
	// if errors != nil {
	// 	// show all errors
	// 	for temp := errors.Front(); temp != nil; temp = temp.Next() {
	// 		fmt.Println(temp.Value)
	// 	}
	// }

	// if errors.Len() == 0 {
	// 	if err := sftp.CleanDir("/home/am/nocturnal.net.au"); err != nil {
	// 		return err
	// 	}
	// }

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

	// for temp := status.Front(); temp != nil; temp = temp.Next() {
	// 	result, _ := json.MarshalIndent(temp.Value, "", " ")
	// 	log.Debug(string(result))
	// }

	// if err != nil {
	// 	contextLogger.Error(err.Error())
	// 	return err
	// }

	return nil
}
