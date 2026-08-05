package primitives

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const (
	PrimitiveEventProcessStarted      PrimitiveEventType = "process.started"
	PrimitiveEventProcessOutput       PrimitiveEventType = "process.output"
	PrimitiveEventProcessStreamFailed PrimitiveEventType = "process.stream_failed"
	PrimitiveEventProcessExited       PrimitiveEventType = "process.exited"

	PrimitiveEventProcessInputWritten     PrimitiveEventType = "process.input_written"
	PrimitiveEventProcessInputWriteFailed PrimitiveEventType = "process.input_write_failed"
	PrimitiveEventProcessInputClosed      PrimitiveEventType = "process.input_closed"
	PrimitiveEventProcessSignaled         PrimitiveEventType = "process.signaled"

	ProcessOutputChunkSize = 32 * 1024

	processTerminationGracePeriod = 5 * time.Second
)

var errProcessDone = errors.New("process invocation is done")

type ProcessStartRequest struct {
	Source        SourceID
	CorrelationID CorrelationID
	Path        string
	Arguments   []string
	Directory   string
	Environment []string
}

type ProcessStartedResult struct {
	PID int
}

type ProcessStream int

const (
	ProcessStdout ProcessStream = 1
	ProcessStderr ProcessStream = 2
)

type ProcessOutputResult struct {
	Stream ProcessStream
	Offset int64
	Data   []byte
}

type ProcessStreamFailureResult struct {
	Stream ProcessStream
	Error  string
}

type ProcessExitResult struct {
	ExitCode int
	Signal   syscall.Signal
}

type ProcessWriteRequest struct {
	Source        SourceID
	CorrelationID CorrelationID
	Data          []byte
}

type ProcessWriteResult struct {
	Count int
}

type ProcessWriteFailureResult struct {
	Count int
	Error string
}

type ProcessCloseInputRequest struct {
	Source        SourceID
	CorrelationID CorrelationID
}

type ProcessSignalRequest struct {
	Source              SourceID
	CorrelationID       CorrelationID
	Signal              syscall.Signal
	PropagateToChildren bool
}

type ProcessInvocation struct {
	ctx      context.Context
	ready    chan struct{}
	process  *os.Process
	stdin    *os.File
	startErr error
}

func StartProcess(
	ctx context.Context,
	request ProcessStartRequest,
) *ProcessInvocation {
	invocation := &ProcessInvocation{
	}
	go runProcess(ctx, request, invocation, events)
	return invocation
}

// WriteInput honors cancellation until the pipe write starts. Once started, the write owns
// its data until it completes or the process/input lifecycle interrupts the pipe. Concurrent
// writes may interleave; callers that require ordering dispatch the next write after its event.
func (process *ProcessInvocation) WriteInput(
	ctx context.Context,
	request ProcessWriteRequest,
	go process.writeInput(ctx, request, events)
}

func (process *ProcessInvocation) writeInput(
	ctx context.Context,
	request ProcessWriteRequest,
	events chan<- PrimitiveEvent,
) {
	if err := process.prepareControl(ctx, false); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			events <- processCanceled(request.Source, request.CorrelationID)
		} else {
			events <- processInputWriteFailure(request, 0, err)
		}
		return
	}
	count, err := process.stdin.Write(request.Data)
	if err == nil {
		events <- PrimitiveEvent{
			Type:          PrimitiveEventProcessInputWritten,
			Source:        request.Source,
			CorrelationID: request.CorrelationID,
			Result:        ProcessWriteResult{Count: count},
		}
		return
	}
	events <- processInputWriteFailure(request, count, err)
}

func processInputWriteFailure(
	request ProcessWriteRequest,
	count int,
	err error,
) PrimitiveEvent {
	return PrimitiveEvent{
		Type:          PrimitiveEventProcessInputWriteFailed,
		Source:        request.Source,
		CorrelationID: request.CorrelationID,
		Result: ProcessWriteFailureResult{
			Count: count,
			Error: fmt.Errorf("write process input: %w", err).Error(),
		},
	}
}

