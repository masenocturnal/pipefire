package sftp

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

//Endpoint is an instance of the SFTP Connection Details
type Endpoint struct {
	Host        string `json:"host"`
	Key         string `json:"key"`
	UserName    string `json:"username"`
	Password    string `json:"password"`
	KeyPassword string `json:"key_password"`
	Port        string `json:"port"`
}

//FileTransferConfirmation is a summmary of the transferred file
type FileTransferConfirmation struct {
	LocalFileName    string
	RemoteFileName   string
	LocalSize        int64
	LocalHash        string
	RemoteSize       int64
	TransferredHash  string
	TransferredBytes int64
}

type transport struct {
	Client  *sftp.Client
	Session *ssh.Client
	Name    string
	log     *log.Entry
}

// Transport is the accessible type for the sftp connection
type Transport interface {
	SendFile(string, string) (*FileTransferConfirmation, error)
	SendDir(string, string) (*list.List, *list.List)
	ListRemoteDir(remoteDir string) error
	GetFile(remoteFile string, localFile string) (*FileTransferConfirmation, error)
	GetDir(remoteDir string, localDir string) (*list.List, *list.List)
	CleanDir(string) error
	RemoveDir(string) error
	RemoveFile(string) error
	Close()
}

//NewConnection establish a connection
func NewConnection(name string, conf Endpoint, correlationID string) (Transport, error) {
	var transport transport

	var authMethod []ssh.AuthMethod = make([]ssh.AuthMethod, 0)

	if len(conf.Key) > 0 {
		keyAuth, err := getPrivateKeyAuthentication(conf.Key, conf.KeyPassword)
		if err != nil {
			return transport, err
		}
		authMethod = append(authMethod, keyAuth)
	}
	if len(conf.Password) > 0 {
		authMethod = append(authMethod, ssh.Password(conf.Password))
	}

	// attempt to connect
	connDetails := &ssh.ClientConfig{
		User:            conf.UserName,
		Auth:            authMethod,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// max time to establish connection
		//HostKeyCallback: ssh.FixedHostKey(hostKey),
	}
	connDetails.SetDefaults()

	// @todo validate config
	if conf.Host == "" {
		return transport, fmt.Errorf("Host has not been set for %s", name)
	}
	if conf.Port == "" {
		log.Println("Port not set, using 22")
		conf.Port = "22"
	}

	connectionString := conf.Host + ":" + conf.Port
	log.Infof("Attempting to connect to %s ", connectionString)

	// connect
	sshClient, err := ssh.Dial("tcp", connectionString, connDetails)
	if err != nil {
		return nil, err
	}
	transport.Session = sshClient

	go func() {
		err := transport.Session.Wait()
		log.Info("Connection Closed")
		log.Error(err.Error())
	}()

	// create new SFTP client
	transport.Client, err = sftp.NewClient(transport.Session)
	if err != nil {
		return nil, err
	}
	log.Printf("Connnected to %s \n", connectionString)
	transport.Name = name
	transport.log = log.WithField("correlationId", correlationID)

	return transport, err
}

//CleanDir will recursively iterate through the directories
//and remove any files in them leaving the directory structure in place
func (c transport) CleanDir(remoteDir string) error {
	log := c.log.WithField("Remote Directory", remoteDir)
	log.Infof("Attempting to clean remote directory: %s", remoteDir)

	// loop through the directory
	// and get all files and directories
	if _, err := c.Client.ReadDir(remoteDir); err != nil {
		return err
	}

	// loop through the directory
	// and get all files and directories
	filesInDir, err := c.Client.ReadDir(remoteDir)
	if err != nil {
		return err
	}
	var lastError error
	// only read it there is something to read
	if len(filesInDir) > 0 {
		// loop throught the files
		for _, file := range filesInDir {
			currentRemoteFilePath := filepath.Join(remoteDir, file.Name())

			if file.IsDir() {
				lastError = c.CleanDir(currentRemoteFilePath)

			} else {
				lastError = c.RemoveFile(currentRemoteFilePath)
			}
		}
	}
	if lastError == nil {
		log.Info("Completed")
	}
	return lastError
}

//RemoveDir wrapper arround the underlying SFTP Client RemoveDir function
func (c transport) RemoveDir(remoteDir string) error {
	err := c.Client.RemoveDirectory(remoteDir)
	if err == nil {
		c.log.Printf("Removed")
	}
	return err
}

//RemoveFile wrapper arround the underlying SFTP Client Remove function
func (c transport) RemoveFile(remoteFile string) error {
	c.log.Infof("Attempting to delete file %s@%s: ", remoteFile, c.Name)
	err := c.Client.Remove(remoteFile)
	if err == nil {
		c.log.Info("Removed")
	}
	return err
}

