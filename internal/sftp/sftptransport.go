package sftp

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

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
	TransferredBytes int
}

type transport struct {
	client *sftp.Client
	Name   string
}

// Transport is the accessible type for the sftp connection
type Transport interface {
	SendFile(string, string) (*FileTransferConfirmation, error)
	SendDir(string, string) error
	ListRemoteDir(remoteDir string) error
	Close()
}

//NewConnection establish a connection
func NewConnection(name string, conf Endpoint) (Transport, error) {
	var transport transport

	// attempt to connect
	connDetails := &ssh.ClientConfig{
		User: conf.UserName,
		Auth: []ssh.AuthMethod{
			ssh.Password(conf.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		//HostKeyCallback: ssh.FixedHostKey(hostKey),
	}

	if conf.Port == "" {
		log.Print("Port not set, using 22")
		conf.Port = "22"
	}
	connectionString := conf.Host + ":" + conf.Port
	log.Printf("Attempting to connect to %s", connectionString)

	// connect
	connection, err := ssh.Dial("tcp", connectionString, connDetails)
	if err != nil {
		return nil, err
	}

	// create new SFTP client
	client, err := sftp.NewClient(connection)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	transport.client = client
	transport.Name = name
	return transport, err
}

// SendFile will transfer the srcPath to the destPath on the server defined by the serviceID
// returns number of bytes transferred
func (c transport) SendFile(srcPath string, destPath string) (*FileTransferConfirmation, error) {
	xfer := &FileTransferConfirmation{}

	// create a hash writer so that we can create a hash as we are
	// copying the files
	hashWriter := sha256.New()

	// make sure local file exists
	localFileInfo, err := os.Stat(srcPath)
	if err != nil {
		return xfer, err
	}
	if localFileInfo.IsDir() {
		return xfer, errors.New(srcPath + ": is a directory. Call SendDir()")
	}
	xfer.LocalFileName = localFileInfo.Name()
	xfer.LocalSize = localFileInfo.Size()

	// ensure we can read the local file first before we create the remote file
	data, err := ioutil.ReadFile(srcPath)

	// calculate local checksum
	_, err = hashWriter.Write(data)
	if err != nil {
		return xfer, err
	}
	xfer.LocalHash = hex.EncodeToString(hashWriter.Sum(nil))

	// reset the hashWriter so that we can use the same writer for the remote file
	hashWriter.Reset()

	// get the SFTP Client connectied to the destination server
	client := c.client
	if err != nil {
		return xfer, err
	}
	defer client.Close()

	// see if the remote file exists..
	p, _ := client.Stat(destPath)
	if p != nil {
		// lets see if it's a directory
		if p.IsDir() {
			// write into the directory with file name
			destPath = destPath + localFileInfo.Name()
			fmt.Printf("Writing to remote server %s: %s", c.Name, destPath)
		} else {
			// file exists already...replace ?
			log.Print("Remote file already exists. Replacing")
		}
	}

	// Create the remote file for writing
	remoteFile, err := client.Create(destPath)
	if err != nil {
		return xfer, err
	}
	xfer.RemoteFileName = remoteFile.Name()

	// write the bytes to the remote file _and_ the hash writer at the same time
	// @todo use TeeReader ?
	multiwriter := io.MultiWriter(remoteFile, hashWriter)
	xfer.TransferredBytes, err = multiwriter.Write(data)
	xfer.TransferredHash = hex.EncodeToString(hashWriter.Sum(nil))

	// sometimes SFTP Servers will lock or whisk away the file after the
	// file handle has closed
	remoteFileInfo, err := client.Stat(destPath)
	if err != nil {
		log.Printf("Error getting size of remote file after transfer, file may have been locked or moved ")
	} else {
		xfer.RemoteSize = remoteFileInfo.Size()
	}

	return xfer, err

}

func (c transport) ListRemoteDir(remoteDir string) error {
	// get the SFTP Client connectied to the destination server
	client := c.client

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

func (c transport) SendDir(srcDir string, destDir string) error {

	return errors.New("dummy")
}

//Close closes
func (c transport) Close() {
	c.client.Close()
}
