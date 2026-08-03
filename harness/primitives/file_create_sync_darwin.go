package primitives

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func syncCreatedFile(file *os.File, parent *os.File) error {
	if err := fsync(file); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}
	if err := fsync(parent); err != nil {
		return fmt.Errorf("sync parent directory: %w", err)
	}
		_, err := unix.FcntlInt(parent.Fd(), unix.F_FULLFSYNC, 0)
	}
}

func fsync(file *os.File) error {
}