// CloseInput is idempotent and interrupts any active WriteInput invocation.
func (process *ProcessInvocation) CloseInput(
	ctx context.Context,
	request ProcessCloseInputRequest,
	go process.closeInput(ctx, request, events)
}

func (process *ProcessInvocation) closeInput(
	ctx context.Context,
	request ProcessCloseInputRequest,
	events chan<- PrimitiveEvent,
) {
	if err := process.prepareControl(ctx, false); err != nil {
		events <- processControlFailure(request.Source, request.CorrelationID, "close process input", err)
		return
	}
	if err := normalizeProcessCloseError(process.stdin.Close()); err != nil {
		events <- processFailure(
			request.Source,
			request.CorrelationID,
			fmt.Errorf("close process input: %w", err),
		)
		return
	}
		Type:          PrimitiveEventProcessInputClosed,
		Source:        request.Source,
		CorrelationID: request.CorrelationID,
	}
}

func (process *ProcessInvocation) Signal(
	ctx context.Context,
	request ProcessSignalRequest,
	go process.signal(ctx, request, events)
}

func (process *ProcessInvocation) signal(
	ctx context.Context,
	request ProcessSignalRequest,
	events chan<- PrimitiveEvent,
) {
	if err := process.prepareControl(ctx, request.PropagateToChildren); err != nil {
		events <- processControlFailure(request.Source, request.CorrelationID, "signal process", err)
		return
	}
	var err error
	if request.PropagateToChildren {
		err = syscall.Kill(-process.process.Pid, request.Signal)
	} else {
		err = process.process.Signal(request.Signal)
	}
	if err != nil {
		events <- processFailure(
			request.Source,
			request.CorrelationID,
			fmt.Errorf("signal process: %w", err),
		)
		return
	}
	events <- PrimitiveEvent{
		Type:          PrimitiveEventProcessSignaled,
		Source:        request.Source,
		CorrelationID: request.CorrelationID,
	}
}

func (process *ProcessInvocation) prepareControl(
	ctx context.Context,
	processGroup bool,
) error {
	select {
	case <-process.ready:
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if process.startErr != nil {
		// A canceled start does not cancel this independently owned control invocation.
		return fmt.Errorf("%w: %v", errProcessDone, process.startErr)
	}
	if process.ctx.Err() != nil {
		return errProcessDone
	}
	if processGroup {
		exists, err := processGroupExists(process.process.Pid)
		if err != nil {
			return fmt.Errorf("inspect process group: %w", err)
		}
		if !exists {
			return errProcessDone
		}
		return nil
	}
	if processWaitCompleted(process.process) {
		return errProcessDone
	}
	return nil
}

func processControlFailure(
	source SourceID,
	correlationID CorrelationID,
	action string,
	err error,
) PrimitiveEvent {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return processCanceled(source, correlationID)
	}
	return processFailure(source, correlationID, fmt.Errorf("%s: %w", action, err))
}

