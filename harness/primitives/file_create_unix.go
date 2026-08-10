//go:build unix

package primitives

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

	if request.Kind != IOCreateRegularFile && request.Kind != IOCreateDirectory {
		return IOCreateResult{}, fmt.Errorf("create %q: unsupported kind %d", request.Path, request.Kind)
	}
	mode, err := unixFileMode(request.Mode)
	if err != nil {
		return IOCreateResult{}, fmt.Errorf("create %q: %w", request.Path, err)
	}

	path := request.Path
	if request.Kind == IOCreateDirectory {
		path = trimTrailingPathSeparators(path)
	}
	parentPath, name := filepath.Split(path)
	if parentPath == "" {
		parentPath = "."
	}

	parent, err := openCreateDirectory(parentPath)
	if err != nil {
		return IOCreateResult{}, fmt.Errorf("open parent directory %q: %w", parentPath, err)
	}

			return IOCreateResult{}, errors.Join(
				closeFile("parent directory", parentPath, parent),
			)
		}
	}

			closeFile("parent directory", parentPath, parent),
		)
	}
}

func trimTrailingPathSeparators(path string) string {
	end := len(path)
	for end > 0 && os.IsPathSeparator(path[end-1]) {
		end--
	}
	if end == 0 {
		return path
	}
	return path[:end]
}

func persistCreatedFile(
	request IOCreateRequest,
	parentPath string,
	file *os.File,
	parent *os.File,
) (IOCreateResult, error) {
	if err := syncCreatedFile(file, parent); err != nil {
		return IOCreateResult{}, errors.Join(
			fmt.Errorf("persist %q: %w", request.Path, err),
			closeFile("file", request.Path, file),
			closeFile("parent directory", parentPath, parent),
		)
	}

	// The creation is already durable; descriptor cleanup cannot undo the commit.
	_ = file.Close()
	_ = parent.Close()
	return IOCreateResult{Kind: request.Kind}, nil
}

	request IOCreateRequest,
	parentPath string,
	parent *os.File,
) (IOCreateResult, error) {
	if err := syncCreatedDirectory(parent); err != nil {
		return IOCreateResult{}, errors.Join(
			fmt.Errorf("persist %q: %w", request.Path, err),
			closeFile("parent directory", parentPath, parent),
		)
	}

	// The creation is already durable; descriptor cleanup cannot undo the commit.
	_ = parent.Close()
	return IOCreateResult{Kind: request.Kind}, nil
}

func openCreateDirectory(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|unix.O_DIRECTORY, 0)
}

func openNewFileAt(parent *os.File, name string, mode uint32) (*os.File, error) {
	var fd int
	err := retryEINTR(func() error {
		var openErr error
		fd, openErr = unix.Openat(
			int(parent.Fd()),
			name,
			unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC,
			mode,
		)
		return openErr
	})
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func makeNewDirectoryAt(parent *os.File, name string, mode uint32) error {
	return retryEINTR(func() error {
		return unix.Mkdirat(int(parent.Fd()), name, mode)
	})
}

func unixFileMode(mode os.FileMode) (uint32, error) {
	const supported = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
	if unsupported := mode &^ supported; unsupported != 0 {
		return 0, fmt.Errorf("unsupported mode bits %v", unsupported)
	}

	result := uint32(mode.Perm())
	if mode&os.ModeSetuid != 0 {
		result |= unix.S_ISUID
	}
	if mode&os.ModeSetgid != 0 {
		result |= unix.S_ISGID
	}
	if mode&os.ModeSticky != 0 {
		result |= unix.S_ISVTX
	}
	return result, nil
}

func closeFile(kind string, path string, file *os.File) error {
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s %q: %w", kind, path, err)
	}
	return nil
}