//	GetFile Acquires a file from the remote service
func (c transport) GetFile(remotePath string, localPath string) (*FileTransferConfirmation, error) {
	xfer := &FileTransferConfirmation{}
	c.log.Infof("Attempting to get: %s@%s to %s@local: ", remotePath, c.Name, localPath)
	// create a hash writer so that we can create a hash as we are
	// copying the files
	hashWriter := sha256.New()

	// make sure local file exists
	// this can be a directory
	localFileInfo, err := os.Stat(localPath)
	if err != nil {
		return xfer, err
	}

	// check the remote file
	remoteFile, err := c.Client.Lstat(remotePath)
	if err != nil {
		return xfer, err
	}
	if remoteFile.IsDir() {
		return xfer, fmt.Errorf("Remote  file %s is a directory, call GetDir()", remotePath)
	}

	// ignore symlinks for now
	if remoteFile.Mode()&os.ModeSymlink != 0 {
		return nil, err
	}

	xfer.RemoteFileName = remotePath
	xfer.RemoteSize = remoteFile.Size()

	// if the local file is a directory then we can write into it
	if localFileInfo.IsDir() {
		localPath = filepath.Join(localPath, remoteFile.Name())
	}

	// All good, now download it
	dstFile, err := os.Create(localPath)
	if err != nil {
		return xfer, err
	}
	defer dstFile.Close()

	// open source file
	sourceReader, err := c.Client.Open(remotePath)
	multiWriter := io.MultiWriter(dstFile, hashWriter)

	// copy source file to destination file
	bytes, err := io.Copy(multiWriter, sourceReader)
	xfer.TransferredBytes = bytes
	if err != nil {
		return xfer, err
	}
	// flush in-memory copy
	err = dstFile.Sync()
	if err != nil {
		return xfer, err
	}
	xfer.TransferredHash = hex.EncodeToString(hashWriter.Sum(nil))
	xfer.LocalFileName = localPath

	// examine the file again to read the bytes
	localFileInfo, err = os.Stat(localPath)
	if err != nil {
		return xfer, err
	}
	xfer.LocalSize = localFileInfo.Size()

	// read it back to hash the file
	contents, _ := ioutil.ReadFile(localPath)

	// we've used the hashWriter prevously so it needs to be reset
	hashWriter.Reset()
	hashWriter.Write(contents)
	xfer.LocalHash = hex.EncodeToString(hashWriter.Sum(nil))

	c.log.Infof("Transferred \n")
	return xfer, err
}

func (c transport) GetDir(remoteDir string, localDir string) (confirmationList *list.List, errorList *list.List) {
	confirmationList = list.New()
	errorList = list.New()

	c.log.Infof("Attempting to GetDir %s to %s ", remoteDir, localDir)
	err := os.MkdirAll(localDir, 0700)
	if err != nil {
		errorList.PushFront(err)
		return
	}

	r, err := c.Client.Stat(remoteDir)
	if err != nil {
		errorList.PushFront(err)
		return
	}

	if !r.IsDir() {
		// remote end is a file...but we have a local directory
		// to stash it im, so let's just make it work
		confirmation, err := c.GetFile(remoteDir, localDir)
		if err != nil {
			errorList.PushFront(err)
		}
		confirmationList.PushFront(confirmation)
		// we should probably bail here as it's not a directory
		return
	}

	// remotePath is a directory, so is the localPath
	// create the remote directory name within the local path
	localDir = filepath.Join(localDir, r.Name())

	// try and make the directory if it doesn't exist
	err = os.MkdirAll(localDir, r.Mode())
	if err != nil {
		errorList.PushFront(err)
		return
	}

	// loop through the directory
	// and get all files and directories
	filesInDir, err := c.Client.ReadDir(remoteDir)
	if err != nil {
		errorList.PushFront(err)
		return
	}

	// only read it there is something to read
	if len(filesInDir) > 0 {
		// loop throught the files
		for _, file := range filesInDir {
			currentRemoteFilePath := filepath.Join(remoteDir, file.Name())

			if file.IsDir() {
				confirmations, errList := c.GetDir(currentRemoteFilePath, filepath.Join(localDir, file.Name()))
				if err != nil && errList.Len() > 0 {
					for temp := errList.Front(); temp != nil; temp = temp.Next() {
						errorList.PushFront(temp.Value)
					}
				}

				confirmationList.PushFrontList(confirmations)
			} else if file.Mode()&os.ModeSymlink != 0 {
				// ignore symlinks
				continue
			} else {
				confirmation, err := c.GetFile(currentRemoteFilePath, localDir)
				if err != nil {
					errorList.PushFront(err)
				}
				confirmationList.PushFront(confirmation)
			}
		}
	}

	return confirmationList, errorList
}

