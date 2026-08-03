//go:build unix

package primitives

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

	t.Chdir(t.TempDir())

		Path: "created",
		Mode: IOCreateDefaultMode,
	})
	if err != nil {
		t.Fatal(err)
	}
	}
	if _, err := os.Stat("created"); err != nil {
		t.Fatal(err)
	}
}

	mode := os.FileMode(0o600)
	path := filepath.Join(t.TempDir(), "created")

	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != mode {
		t.Fatalf("mode = %o, want %o", info.Mode().Perm(), mode)
	}
}

func TestUnixFileMode(t *testing.T) {
	mode := os.FileMode(0o640) | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
	got, err := unixFileMode(mode)
	if err != nil {
		t.Fatal(err)
	}
	want := uint32(0o640 | 0o4000 | 0o2000 | 0o1000)
	if got != want {
		t.Fatalf("mode = %#o, want %#o", got, want)
	}
}

	path := filepath.Join(t.TempDir(), "created")
	if err == nil || !strings.Contains(err.Error(), "unsupported mode bits") {
		t.Fatalf("error = %v, want unsupported mode failure", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("stat rejected path: %v, want not exist", statErr)
	}
}

func TestOpenCreateDirectoryRejectsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	directory, err := openCreateDirectory(path)
	if err == nil {
		if closeErr := directory.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal("opening a file as the parent directory succeeded")
	}
}

func TestOpenNewFileAtRejectsClosedDirectory(t *testing.T) {
	parent, err := openCreateDirectory(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := parent.Close(); err != nil {
		t.Fatal(err)
	}

	mode, err := unixFileMode(IOCreateDefaultMode)
	if err != nil {
		t.Fatal(err)
	}
	file, err := openNewFileAt(parent, "created", mode)
	if err == nil {
		if closeErr := file.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal("creating relative to a closed directory succeeded")
	}
}

func TestSyncCreatedFileReportsFileSyncFailure(t *testing.T) {
	parent, err := openCreateDirectory(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := parent.Close(); err != nil {
			t.Error(err)
		}
	})

	file, err := os.CreateTemp(t.TempDir(), "created-")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	err = syncCreatedFile(file, parent)
	if err == nil || !strings.Contains(err.Error(), "sync file") {
		t.Fatalf("error = %v, want file synchronization failure", err)
	}
}

func TestSyncCreatedFileReportsParentSyncFailure(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "created-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})

	parent, err := openCreateDirectory(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := parent.Close(); err != nil {
		t.Fatal(err)
	}

	err = syncCreatedFile(file, parent)
	if err == nil || !strings.Contains(err.Error(), "sync parent directory") {
		t.Fatalf("error = %v, want parent-directory synchronization failure", err)
	}
}

func TestPersistCreatedFileReportsSyncAndCloseFailures(t *testing.T) {
	parentPath := t.TempDir()
	parent, err := openCreateDirectory(parentPath)
	if err != nil {
		t.Fatal(err)
	}

	file, err := os.CreateTemp(parentPath, "created-")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if err == nil {
		t.Fatal("persisting a closed file succeeded")
	}
	for _, want := range []string{"sync file", "close file"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err, want)
		}
	}
}

func TestCloseFileReportsFailure(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "created-")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	err = closeFile("file", file.Name(), file)
	if err == nil || !errors.Is(err, os.ErrClosed) {
		t.Fatalf("error = %v, want closed-file error", err)
	}
}
