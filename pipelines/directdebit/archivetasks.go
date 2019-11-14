package directdebit

import (
	"archive/tar"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

//ArchiveConfig configuration for the archive task
type ArchiveConfig struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
}

// ArchiveTransferred Creates a tar archive of the encrypted files.
// As the files are encrypted there is no point compressing them
func (d ddPipeline) archiveTransferred(conf *ArchiveConfig) (err error) {

	src, err := os.Stat(conf.Src)
	if err != nil {
		return fmt.Errorf("Unable to read: %s. %s", conf.Src, err.Error())
	}

	var fileList []string
	if src.IsDir() {
		// read the dir
		files, err := ioutil.ReadDir(conf.Src)
		if err != nil {
			return err
		}
		for _, f := range files {
			fileList = append(fileList, filepath.Join(conf.Src, f.Name()))
		}
	} else {
		fileList = []string{
			conf.Src,
		}
	}

	errors := d.createTar(fileList, conf.Dest)
	if errors != nil && len(errors) > 0 {
		for _, e := range errors {
			d.log.Errorf("Error: %s ", e.Error())
		}
		return fmt.Errorf("Unable to create archive")
	}
	return
}

func (d ddPipeline) createTar(filePaths []string, destDir string) (errors []error) {

	if err := os.MkdirAll(destDir, 0760); err != nil {
		return append(errors, fmt.Errorf("Can't create destination directory %s : %s ", destDir, err.Error()))
	}

	// Create and add some files to the archive.
	archiveName := filepath.Join(destDir, time.Now().Format("2006-01-02")+".tar")

	f, err := os.Create(archiveName)
	if err != nil {
		return append(errors, fmt.Errorf("Unable to create file %s, %s", archiveName, err.Error()))
	}

	defer f.Close()

	tw := tar.NewWriter(f)

	for _, file := range filePaths {

		s, err := os.Stat(file)
		if err != nil {
			d.log.Errorf("File %s can't be read %s", file, err.Error())
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
			d.log.Errorf("Error reading file: %s : %s", file, err.Error())
			return append(errors, err)
		}
		if _, err := tw.Write(contents); err != nil {
			d.log.Errorf("Unable to close tar writer %s ", err.Error())
			return append(errors, err)
		}
	}
	if err := tw.Close(); err != nil {
		d.log.Errorf("Unable to close tar writer %s ", err.Error())
		return append(errors, err)
	}
	return
}
