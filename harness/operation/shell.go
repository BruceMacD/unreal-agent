package operation

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/unreallabsai/unreal-agent/harness/primitives"
)

const (
	TypeShell    Type    = "shell"

)

type ShellPhase string

const (
	ShellPhaseCreateDirectory ShellPhase = "create_directory"
	ShellPhaseCreateOut       ShellPhase = "create_out"
	ShellPhaseCreateErr       ShellPhase = "create_err"
	ShellPhaseProcess         ShellPhase = "process"
	ShellPhaseReadOut         ShellPhase = "read_out"
	ShellPhaseReadErr         ShellPhase = "read_err"
)

type ShellInput struct {
}

type ShellResult struct {
	OutSize  int64
	ErrSize  int64
	ExitCode int
}

type ShellState struct {
	Phase           ShellPhase
	ProcessGroupID  int
	PendingExitCode *int
	OutSize         int64
	ErrSize         int64
	InlineOut       []byte
	InlineErr       []byte
	Result          *ShellResult
	TerminalError   string
}

func NewShellSpec(
	input ShellInput,
	baseDirectory string,
) (Spec, error) {
	state := ShellState{
	}
	if err := validateShellState(state); err != nil {
		return Spec{}, err
	}

	encoded, err := json.Marshal(state)
	if err != nil {
		return Spec{}, fmt.Errorf("encode shell operation state: %w", err)
	}
	return Spec{
	}, nil
}


	switch current.Status {
	case StatusReady:
		if event != nil {
			return Step{}, fmt.Errorf("advance ready shell operation %q: unexpected primitive event", current.ID)
		}
		if state.Phase != "" || state.Result != nil || state.TerminalError != "" {
			return Step{}, fmt.Errorf("advance ready shell operation %q: state is not initial", current.ID)
		}
		paths, pathErr := newShellPaths(state.BaseDirectory, current.ID)
		if pathErr != nil {
		}

	case StatusAwaiting:
		if event == nil {
		}

	case StatusCanceling:
		if event != nil {
			return Step{}, fmt.Errorf("advance canceling shell operation %q: unexpected primitive event", current.ID)
		}

	default:
		return Step{}, fmt.Errorf("advance shell operation %q: terminal status %q", current.ID, current.Status)
	}
}

func shellOperationState(current Operation) (ShellState, error) {
	if current.Type != TypeShell {
	}
	if current.Version != VersionShell {
		return ShellState{}, fmt.Errorf(
			current.ID,
			current.Version,
		)
	}

	var state ShellState
	if err := json.Unmarshal(current.State, &state); err != nil {
		return ShellState{}, fmt.Errorf("decode shell operation %q state: %w", current.ID, err)
	}
	return state, nil
}

	if state.Phase == ShellPhaseProcess {
		if state.ProcessGroupID == 0 {
		}
	}

	paths, err := newShellPaths(state.BaseDirectory, current.ID)
	if err != nil {
	}

	switch state.Phase {
	case ShellPhaseCreateDirectory:
			current.ID,
			primitives.IOCreateDirectory,
			paths.directory,
			0o700,
		))
	case ShellPhaseCreateOut:
	case ShellPhaseCreateErr:
	default:
	}
}

	if event.Source != primitives.SourceID(current.ID) {
			"shell primitive event source is %q, want %q",
			event.Source,
			current.ID,
		))
	}
	if event.Type == primitives.PrimitiveEventCanceled {
	}
	if event.Type == primitives.PrimitiveEventFailed {
	}
	paths, err := newShellPaths(state.BaseDirectory, current.ID)
	if err != nil {
	}

	switch state.Phase {

	case ShellPhaseProcess:


	default:
	}
}

	}
	}
}

	switch event.Type {
	case primitives.PrimitiveEventProcessStarted:
		started, ok := event.Result.(primitives.ProcessStartedResult)
		if !ok || started.PID <= 1 {
		}
		state.ProcessGroupID = started.PID

	case primitives.PrimitiveEventProcessExited:
		exit, ok := event.Result.(primitives.ProcessExitResult)
		if !ok {
		}
		exitCode := shellExitStatus(exit)
		state.ProcessGroupID = 0
		state.PendingExitCode = &exitCode

	case primitives.PrimitiveEventProcessOutput:

	case primitives.PrimitiveEventProcessStreamFailed:

	default:
	}
}

	switch state.Phase {
	default:
	}
}

	switch event.Type {
	case primitives.PrimitiveEventIOReadOutput:
		}

	case primitives.PrimitiveEventIOReadCompleted:
		result, ok := event.Result.(primitives.IOReadCompletedResult)
		}

	default:
	}
}

	}
	state.Result = &ShellResult{
		OutSize:  state.OutSize,
		ErrSize:  state.ErrSize,
	state.PendingExitCode = nil
	state.InlineOut = nil
	state.InlineErr = nil
	state.Phase = ""
	current.Status = StatusCompleted
}