// SendFile will transfer the srcPath to the destPath on the server defined by the serviceID
// returns number of bytes transferred
func (c transport) SendFile(localPath string, remotePath string) (*FileTransferConfirmation, error) {
	xfer := &FileTransferConfirmation{}

	c.log.Infof("Attempting to send: %s to %s@%s: ", localPath, remotePath, c.Name)
	// create a hash writer so that we can create a hash as we are
	// copying the files
	hashWriter := sha256.New()

	// make sure local file exists
	localFileInfo, err := os.Stat(localPath)
	if err != nil {
		return xfer, err
	}
	if localFileInfo.IsDir() {
		return xfer, errors.New(localPath + ": is a directory. Call SendDir()")
	}
	xfer.LocalFileName = filepath.Join(localPath, localFileInfo.Name())
	xfer.LocalSize = localFileInfo.Size()

	// ensure we can read the local file first before we create the remote file
	data, err := ioutil.ReadFile(localPath)

	// calculate local checksum
	_, err = hashWriter.Write(data)
	if err != nil {
		return xfer, err
	}
	xfer.LocalHash = hex.EncodeToString(hashWriter.Sum(nil))

	// reset the hashWriter so that we can use the same writer for the remote file
	hashWriter.Reset()

	// get the SFTP Client connectied to the destination server
	client := c.Client
	if err != nil {
		return xfer, err
	}

	// see if the remote file exists..
	p, err := client.Stat(remotePath)
	if err != nil {
		return xfer, fmt.Errorf("Can't stat %s : %s  ", remotePath, err.Error())
	}

	if p != nil {
		// lets see if it's a directory
		if p.IsDir() {
			// write into the directory with file name
			remotePath = remotePath + localFileInfo.Name()
			c.log.Infof("Writing to remote server %s: %s \n", c.Name, remotePath)
		} else {
			// file exists already...replace ?
			c.log.Info("Remote file already exists. Replacing")
		}
	}

	// Create the remote file for writing
	remoteFile, err := client.Create(remotePath)
	if err != nil {
		return xfer, err
	}
	xfer.RemoteFileName = remoteFile.Name()

	// write the bytes to the remote file _and_ the hash writer at the same time
	// @todo use TeeReader ?
	multiwriter := io.MultiWriter(remoteFile, hashWriter)
	transferredBytes, err := multiwriter.Write(data)
	xfer.TransferredBytes = int64(transferredBytes)
	xfer.TransferredHash = hex.EncodeToString(hashWriter.Sum(nil))

	// sometimes SFTP Servers will lock or whisk away the file after the
	// file handle has closed
	remoteFileInfo, err := client.Stat(remotePath)
	if err != nil {
		c.log.Printf("Error getting size of remote file after transfer, file may have been locked or moved ")
	} else {
		xfer.RemoteSize = remoteFileInfo.Size()
	}
	c.log.Info("Transferred")
	return xfer, err
}

//ListRemoteDir Lists the files in a remote directory
func (c transport) ListRemoteDir(remoteDir string) error {
	// get the SFTP Client connectied to the destination server
	client := c.Client

	// list the directory
	w, err := client.ReadDir(remoteDir)
	if err != nil {
		return err
	}
	for _, file := range w {
		fmt.Println(file.Name())
	}
	return err
}

func (c transport) SendDir(srcDir string, destDir string) (confirmationList *list.List, errorList *list.List) {

	confirmationList = list.New()
	errorList = list.New()

	// get the SFTP Client connectied to the destination server
	client := c.Client

	// make sure what we have is a directory and it's accessible
	localDir, err := os.Stat(srcDir)
	if err != nil {
		errorList.PushFront(err)
		return
	}

	if !localDir.IsDir() {
		confirmation, err := c.SendFile(srcDir, destDir)
		if err != nil {
			errorList.PushFront(err)
		}
		confirmationList.PushFront(confirmation)
	}

	filesInDir, err := ioutil.ReadDir(srcDir)
	if err != nil {
		errorList.PushFront(err)
		return
	}

	// try and make the directory if it doesn't exist
	err = client.MkdirAll(destDir)
	if err != nil {
		errorList.PushFront(err)
		return
	}

	// only read it there is something to read
	if len(filesInDir) > 0 {
		// loop throught the files
		for _, file := range filesInDir {
			currentFilePath := filepath.Join(srcDir, file.Name())

			if file.IsDir() {
				confirmations, errList := c.SendDir(currentFilePath, filepath.Join(destDir, file.Name()))
				if err != nil && errList.Len() > 0 {
					for temp := errList.Front(); temp != nil; temp = temp.Next() {
						errorList.PushFront(temp.Value)
					}
				}
				confirmationList.PushFrontList(confirmations)
			} else {
				confirmation, err := c.SendFile(currentFilePath, destDir)
				if err != nil {
					errorList.PushFront(err)
				}
				confirmationList.PushFront(confirmation)
			}
		}
	}
	return
}

func (c transport) handleReconnects() {
	closed := make(chan error, 1)
	go func() {
		closed <- c.Session.Wait()
	}()
	c.log.Printf("Here %v ", closed)
	c.log.Println("IN HERE")
}

//Close closes
func (c transport) Close() {
	c.Client.Close()
	c.Session.Close()

}