func runProcess(
	ctx context.Context,
	request ProcessStartRequest,
	process *ProcessInvocation,
) {
	if err := validateProcessStartRequest(request); err != nil {
		process.startErr = err
		close(process.ready)
		return
	}
	if ctx.Err() != nil {
		process.startErr = ctx.Err()
		close(process.ready)
		return
	}

	command, pipes, err := prepareProcess(request)
	if err != nil {
		process.startErr = err
		close(process.ready)
		return
	}
	if ctx.Err() != nil {
		cleanupErr := pipes.closeAll()
		process.startErr = ctx.Err()
		close(process.ready)
		sendProcessTerminalEvent(
			events,
			processCompletionEvent(
				request,
				processCanceled(request.Source, request.CorrelationID),
				cleanupErr,
			),
		)
		return
	}
	if err := command.Start(); err != nil {
		startErr := errors.Join(
			fmt.Errorf("start process %q: %w", request.Path, err),
			pipes.closeAll(),
		)
		process.startErr = startErr
		close(process.ready)
		return
	}

	parentPipes := pipes.parent()
	process.process = command.Process
	process.stdin = parentPipes.stdin
	close(process.ready)
	childPipeCloseErr := pipes.closeChildEnds()

	sendProcessEvent(ctx, events, PrimitiveEvent{
		Type:          PrimitiveEventProcessStarted,
		Source:        request.Source,
		CorrelationID: request.CorrelationID,
		Result:        ProcessStartedResult{PID: command.Process.Pid},
	})

	waitCompleted := make(chan error, 1)
	go func() {
		waitCompleted <- command.Wait()
	}()
	var workers sync.WaitGroup

		ctx,
		command.Process,
		parentPipes,
		waitCompleted,
	)
	outputFinishErr := error(nil)
	if !canceled {
		outputFinishErr = parentPipes.finishOutput()
	}
	workers.Wait()
	outputCloseErr := parentPipes.closeOutput()

	if canceled {
		_, waitCompletionErr := processResult(command, waitErr)
		shutdownErr := errors.Join(
			childPipeCloseErr,
			stdinCloseErr,
			outputFinishErr,
			outputCloseErr,
			waitCompletionErr,
		)
		sendProcessTerminalEvent(
			events,
			processCompletionEvent(
				request,
				processCanceled(request.Source, request.CorrelationID),
				shutdownErr,
			),
		)
		return
	}

	exitResult, exitErr := processResult(command, waitErr)
	terminalErr := errors.Join(
		childPipeCloseErr,
		stdinCloseErr,
		outputFinishErr,
		outputCloseErr,
		exitErr,
	)
	sendProcessTerminalEvent(
		events,
		processCompletionEvent(
			request,
			PrimitiveEvent{
				Type:          PrimitiveEventProcessExited,
				Source:        request.Source,
				CorrelationID: request.CorrelationID,
				Result:        exitResult,
			},
			terminalErr,
		),
	)
}

func awaitProcessCompletion(
	ctx context.Context,
	process *os.Process,
	pipes processParentPipes,
	waitCompleted <-chan error,
) (error, bool, error) {
	select {
	case waitErr := <-waitCompleted:
	case <-ctx.Done():
		if processWaitCompleted(process) {
		}
		cancellationErr := terminateProcess(
			process,
			pipes,
		)
		return <-waitCompleted, true, cancellationErr
	}
}

func processWaitCompleted(process *os.Process) bool {
	if process == nil {
		return false
	}
	// os.Process marks itself done at the OS wait boundary, before exec.Cmd.Wait returns.
	return errors.Is(process.Signal(syscall.Signal(0)), os.ErrProcessDone)
}

type processPipes struct {
}

type processParentPipes struct {
}

func prepareProcess(request ProcessStartRequest) (*exec.Cmd, processPipes, error) {
	var pipes processPipes
	}

	}

	}

	command := exec.Command(request.Path, request.Arguments...)
	command.Dir = request.Directory
	command.Env = request.Environment
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return command, pipes, nil
}

func (pipes processPipes) parent() processParentPipes {
	return processParentPipes{
	}
}

func (pipes processPipes) closeChildEnds() error {
	return errors.Join(
	)
}

func (pipes processPipes) closeAll() error {
	return errors.Join(
		closeProcessFile(pipes.stdinRead),
		closeProcessFile(pipes.stdinWrite),
		closeProcessFile(pipes.stdoutRead),
		closeProcessFile(pipes.stdoutWrite),
		closeProcessFile(pipes.stderrRead),
		closeProcessFile(pipes.stderrWrite),
	)
}

func (pipes processParentPipes) closeAll() error {
	return errors.Join(
		closeProcessFile(pipes.stdin),
		closeProcessFile(pipes.stdout),
		closeProcessFile(pipes.stderr),
	)
}

