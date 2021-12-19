package trellis

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func LoadFixtureProject(t *testing.T) func() {
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	tempDir, err := ioutil.TempDir("", "trellis")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	os.Chdir("../trellis")
	cmd := exec.Command("cp", "-a", "testdata/trellis", tempDir)
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	os.Chdir(filepath.Join(tempDir, "trellis"))

	return func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("err: %s", err)
		}

		os.RemoveAll(tempDir)
	}
}

func TestChdir(t *testing.T, dir string) func() {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("err: %s", err)
	}
	return func() { os.Chdir(old) }
}
