package directdebit

import (
	"encoding/json"
	"fmt"

	"github.com/masenocturnal/pipefire/internal/sftp"
)

//SftpConfig Required Params for transferring to or from an SFTP Server
type SftpConfig struct {
	RemoteDir string        `json:"remoteDir"`
	LocalDir  string        `json:"localDir"`
	Sftp      sftp.Endpoint `json:"sftp"`
	Enabled   bool          `json:"enabled"`
}

// get files from a particular endpoint
func (p pipeline) sftpGet(conf SftpConfig) error {
	p.log.Infof("Begin sftpGet: %s ", conf.Sftp.Host)
	sftp, err := sftp.NewConnection("From", conf.Sftp, p.log)
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
		result, _ := json.MarshalIndent(temp.Value, "", " ")
		p.log.Info(string(result))
	}

	p.log.Info("sftpGet Complete")
	return err
}

// sftpClean cleans the repote directory
func (p pipeline) sftpClean(conf SftpConfig) (err error) {
	p.log.Infof("Begin sftpClean: %s", conf.Sftp.Host)
	p.log.Debugf("Cleaning remote dir: %s ", conf.RemoteDir)

	sftp, err := sftp.NewConnection("To", conf.Sftp, p.log)
	if err != nil {
		return
	}
	defer sftp.Close()

	err = sftp.CleanDir(conf.RemoteDir)
	if err == nil {
		p.log.Infof("sftpClean Complete: Removed files from: %s ", conf.RemoteDir)
	}
	return err
}

// send files to a particular endpoint
func (p pipeline) sftpTo(conf SftpConfig) (err error) {
	p.log.Infof("Begin sftpTo: %s", conf.Sftp.Host)
	p.log.Debugf("Sftp transfer from %s to %s @ %s ", conf.LocalDir, conf.RemoteDir, conf.Sftp.Host)

	sftp, err := sftp.NewConnection("To", conf.Sftp, p.log)
	if err != nil {
		return
	}

	defer sftp.Close()

	confirmations, errors := sftp.SendDir(conf.LocalDir, conf.RemoteDir)
	if errors.Len() > 0 {
		// show all errors
		for temp := errors.Front(); temp != nil; temp = temp.Next() {
			p.log.Error(temp.Value)
		}
		return fmt.Errorf("Error Sending files to %s ", conf.RemoteDir)
	}

	for temp := confirmations.Front(); temp != nil; temp = temp.Next() {
		result, _ := json.MarshalIndent(temp.Value, "", " ")
		p.log.Info(string(result))
	}

	p.log.Infof("sftpTo Complete, remote %s ", conf.RemoteDir)
	return nil
}