func (pipes processParentPipes) finishOutput() error {
	// Wake reads held open by descendants; each reader snapshots and drains the bytes already buffered.
	deadline := time.Now()
	return errors.Join(
		finishProcessOutput(pipes.stdout, deadline),
		finishProcessOutput(pipes.stderr, deadline),
	)
}

func finishProcessOutput(file *os.File, deadline time.Time) error {
	deadlineErr := file.SetReadDeadline(deadline)
	if deadlineErr == nil {
		return nil
	}
	return errors.Join(deadlineErr, closeProcessFile(file))
}

func (pipes processParentPipes) closeOutput() error {
	return errors.Join(closeProcessFile(pipes.stdout), closeProcessFile(pipes.stderr))
}

func terminateProcess(
	process *os.Process,
	pipes processParentPipes,
	gracePeriod time.Duration,
) error {
	termErr := signalProcessInvocation(process, syscall.SIGTERM)
	exited, waitErr := waitForProcessInvocation(process, gracePeriod)
	var killErr error
	if !exited {
		killErr = signalProcessInvocation(process, syscall.SIGKILL)
	}
	return errors.Join(termErr, waitErr, killErr, pipes.closeAll())
}

func signalProcessInvocation(process *os.Process, signal syscall.Signal) error {
	err := syscall.Kill(-process.Pid, signal)
	if err == nil {
		return nil
	}
	if !errors.Is(err, syscall.ESRCH) {
		return err
	}

	err = process.Signal(signal)
	if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

func waitForProcessInvocation(
	process *os.Process,
	gracePeriod time.Duration,
) (bool, error) {
	deadline := time.Now().Add(gracePeriod)
	delay := time.Millisecond
	for {
		groupExists, err := processGroupExists(process.Pid)
		if err != nil {
			return false, err
		}
		if !groupExists && processWaitCompleted(process) {
			return true, nil
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return false, nil
		}
		timer := time.NewTimer(min(delay, remaining))
		<-timer.C
		delay = min(delay*2, 50*time.Millisecond)
	}
}

func processGroupExists(processGroupID int) (bool, error) {
	err := syscall.Kill(-processGroupID, 0)
	if err == nil || errors.Is(err, syscall.EPERM) {
		return true, nil
	}
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}
	return false, err
}

func closeProcessFile(file *os.File) error {
	if file == nil {
		return nil
	}
	return normalizeProcessCloseError(file.Close())
}

func normalizeProcessCloseError(err error) error {
	if errors.Is(err, os.ErrClosed) {
		return nil
	}
	return err
}

type processOutputState struct {
	stream ProcessStream
	reader *os.File
}

func runProcessOutput(
	ctx context.Context,
	request ProcessStartRequest,
	output *processOutputState,
	events chan<- PrimitiveEvent,
) {
	err := streamProcessOutput(ctx, request, output, events)
	if err != nil {
		sendProcessEvent(ctx, events, processStreamFailure(request, output.stream, err))
	}
}

func streamProcessOutput(
	ctx context.Context,
	request ProcessStartRequest,
	output *processOutputState,
	events chan<- PrimitiveEvent,
) error {
	buffer := make([]byte, ProcessOutputChunkSize)
	var offset int64
	for {
		count, readErr := output.reader.Read(buffer)
		if count > 0 {
			sendProcessOutput(ctx, request, output.stream, offset, buffer[:count], events)
			offset += int64(count)
		}

		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if os.IsTimeout(readErr) {
			return drainProcessOutput(
				ctx,
				request,
				output.stream,
				output.reader,
				&offset,
				events,
			)
		}
		if readErr != nil {
			if ctx.Err() != nil && errors.Is(readErr, os.ErrClosed) {
				return nil
			}
			return fmt.Errorf("read process %s: %w", processStreamName(output.stream), readErr)
		}
	}
}