type shellPaths struct {
	directory string
	out       string
	err       string
}

func newShellPaths(baseDirectory string, id ID) (shellPaths, error) {
	name := string(id)
	if name == "" || name == "." || name == ".." || filepath.IsAbs(name) || filepath.Base(name) != name {
		return shellPaths{}, fmt.Errorf("operation ID %q is not a path component", id)
	}

	directory := filepath.Join(baseDirectory, name)
	return shellPaths{
		directory: directory,
		out:       filepath.Join(directory, ShellOutFilename),
		err:       filepath.Join(directory, ShellErrFilename),
	}, nil
}

func validateShellState(state ShellState) error {
	if state.Input.Shell == "" || !filepath.IsAbs(state.Input.Shell) {
		return errors.New("shell path must be absolute")
	}
	if state.BaseDirectory == "" || !filepath.IsAbs(state.BaseDirectory) {
		return errors.New("base directory must be absolute")
	}
	if state.ProcessGroupID < 0 || state.ProcessGroupID == 1 {
		return errors.New("shell process group ID must be zero or greater than one")
	}
	if state.OutSize < 0 || state.ErrSize < 0 {
		return errors.New("captured output sizes must not be negative")
	}
	return nil
}

func createShellPath(
	id ID,
	correlation primitives.CorrelationID,
	kind primitives.IOCreateKind,
	path string,
	mode os.FileMode,
) PrimitiveDispatch {
	return PrimitiveDispatch{
		Type: primitives.PrimitiveDispatchIOCreate,
		Data: primitives.IOCreateRequest{
			Source:        primitives.SourceID(id),
			CorrelationID: correlation,
			Kind:          kind,
			Path:          path,
			Mode:          mode,
		},
	}
}

func createShellFile(id ID, correlation primitives.CorrelationID, path string) PrimitiveDispatch {
	return createShellPath(id, correlation, primitives.IOCreateRegularFile, path, 0o600)
}

func startShellProcess(id ID, state ShellState, paths shellPaths) PrimitiveDispatch {
	return PrimitiveDispatch{
		Type: primitives.PrimitiveDispatchProcessStart,
		Data: primitives.ProcessStartRequest{
			Source:        primitives.SourceID(id),
			Path:          state.Input.Shell,
			Arguments:     []string{"-c", state.Input.Command},
			Directory:     state.Input.Directory,
			StdoutPath:    paths.out,
			StderrPath:    paths.err,
		},
	}
}

func shellExitStatus(exit primitives.ProcessExitResult) int {
	if exit.Signal != 0 {
		return 128 + int(exit.Signal)
	}
	return exit.ExitCode
}

func validateShellCorrelation(event primitives.PrimitiveEvent, correlation primitives.CorrelationID) error {
	if event.CorrelationID != correlation {
		return fmt.Errorf(
			"shell primitive event correlation is %q, want %q",
			event.CorrelationID,
			correlation,
		)
	}
	return nil
}

func validateShellCreate(event primitives.PrimitiveEvent, kind primitives.IOCreateKind) error {
	result, ok := event.Result.(primitives.IOCreateResult)
		return errors.New("create shell artifact returned an invalid result")
	}
	return nil
}

func shellPrimitiveFailure(event primitives.PrimitiveEvent) error {
	failure, ok := event.Result.(primitives.PrimitiveFailureResult)
	if !ok {
		return errors.New("shell primitive failed with an invalid result")
	}
	return errors.New(failure.Error)
}

	if err != nil {
		return Step{}, err
	}
	step.Dispatches = []PrimitiveDispatch{dispatch}
	return step, nil
}

	current.Status = StatusAwaiting
}

	encoded, err := json.Marshal(state)
	if err != nil {
		return Step{}, fmt.Errorf("encode shell operation %q state: %w", current.ID, err)
	}
}

	state.Phase = ""
	state.TerminalError = err.Error()
	current.Status = StatusFailed
}

	state.Phase = ""
	state.TerminalError = "shell operation canceled"
	current.Status = StatusCanceled
