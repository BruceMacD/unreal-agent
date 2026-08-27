package primitives

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

const (
	PrimitiveEventIOCreateCompleted PrimitiveEventType = "io.create_completed"
	PrimitiveEventIOReadOutput      PrimitiveEventType = "io.read_output"
	PrimitiveEventIOReadCompleted   PrimitiveEventType = "io.read_completed"

	IOCreateDefaultMode os.FileMode = 0o664
	IOReadChunkSize                 = 32 * 1024
)

type IOCreateKind uint8

const (
	IOCreateRegularFile IOCreateKind = iota + 1
	IOCreateDirectory
)

type IOReadRequest struct {
	Source        SourceID
	CorrelationID CorrelationID
	Path          string
	Offset        int64
	Count         int64
}

type IOReadOutputResult struct {
	Offset int64
	Data   []byte
}

type IOReadCompletedResult struct {
}

	go streamFile(ctx, request, events)
}

func streamFile(ctx context.Context, request IOReadRequest, events chan<- PrimitiveEvent) {
	if err := validateIOReadRequest(request); err != nil {
		events <- ioReadFailure(request, err)
		return
	}
	if ctx.Err() != nil {
		events <- ioReadCanceled(request)
		return
	}

	file, err := os.OpenFile(request.Path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		events <- ioReadFailure(request, fmt.Errorf("open %q: %w", request.Path, err))
		return
	}
	inspectAndStreamFile(ctx, request, file, events)
}

type inspectedReadFile interface {
	Stat() (os.FileInfo, error)
}

func inspectAndStreamFile(
	ctx context.Context,
	request IOReadRequest,
	file inspectedReadFile,
	events chan<- PrimitiveEvent,
) {
	info, err := file.Stat()
	if err != nil {
		events <- ioReadFailure(request, errors.Join(
			fmt.Errorf("inspect %q: %w", request.Path, err),
			file.Close(),
		))
		return
	}
	if !info.Mode().IsRegular() {
		events <- ioReadFailure(request, errors.Join(
			fmt.Errorf("open %q: unsupported file type %s", request.Path, info.Mode().Type()),
			file.Close(),
		))
		return
	}
}

func streamOpenFile(
	ctx context.Context,
	request IOReadRequest,
	events chan<- PrimitiveEvent,
) {
		if ctx.Err() != nil {
			finishCanceledRead(events, request, file)
			return
		}
		if ctx.Err() != nil {
			finishCanceledRead(events, request, file)
			return
		}
		}
		}
		if readErr != nil {
			events <- ioReadFailure(request, errors.Join(
				fmt.Errorf("read %q: %w", request.Path, readErr),
				file.Close(),
			))
			return
		}
	}

	if err := file.Close(); err != nil {
		events <- ioReadFailure(request, fmt.Errorf("close %q: %w", request.Path, err))
		return
	}
	if ctx.Err() != nil {
		events <- ioReadCanceled(request)
		return
	}

	events <- PrimitiveEvent{
		Type:          PrimitiveEventIOReadCompleted,
		Source:        request.Source,
		CorrelationID: request.CorrelationID,
		Result: IOReadCompletedResult{
		},
	}
}

func finishCanceledRead(events chan<- PrimitiveEvent, request IOReadRequest, file io.Closer) {
	if err := file.Close(); err != nil {
		events <- ioReadFailure(request, fmt.Errorf("close %q: %w", request.Path, err))
		return
	}
	events <- ioReadCanceled(request)
}

func validateIOReadRequest(request IOReadRequest) error {
	if request.Offset < 0 {
		return fmt.Errorf("read %q: offset must not be negative", request.Path)
	}
	if request.Count < 0 {
		return fmt.Errorf("read %q: count must not be negative", request.Path)
	}
	return nil
}

func ioReadFailure(request IOReadRequest, err error) PrimitiveEvent {
	return primitiveFailure(request.Source, request.CorrelationID, err)
}

func ioReadCanceled(request IOReadRequest) PrimitiveEvent {
	return primitiveCanceled(request.Source, request.CorrelationID)
}

type IOCreateRequest struct {
	Source        SourceID
	CorrelationID CorrelationID
	Kind          IOCreateKind
	Path          string
	// Mode is used only when creating a new entry and is subject to umask.
	Mode os.FileMode
}

type IOCreateResult struct {
	Kind IOCreateKind
}

	go createPath(ctx, request, events)
}

func createPath(ctx context.Context, request IOCreateRequest, events chan<- PrimitiveEvent) {
	if ctx.Err() != nil {
		events <- ioCreateCanceled(request)
		return
	}

	result, err := createNewPath(ctx, request)
	if err != nil {
		if isOnlyContextCancellation(err) {
			events <- ioCreateCanceled(request)
			return
		}
		events <- ioCreateFailure(request, err)
		return
	}

	events <- PrimitiveEvent{
		Type:          PrimitiveEventIOCreateCompleted,
		Source:        request.Source,
		CorrelationID: request.CorrelationID,
		Result:        result,
	}
}

func ioCreateFailure(request IOCreateRequest, err error) PrimitiveEvent {
	return primitiveFailure(request.Source, request.CorrelationID, err)
}

func ioCreateCanceled(request IOCreateRequest) PrimitiveEvent {
	return primitiveCanceled(request.Source, request.CorrelationID)
}

func isOnlyContextCancellation(err error) bool {
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		causes := joined.Unwrap()
		for _, cause := range causes {
			if !isOnlyContextCancellation(cause) {
				return false
			}
		}
		return len(causes) > 0
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return isOnlyContextCancellation(wrapped.Unwrap())
	}
	return err == context.Canceled || err == context.DeadlineExceeded
}
