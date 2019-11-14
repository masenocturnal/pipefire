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
	Src     string `json:"src"`
	Dest    string `json:"dest"`
	Enabled bool   `json:"enabled"`
}

// ArchiveTransferred Creates a tar archive of the encrypted files.
// As the files are encrypted there is no point compressing them
func (d ddPipeline) archiveTransferred(conf *ArchiveConfig) (err error) {

	fileList, err := getFileList(conf.Src)

	errors := d.createTar(fileList, conf.Dest)
	if errors != nil && len(errors) > 0 {
		for _, e := range errors {
			d.log.Errorf("Error: %s ", e.Error())
		}
		return fmt.Errorf("Unable to create archive")
	}
	return
}

// recursively iterates through the files obtaining the full file path to create a canonical list of files to add to the archive
func getFileList(src string) ([]string, error) {

	fInfo, err := os.Stat(src)

	if err != nil {
		return nil, fmt.Errorf("Unable to read: %s. %s", src, err.Error())
	}

	var fileList []string
	if fInfo.IsDir() {
		// read the dir
		files, err := ioutil.ReadDir(src)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if f.IsDir() {
				dirPath := filepath.Join(src, f.Name())
				filesInDir, err := getFileList(dirPath)
				if err != nil {
					return nil, err
				}
				fileList = append(fileList, filesInDir...)

			} else {
				// regular files we will just add to the list
				fileList = append(fileList, filepath.Join(src, f.Name()))
			}
		}
	} else {
		fileList = []string{
			src,
		}
	}
	return fileList, err
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
