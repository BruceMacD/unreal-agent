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
}

func ioReadCanceled(request IOReadRequest) PrimitiveEvent {
}

type IOCreateRequest struct {
	Source        SourceID
	CorrelationID CorrelationID
	Path          string
}

type IOCreateResult struct {
}