func drainProcessOutput(
	ctx context.Context,
	request ProcessStartRequest,
	stream ProcessStream,
	reader *os.File,
	offset *int64,
	events chan<- PrimitiveEvent,
) error {
	fileDescriptor := int(reader.Fd())
	if err := unix.SetNonblock(fileDescriptor, true); err != nil {
		return fmt.Errorf("drain process %s: set nonblocking: %w", processStreamName(stream), err)
	}

	buffer := make([]byte, ProcessOutputChunkSize)
	for {
		count, readErr := unix.Read(fileDescriptor, buffer)
		if count > 0 {
			if !sendProcessOutput(ctx, request, stream, *offset, buffer[:count], events) {
				return nil
			}
			*offset += int64(count)
		}
		if errors.Is(readErr, unix.EINTR) {
			continue
		}
		if errors.Is(readErr, unix.EAGAIN) || errors.Is(readErr, unix.EWOULDBLOCK) {
			return nil
		}
		if readErr != nil {
			return fmt.Errorf("drain process %s: %w", processStreamName(stream), readErr)
		}
		if count == 0 {
			return nil
		}
	}
}

func sendProcessOutput(
	ctx context.Context,
	request ProcessStartRequest,
	stream ProcessStream,
	offset int64,
	data []byte,
	events chan<- PrimitiveEvent,
) bool {
	return sendProcessEvent(ctx, events, PrimitiveEvent{
		Type:          PrimitiveEventProcessOutput,
		Source:        request.Source,
		CorrelationID: request.CorrelationID,
		Result: ProcessOutputResult{
			Stream: stream,
			Offset: offset,
			Data:   append([]byte(nil), data...),
		},
	})
}

func processResult(command *exec.Cmd, waitErr error) (ProcessExitResult, error) {
	if command.ProcessState == nil {
		return ProcessExitResult{}, fmt.Errorf("wait for process %q: %w", command.Path, waitErr)
	}
	var exitError *exec.ExitError
	if waitErr != nil && !errors.As(waitErr, &exitError) {
		return ProcessExitResult{}, fmt.Errorf("wait for process %q: %w", command.Path, waitErr)
	}

	result := ProcessExitResult{ExitCode: command.ProcessState.ExitCode()}
	if status, ok := command.ProcessState.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		result.Signal = status.Signal()
	}
	return result, nil
}

func processStreamName(stream ProcessStream) string {
	if stream == ProcessStderr {
		return "stderr"
	}
	return "stdout"
}

func validateProcessStartRequest(request ProcessStartRequest) error {
	if request.Path == "" {
		return errors.New("start process: path must be set")
	}
	if !filepath.IsAbs(request.Path) {
		return errors.New("start process: path must be absolute")
	}
	return nil
}

func processStreamFailure(
	request ProcessStartRequest,
	stream ProcessStream,
	err error,
) PrimitiveEvent {
	return PrimitiveEvent{
		Type:          PrimitiveEventProcessStreamFailed,
		Source:        request.Source,
		CorrelationID: request.CorrelationID,
		Result: ProcessStreamFailureResult{
			Stream: stream,
			Error:  err.Error(),
		},
	}
}

func sendProcessEvent(
	ctx context.Context,
	events chan<- PrimitiveEvent,
	event PrimitiveEvent,
) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case events <- event:
		return true
	case <-ctx.Done():
		return false
	}
}

}

func processCompletionEvent(
	request ProcessStartRequest,
	event PrimitiveEvent,
	err error,
) PrimitiveEvent {
	if err != nil {
		return processFailure(request.Source, request.CorrelationID, err)
	}
	return event
}

func processFailure(
	source SourceID,
	correlationID CorrelationID,
	err error,
) PrimitiveEvent {
	return primitiveFailure(source, correlationID, err)
}

func processCanceled(source SourceID, correlationID CorrelationID) PrimitiveEvent {
	return primitiveCanceled(source, correlationID)
}
