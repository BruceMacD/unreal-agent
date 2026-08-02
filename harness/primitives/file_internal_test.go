package primitives

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestStreamOpenFileReportsReadFailure(t *testing.T) {
	readErr := errors.New("read failed")
		t.Context(),
	))

	failure := singleInternalEvent(t, events, PrimitiveEventFailed)
	result := failure.Result.(PrimitiveFailureResult)
	if !strings.Contains(result.Error, readErr.Error()) {
		t.Fatalf("error = %q, want %q", result.Error, readErr)
	}
}

func TestStreamOpenFileReportsCompletionCloseFailure(t *testing.T) {
	closeErr := errors.New("close failed")
		t.Context(),
		IOReadRequest{Path: "input"},
	))

	failure := singleInternalEvent(t, events, PrimitiveEventFailed)
	result := failure.Result.(PrimitiveFailureResult)
	if !strings.Contains(result.Error, closeErr.Error()) {
		t.Fatalf("error = %q, want %q", result.Error, closeErr)
	}
}

func TestStreamOpenFileCancellationWhileSendingOutput(t *testing.T) {
		name      string
		closeErr  error
		eventType PrimitiveEventType
	}{
		{name: "close failure", closeErr: errors.New("close failed"), eventType: PrimitiveEventFailed},
		t.Run(test.name, func(t *testing.T) {
			defer cancel()
				started:  make(chan struct{}),
				closeErr: test.closeErr,
			}

			cancel()

		})
	}
}

func TestStreamOpenFileCancellationAfterClose(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	file := &blockingCloseFile{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}

	<-file.started
	cancel()
	close(file.release)

}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

}

func TestStreamOpenFileCancellationBeforeRead(t *testing.T) {

}

type readFailureFile struct {
	err error
}

type closeFailureFile struct {
	err error
}

func (file *closeFailureFile) Close() error {
}

	started  chan struct{}
	closeErr error
}

	file.started <- struct{}{}
}

type cancelingFile struct {
	cancel context.CancelFunc
}

	file.cancel()
}

	ctx context.Context,
	request IOReadRequest,
) <-chan PrimitiveEvent {
	events := make(chan PrimitiveEvent)
	go func() {
		defer close(events)
	}()
	return events
}

func collectInternalEvents(events <-chan PrimitiveEvent) []PrimitiveEvent {
	var collected []PrimitiveEvent
		collected = append(collected, event)
	}
}

func singleInternalEvent(
	t *testing.T,
	events []PrimitiveEvent,
	eventType PrimitiveEventType,
) PrimitiveEvent {
	t.Helper()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Type != eventType {
		t.Fatalf("event type = %q, want %q", events[0].Type, eventType)
	}
	return events[0]
}
