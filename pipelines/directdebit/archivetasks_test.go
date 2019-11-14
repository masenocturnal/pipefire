package directdebit

import (
	"io/ioutil"
	"testing"
)

func TestArchive(t *testing.T) {

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

	archiveConfig := ArchiveConfig{
		Src:  src,
		Dest: dest,
	}

	tasksConfig := &TasksConfig{
		ArchiveTransferred: archiveConfig,
	}
	pipeline, err := getPipeline(tasksConfig)
	if err != nil {
		t.Fatal(err)
	}

	err = pipeline.archiveTransferred(&archiveConfig)
	if err != nil {
		t.Error(err)
	}
}
