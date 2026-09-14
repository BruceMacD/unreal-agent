package primitives

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestInspectAndStreamFileReportsStatFailure(t *testing.T) {
	statErr := errors.New("stat failed")
	closeErr := errors.New("close failed")
	events := make(chan PrimitiveEvent, 1)
	inspectAndStreamFile(
		t.Context(),
		IOReadRequest{Path: "input"},
		events,
	)
	close(events)

	failure := singleInternalEvent(t, collectInternalEvents(events), PrimitiveEventFailed)
	result := failure.Result.(PrimitiveFailureResult)
	if !strings.Contains(result.Error, statErr.Error()) || !strings.Contains(result.Error, closeErr.Error()) {
		t.Fatalf("error = %q, want stat and close failures", result.Error)
	}
}

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
		{name: "closed", eventType: PrimitiveEventCanceled},
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

	singleInternalEvent(t, collectInternalEvents(events), PrimitiveEventCanceled)
}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	singleInternalEvent(t, collectInternalEvents(events), PrimitiveEventCanceled)
}

func TestStreamOpenFileCancellationBeforeRead(t *testing.T) {

}

type readFailureFile struct {
	err error
}

type statFailureReadFile struct {
	statErr  error
	closeErr error
}

func (file *statFailureReadFile) Close() error {
}

func (file *statFailureReadFile) Stat() (os.FileInfo, error) {
	return nil, file.statErr
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
	for {
		event := <-events
		collected = append(collected, event)
		if internalPrimitiveEventIsTerminal(event.Type) {
			return collected
		}
	}
}

func internalPrimitiveEventIsTerminal(eventType PrimitiveEventType) bool {
	switch eventType {
	case PrimitiveEventFailed,
		PrimitiveEventCanceled,
		PrimitiveEventIOCreateCompleted,
		PrimitiveEventIOReadCompleted,
		PrimitiveEventProcessExited,
		PrimitiveEventProcessInputWritten,
		PrimitiveEventProcessInputWriteFailed,
		PrimitiveEventProcessInputClosed,
		PrimitiveEventProcessSignaled,
		PrimitiveEventRemoteCompleted,
		PrimitiveEventTimerFired:
		return true
	default:
		return false
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

func TestIsOnlyContextCancellation(t *testing.T) {
	otherErr := errors.New("cleanup failed")
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "canceled", err: context.Canceled, want: true},
		{name: "deadline", err: context.DeadlineExceeded, want: true},
		{name: "wrapped", err: fmt.Errorf("read: %w", context.Canceled), want: true},
		{name: "joined cancellations", err: errors.Join(context.Canceled, context.DeadlineExceeded), want: true},
		{name: "mixed join", err: errors.Join(context.Canceled, otherErr), want: false},
		{name: "other", err: otherErr, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isOnlyContextCancellation(test.err); got != test.want {
				t.Fatalf("isOnlyContextCancellation() = %t, want %t", got, test.want)
			}
		})
	}
}
