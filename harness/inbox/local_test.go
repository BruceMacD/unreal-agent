package inbox_test

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
)

	}
	}
}

	}
	}
}

		ID:      "same-input",
		Kind:    inbox.InputExternal,
		Payload: jsontext.Value(`{"message":"first"}`),
		ID:      "same-input",
	}
	}
	}
	}
}

	}
	}
}

	const count = 100
	errors := make(chan error, count)
	var group sync.WaitGroup
		group.Go(func() {
		})
	}
	group.Wait()
	close(errors)
	for err := range errors {
	}

	}
		}
	}
}


	for index := range 100 {
			t.Fatalf("submit %d: %v", index, err)
		}
	}
}

	}

	}
	}
}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	select {
		if open {
		}
	case <-time.After(time.Second):
	}
		t.Fatalf("submit error = %v, want context canceled", err)
	}
}

	tests := []inbox.Input{
	}
	for _, input := range tests {
			t.Fatalf("Submit(%#v) succeeded", input)
		}
	}
}

	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	}
}

	t.Helper()
	select {
		if !open {
		}
	case <-time.After(time.Second):
	}
}
