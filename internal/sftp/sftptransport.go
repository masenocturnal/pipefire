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
	"os/user"
	"path/filepath"
	"strings"

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
	KeyPassword string `json:"keyPassword"`
	Port        int64  `json:"port"`
}

//FileTransferConfirmation is a summmary of the transferred file
type FileTransferConfirmation struct {
	LocalFileName    string
	LocalPath        string
	RemoteFileName   string
	RemotePath       string
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
func NewConnection(name string, conf Endpoint, log *log.Entry) (Transport, error) {
	var transport transport

	var authMethod []ssh.AuthMethod = make([]ssh.AuthMethod, 0)

	if len(conf.Key) > 0 {

		if strings.Index(conf.Key, "~") == 0 {
			usr, err := user.Current()
			if err != nil {
				return nil, err
			}

			conf.Key = strings.Replace(conf.Key, "~", usr.HomeDir, 1)
		}
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
	if conf.Port == 0 {
		log.Println("Port not set, using 22")
		conf.Port = 22
	}

	connectionString := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	log.Infof("Attempting to connect to %s ", connectionString)

	// connect
	sshClient, err := ssh.Dial("tcp", connectionString, connDetails)
	if err != nil {
		return nil, err
	}
	transport.Session = sshClient

	go func() {
		log.Debug("Connection Closed")
	}()

	opts := sftp.MaxConcurrentRequestsPerFile(1)

	// create new SFTP client
	transport.Client, err = sftp.NewClient(transport.Session, opts)
	if err != nil {
		return nil, err
	}
	log.Printf("Connnected to %s ", connectionString)
	transport.Name = name
	transport.log = log

	return transport, err
}

//CleanDir will recursively iterate through the directories
//and remove any files in them leaving the directory structure in place
func (c transport) CleanDir(remoteDir string) error {
	log := c.log.WithField("Remote Directory", remoteDir)
	log.Debugf("Attempting to clean remote directory: %s", remoteDir)

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
		log.Debug("Completed")
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
	c.log.Debugf("Attempting to delete file %s@%s: ", remoteFile, c.Name)
	err := c.Client.Remove(remoteFile)
	if err == nil {
		c.log.Debug("Removed")
	}
	return err
}

//	GetFile Acquires a file from the remote service
func (c transport) GetFile(remotePath string, localPath string) (*FileTransferConfirmation, error) {
	xfer := &FileTransferConfirmation{}
	c.log.Debugf("Attempting to get: %s@%s to %s@local: ", remotePath, c.Name, localPath)
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

		return nil, fmt.Errorf("File %s: %s", remotePath, err.Error())
	}
	if remoteFile.IsDir() {
		return xfer, fmt.Errorf("Remote  file %s is a directory, call GetDir()", remotePath)
	}

	// ignore symlinks for now
	if remoteFile.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("File %s is a symlink..ignoring. %s", remotePath, err.Error())
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

	c.log.Debugln("Transferred")
	return xfer, err
}

func (c transport) GetDir(remoteDir string, localDir string) (confirmationList *list.List, errorList *list.List) {
	confirmationList = list.New()
	errorList = list.New()

	c.log.Debugf("Attempting to GetDir %s to %s ", remoteDir, localDir)
	err := os.MkdirAll(localDir, 0700)
	if err != nil {
		errorList.PushFront(err)
		return
	}

	r, err := c.Client.Stat(remoteDir)
	if err != nil {
		errorList.PushFront(fmt.Errorf("Remote file : %s : %s", remoteDir, err.Error()))
		return
	}

	if !r.IsDir() {
		// remote end is a file...but we have a local directory
		// to stash it in, so let's just make it work
		confirmation, err := c.GetFile(remoteDir, localDir)
		if err != nil {
			errorList.PushFront(fmt.Errorf("Remote file : %s : %s", remoteDir, err.Error()))
		}
		confirmationList.PushFront(confirmation)
		// we should probably bail here as it's not a directory
		return
	}

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
				newLocalFilePath := filepath.Join(localDir, file.Name())
				confirmations, errList := c.GetDir(currentRemoteFilePath, newLocalFilePath)
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

	if len(remotePath) == 0 || len(localPath) == 0 {
		err := fmt.Errorf("Either the local path %s: or the remotePath is emtpy: %s", localPath, remotePath)
		c.log.Errorf(err.Error())
		return xfer, err
	}

	c.log.Debugf("Attempting to send: localPath: %s to remotePath: %s ", localPath, remotePath)
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
	xfer.LocalFileName = localFileInfo.Name()
	xfer.LocalSize = localFileInfo.Size()
	xfer.LocalPath = localPath

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
	c.log.Debugf("Stat %s ", remotePath)
	if err != nil {
		c.log.Debugf("Remote file %s doesn't exist", remotePath)
	}

	if p != nil {
		if p.IsDir() {
			remotePath = filepath.Join(remotePath, localFileInfo.Name())
			c.log.Debugf("Is dir %s ", remotePath)
		} else {
			fileMode := p.Mode()
			if fileMode.IsRegular() {
				c.log.Debugf("%s is a regular file : ", fileMode.String())
			} else {
				remotePath = filepath.Join(remotePath, localFileInfo.Name())
				c.log.Debugf("Mode something else : %v, %s", fileMode.IsRegular(), fileMode.String())
			}
		}
	} else {
		c.log.Debugf("remote %s doesn't exist, attempting to create it", remotePath)

		if err := client.MkdirAll(remotePath); err != nil {
			c.log.Errorf("remote %s is either a file or you do not have permission, err : %s", remotePath, err.Error())
			c.log.Debugf(err.Error())
			return xfer, err
		}
	}

	c.log.Debugf("Trying to create %s", remotePath)
	// Create the remote file for writing
	//remoteFile, err := client.Create(remotePath)
	remoteFile, err := client.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		c.log.Errorf("Unable to create file %s , %s", remotePath, err.Error())
		// do we close the connection here ?
		return xfer, err
	}

	xfer.RemoteFileName = remoteFile.Name()
	xfer.RemotePath = remotePath

	// write the bytes to the remote file _and_ the hash writer at the same time
	// @todo use TeeReader ?
	multiwriter := io.MultiWriter(remoteFile, hashWriter)

	// actually write the packets
	transferredBytes, err := multiwriter.Write(data)

	// close the connection
	err = remoteFile.Close()
	if err != nil {
		c.log.Debug("File successfully closed on the remote end", remotePath, err.Error())
		c.log.Errorf("Error writing %s. Error: %s", remotePath, err.Error())
		c.log.Error("File DID NOT TRANSFER.", remotePath, err.Error())
	} else {
		xfer.TransferredBytes = int64(transferredBytes)
		xfer.TransferredHash = hex.EncodeToString(hashWriter.Sum(nil))
		xfer.RemoteSize = xfer.TransferredBytes
	}

	// sometimes SFTP Servers will lock or whisk away the file after the
	// file handle has closed
	// I think some sftp servers have issues with this
	remoteFileInfo, err := client.Stat(remotePath)
	if err != nil {
		c.log.Warnf("Error getting size of remote file after transfer, file may have been locked or moved ")
	} else {
		xfer.RemoteSize = remoteFileInfo.Size()
	}

	c.log.Debug("Transferred")
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

	c.log.Infof("Listing Remote directory %s after transfer", remoteDir)
	var sb strings.Builder
	for _, file := range w {

		sb.WriteString(file.Name() + "\n")
	}
	c.log.Info(sb.String())
	return err
}

