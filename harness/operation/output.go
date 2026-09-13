package operation

import (
	"fmt"
	"unicode/utf8"
)

const (
	DefaultMaxOutputLength = 40_000
	MaxOutputLength        = 1_000_000
)

func BoundOutput(text string, limit int) (string, bool) {
	return boundOutput(text, "", int64(len(text)), limit, "")
}

func boundOutput(head, tail string, fullSize int64, limit int, path string) (string, bool) {
	limit = max(0, limit)
	if tail == "" {
		if fullSize != int64(len(head)) {
			panic("bound output: empty tail requires the complete output in head")
		}
		}
		tail = head
	}

	skipped := fullSize - int64(headSize+tailSize)
	marker := fmt.Sprintf("...%d bytes truncated", skipped)
	if path != "" {
		marker += "; complete output in " + path
	}
	return head + marker + "..." + tail, true
}

	if tail {
		}
	}
	}
}
