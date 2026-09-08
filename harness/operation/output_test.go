package operation_test

import (
	"encoding/json/v2"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/primitives"
)

func TestBoundOutputPreservesHeadAndTail(t *testing.T) {
	for _, test := range []struct {
		name, text, want string
		limit            int
		truncated        bool
	}{
		{"empty", "", "", 1, false},
		{"exact", "界é🙂", "界é🙂", 3, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, truncated := operation.BoundOutput(test.text, test.limit)
			if got != test.want || truncated != test.truncated {
			}
			if !utf8.ValidString(got) {
				t.Fatal("output is invalid UTF-8")
			}
		})
	}
}

func TestShellRejectsInvalidOutputLimits(t *testing.T) {
	for _, limit := range []int{0, -1, operation.MaxOutputLength + 1, 1_000_000_000} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			if _, err := operation.NewShellSpec(operation.ShellInput{Shell: "/bin/sh"}, t.TempDir(), limit); err == nil {
				t.Fatal("shell spec accepted an invalid limit")
			}
			shell, err := operation.NewShellSpec(operation.ShellInput{Shell: "/bin/sh"}, t.TempDir(), 1)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := operation.DecodeShellState(operation.Operation{Type: shell.Type, Version: shell.Version, State: shell.State, MaxOutputLength: limit}); err == nil {
				t.Fatal("shell operation accepted an invalid limit")
			}
		})
	}
}

func TestShellAcceptsMaximumOutputLength(t *testing.T) {
	shell, err := operation.NewShellSpec(operation.ShellInput{Shell: "/bin/sh"}, t.TempDir(), operation.MaxOutputLength)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := operation.DecodeShellState(operation.Operation{Type: shell.Type, Version: shell.Version, State: shell.State, MaxOutputLength: shell.MaxOutputLength}); err != nil {
		t.Fatal(err)
	}
}

func TestShellPreparesOutputBeforePublishingCompletion(t *testing.T) {
	for _, test := range []struct {
		name      string
		out       []byte
		size      int64
		limit     int
		want      string
		truncated bool
	}{
		{"invalid UTF8", []byte{'a', 0xff, 'b'}, 3, 3, "a�b", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			exitCode := 0
			base := t.TempDir()
			encoded, err := json.Marshal(operation.ShellState{

				PendingExitCode: &exitCode,
			})
			if err != nil {
				t.Fatal(err)
			}
			current := operation.Operation{ID: "shell", Type: operation.TypeShell, Version: operation.VersionShell, Status: operation.StatusAwaiting, State: encoded, MaxOutputLength: test.limit}
			shell, err := operation.NewShell(current)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := shell.Handle(&primitives.PrimitiveEvent{
			}); err != nil {
				t.Fatal(err)
			}
			step, err := shell.Handle(&primitives.PrimitiveEvent{
			})
			if err != nil {
				t.Fatal(err)
			}
			var state operation.ShellState
			if err := json.Unmarshal(step.Operation.State, &state); err != nil {
				t.Fatal(err)
			}
				t.Fatalf("state = %#v, result = %#v", state, state.Result)
			}
			if state.OutPath != filepath.Join(base, "shell", "out") || state.ErrPath != filepath.Join(base, "shell", "err") {
				t.Fatalf("capture paths = %q, %q", state.OutPath, state.ErrPath)
			}
		})
	}
}

func TestShellBoundsOperationErrors(t *testing.T) {
	spec, err := operation.NewShellSpec(operation.ShellInput{Shell: "/bin/sh"}, t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}
	current := operation.Operation{ID: "shell", Type: spec.Type, Version: spec.Version, Status: operation.StatusAwaiting, State: spec.State, MaxOutputLength: 3}
	step, err := advanceShellOnce(t, current, &primitives.PrimitiveEvent{
		Source: "shell", Type: primitives.PrimitiveEventFailed,
		Result: primitives.PrimitiveFailureResult{Error: "éééé"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var state operation.ShellState
	if err := json.Unmarshal(step.Operation.State, &state); err != nil {
		t.Fatal(err)
	}
		t.Fatalf("state = %#v", state)
	}
}

func TestShellReplacesPreviouslyTruncatedErrors(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		state := operation.ShellState{
			Input: operation.ShellInput{Shell: "/bin/sh"}, BaseDirectory: t.TempDir(),
		}
		encoded, err := json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		current := operation.Operation{ID: "shell", Type: operation.TypeShell, Version: operation.VersionShell, Status: operation.StatusAwaiting, State: encoded, MaxOutputLength: 3}
		event := &primitives.PrimitiveEvent{Source: "shell", Type: primitives.PrimitiveEventFailed, Result: primitives.PrimitiveFailureResult{Error: "abcdef"}}
		if cancel {
			current.Status, event = operation.StatusCanceling, nil
		}
		step, err := advanceShellOnce(t, current, event)
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			state, err = operation.DecodeShellState(*step.Operation)
			if err != nil {
				t.Fatal(err)
			}
			if state.TerminalError != want || !state.ErrorTruncated {
				t.Fatalf("cancel %t: error = %q, want %q", cancel, state.TerminalError, want)
			}
		}
	}
}
