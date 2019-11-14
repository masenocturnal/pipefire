package directdebit

import (
	"io/ioutil"
	"testing"
)

func TestCleanup(t *testing.T) {

	// make a directory and fill it will stuff
	dir1Name, err := ioutil.TempDir("/tmp", "pipefire_test")
	if err != nil {
		t.Fatal(err)
	}

	dir2Name, err := ioutil.TempDir("/tmp", "pipefire_test")
	if err != nil {
		t.Fatal(err)
	}

	paths := []string{dir1Name, dir2Name}

	for _, path := range paths {
		for i := 0; i < 10; i++ {
			if _, err := ioutil.TempFile(path, "pipetest"); err != nil {
				t.Fatal(err)
			}
		}
	}

	cleanUpConfig := CleanUpConfig{
		Paths: paths,
	}

	tasksConfig := &TasksConfig{
		CleanDirtyFiles: cleanUpConfig,
	}
	pipeline, err := getPipeline(tasksConfig)
	if err != nil {
		t.Fatal(err)
	}

	errs := pipeline.cleanDirtyFiles(&cleanUpConfig)
	if len(errs) > 0 {
		t.Error(err)
	}
}
