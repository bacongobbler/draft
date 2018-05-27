package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

const Gitkeepfile = ".gitkeep"

func TestEnsureDirectories(t *testing.T) {
	resetEnvVars := unsetEnvVars()
	tempHome, teardown := tempDir(t, "draft-init")
	defer func() {
		teardown()
		resetEnvVars()
	}()

	cmd := &initCmd{
		home: draftpath.Home(tempHome),
		out:  ioutil.Discard,
	}

	if err := cmd.ensureDirectories(); err != nil {
		t.Fatal(err)
	}

	fi, err := os.Stat(cmd.home.String())
	if err != nil {
		t.Errorf("Expected home directory but got err: %v", err)
	}

	if !fi.IsDir() {
		t.Error("Expected home to be directory but isn't")
	}

	fi, err = os.Stat(cmd.home.Plugins())
	if err != nil {
		t.Errorf("Expected plugins directory but got err: %v", err)
	}

	if !fi.IsDir() {
		t.Error("Expected plugins to be directory but isn't")
	}

	fi, err = os.Stat(cmd.home.Packs())
	if err != nil {
		t.Errorf("Expected packs directory but got err: %v", err)
	}

	if !fi.IsDir() {
		t.Error("Expected packs to be directory but isn't")
	}

}

// tempDir create and clean a temporary directory to work in our tests
func tempDir(t *testing.T, description string) (string, func()) {
	t.Helper()
	path, err := ioutil.TempDir("", description)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	return path, func() {
		if err := os.RemoveAll(path); err != nil {
			t.Fatalf("err: %s", err)
		}
	}
}

// add .gitkeep to generated empty directories
func addGitKeep(t *testing.T, p string) {
	t.Helper()
	if err := filepath.Walk(p, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		files, err := ioutil.ReadDir(p)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			f, err := os.OpenFile(filepath.Join(p, Gitkeepfile), os.O_RDONLY|os.O_CREATE, 0666)
			if err != nil {
				return err
			}
			defer f.Close()
		}
		return nil
	}); err != nil {
		t.Fatalf("couldn't stamp git keep files: %v", err)
	}
}

// Compares two strings and asserts equivalence.
func assertEqualString(t *testing.T, is string, shouldBe string) {
	t.Helper()
	if is == shouldBe {
		return
	}

	t.Fatalf("Assertion failed: Expected: %s. Got: %s", shouldBe, is)
}

// assertIdentical compares recursively all original and generated file content
func assertIdentical(t *testing.T, original, generated string) {
	t.Helper()
	if err := filepath.Walk(original, func(f string, fi os.FileInfo, err error) error {
		relp := strings.TrimPrefix(f, original)
		// root path
		if relp == "" {
			return nil
		}
		relp = relp[1:]
		p := filepath.Join(generated, relp)

		// .keep files are only for keeping directory creations in remote git repo
		if filepath.Base(p) == Gitkeepfile {
			return nil
		}

		fo, err := os.Stat(p)
		if err != nil {
			t.Fatalf("%s doesn't exist while %s does", p, f)
		}

		if fi.IsDir() {
			if !fo.IsDir() {
				t.Fatalf("%s is a directory and %s isn't", f, p)
			}
			// else, it's a directory as well and we are done.
			return nil
		}

		wanted, err := ioutil.ReadFile(f)
		if err != nil {
			t.Fatalf("Couldn't read %s: %v", f, err)
		}
		actual, err := ioutil.ReadFile(p)
		if err != nil {
			t.Fatalf("Couldn't read %s: %v", p, err)
		}
		if !bytes.Equal(actual, wanted) {
			t.Errorf("%s and %s content differs:\nACTUAL:\n%s\n\nWANTED:\n%s", p, f, actual, wanted)
		}
		return nil
	}); err != nil {
		t.Fatalf("err: %s", err)
	}

	// on the other side, check that all generated items are in origin
	if err := filepath.Walk(generated, func(f string, _ os.FileInfo, err error) error {
		relp := strings.TrimPrefix(f, generated)
		// root path
		if relp == "" {
			return nil
		}
		relp = relp[1:]
		p := filepath.Join(original, relp)

		// .keep files are only for keeping directory creations in remote git repo
		if filepath.Base(p) == Gitkeepfile {
			return nil
		}

		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s doesn't exist while %s does", p, f)
		}
		return nil
	}); err != nil {
		t.Fatalf("err: %s", err)
	}
}