func (c transport) SendDir(srcDir string, destDir string) (confirmationList *list.List, errorList *list.List) {

	confirmationList = list.New()
	errorList = list.New()

	if len(srcDir) == 0 || len(destDir) == 0 {
		errorList.PushFront(fmt.Errorf("Either the srcDir %s, or the destDir: %s is not present but both are required", srcDir, destDir))
		return
	}

	// get the SFTP Client connectied to the destination server
	client := c.Client

	// make sure what we have is a directory and it's accessible
	localDir, err := os.Stat(srcDir)
	if err != nil {

		errorList.PushFront(fmt.Errorf("Unable to read local directory %s ", srcDir))
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

	// see if directory exist
	rdir, err := client.Stat(destDir)
	if err != nil {
		c.log.Warnf("Directory: %s doesn't exist. We will try to make the directory", destDir)
	}

	if rdir == nil {
		// try and make the directory if it doesn't exist
		err = client.MkdirAll(destDir)
		if err != nil {

			// It's been reported that some sftp servers fail on this however it could just
			// be because the directory already exists
			// @todo see
			c.log.Debugf("Failed trying to MkdirAll: %s", destDir)
			c.log.Warnf("Unable to create remote directory: %s Error: %s", destDir, err.Error())
			errorList.PushFront(err)
			return
		}
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
