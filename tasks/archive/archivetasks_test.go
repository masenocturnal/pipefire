package archive

import (
	"io/ioutil"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestArchiveFiles(t *testing.T) {

	// make a directory and fill it will stuff
	src, err := ioutil.TempDir("/tmp", "pipefire_test")
	if err != nil {
		t.Fatal(err)
	}

	dest, err := ioutil.TempDir("/tmp", "pipefire_test")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		if _, err := ioutil.TempFile(src, "pipetest"); err != nil {
			t.Fatal(err)
		}
	}

	archiveConfig := &ArchiveConfig{
		Src:  src,
		Dest: dest,
	}

	// tasksConfig := &TasksConfig{
	// 	ArchiveTransferred: archiveConfig,
	// }
	// pipeline, err := getPipeline(tasksConfig)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	err = ArchiveTransferred(archiveConfig, logrus.Entry{})
	if err != nil {
		t.Error(err)
	}
}

func TestArchiveDir(t *testing.T) {

	// make a directory and fill it will stuff
	src, err := ioutil.TempDir("/tmp", "pipefire_test")
	if err != nil {
		t.Fatal(err)
	}

	// make a subdirectory
	srcSub, err := ioutil.TempDir(src, "pipefire_sub")
	if err != nil {
		t.Fatal(err)
	}

	dest, err := ioutil.TempDir("/tmp", "pipefire_archive")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		if _, err := ioutil.TempFile(srcSub, "pipetest"); err != nil {
			t.Fatal(err)
		}
	}

	archiveConfig := &ArchiveConfig{
		Src:  src,
		Dest: dest,
	}

	// tasksConfig := &TasksConfig{
	// 	ArchiveTransferred: archiveConfig,
	// }
	// pipeline, err := getPipeline(tasksConfig)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	err = ArchiveTransferred(archiveConfig, logrus.Entry{})
	if err != nil {
		t.Error(err)
	}
}
