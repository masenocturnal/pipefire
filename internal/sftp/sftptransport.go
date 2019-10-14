package sftp

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
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
}

// Transport is the accessible type for the sftp connection
type Transport interface {
	SendFile(string, string) (*FileTransferConfirmation, error)
	SendDir(string, string) (*list.List, *list.List)
	ListRemoteDir(remoteDir string) error
	GetFile(string, string) (*FileTransferConfirmation, error)
	Close()
}

//NewConnection establish a connection
func NewConnection(name string, conf Endpoint) (Transport, error) {
	var transport transport

	keyAuth, err := getPrivateKeyAuthentication(conf.Key, conf.KeyPassword)
	if err != nil {
		return transport, err
	}

	// attempt to connect
	connDetails := &ssh.ClientConfig{
		User: conf.UserName,
		Auth: []ssh.AuthMethod{
			keyAuth,
			ssh.Password(conf.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// max time to establish connection
		//HostKeyCallback: ssh.FixedHostKey(hostKey),
	}
	connDetails.SetDefaults()

	if conf.Port == "" {
		log.Println("Port not set, using 22")
		conf.Port = "22"
	}
	connectionString := conf.Host + ":" + conf.Port
	log.Printf("Attempting to connect to %s \n", connectionString)

	// connect
	transport.Session, err = ssh.Dial("tcp", connectionString, connDetails)
	if err != nil {
		return nil, err
	}

	go func() {
		err := transport.Session.Wait()
		fmt.Println("Connection dropped")
		fmt.Println(err.Error())
	}()

	// create new SFTP client
	transport.Client, err = sftp.NewClient(transport.Session)
	if err != nil {
		return nil, err
	}
	transport.Name = name

	return transport, err
}

//GetFile Acquires a file from the remote service
func (c transport) GetFile(remotePath string, localPath string) (*FileTransferConfirmation, error) {
	xfer := &FileTransferConfirmation{}

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
	remoteFile, err := c.Client.Stat(remotePath)
	if err != nil {
		return xfer, err
	}

	if remoteFile.IsDir() {
		return xfer, fmt.Errorf("Remote  file %s is a directory, call GetDir()", remotePath)
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

	return xfer, err
}

// SendFile will transfer the srcPath to the destPath on the server defined by the serviceID
// returns number of bytes transferred
func (c transport) SendFile(localPath string, remotePath string) (*FileTransferConfirmation, error) {
	xfer := &FileTransferConfirmation{}

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
			fmt.Printf("Writing to remote server %s: %s \n", c.Name, remotePath)
		} else {
			// file exists already...replace ?
			log.Print("Remote file already exists. Replacing")
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
		log.Printf("Error getting size of remote file after transfer, file may have been locked or moved ")
	} else {
		xfer.RemoteSize = remoteFileInfo.Size()
	}

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
		errorList.PushFront(fmt.Errorf("The path %s is not a directory", srcDir))
		return
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

	if len(filesInDir) > 0 {
		// read the dir
		for _, file := range filesInDir {
			currentFilePath := filepath.Join(srcDir, file.Name())

			if file.IsDir() {
				confirmations, errList := c.SendDir(currentFilePath, filepath.Join(destDir, file.Name()))
				if err != nil {
					errorList.PushFrontList(errList)
				}

				confirmationList.PushFrontList(confirmations)
			} else {
				confirmation, err := c.SendFile(currentFilePath, destDir)
				if err != nil {
					errorList.PushFront(err)
				}
				confirmationList.PushFront(confirmation)
			}

			if err != nil {
				fmt.Println(err.Error())
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
	fmt.Printf("Here %v ", closed)
	fmt.Println("IN HERE")
}

//Close closes
func (c transport) Close() {
	c.Session.Close()
	c.Client.Close()
}
