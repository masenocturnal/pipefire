package directdebit

import (
	"archive/tar"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

//ArchiveConfig configuration for the archive task
type ArchiveConfig struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
}

func (d ddPipeline) archiveTransferred(conf *ArchiveConfig) (err error) {

	src, err := os.Stat(conf.Src)
	if err != nil {
		return fmt.Errorf("Unable to read: %s. %s", conf.Src, err.Error())
	}

	var fileList []string
	if src.IsDir() {
		// read the dir
		fileList, err := ioutil.ReadDir(conf.Src)
		if err != nil {
			return err
		}
	} else {
		fileList = []string{
			conf.Src,
		}
	}

	return err
}

func (d ddPipeline) createTar(filePaths []string, dest string) (errors []error) {
	// Create and add some files to the archive.
	os.Create()

	tw := tar.NewWriter(&buf)

	for _, file := range filePaths {

		s, err := os.Stat(file)
		if err != nil {
			d.log.Error("File %s can't be read %s", file, err.Error())
			return append(errors, err)
		}

		hdr := &tar.Header{
			Name: file,
			Mode: 0600,
			Size: s.Size(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			log.Fatal(err)
		}
		contents, err := ioutil.ReadFile(file)
		if err != nil {
			d.log.Error("Error reading file: %s", file, err.Error())
			return append(errors, err)
		}
		if _, err := tw.Write(contents); err != nil {
			d.log.Error("Unable to close tar writer %s ", err.Error())
			return append(errors, err)
		}
	}
	if err := tw.Close(); err != nil {
		d.log.Error("Unable to close tar writer %s ", err.Error())
		return append(errors, err)
	}
	return
}
