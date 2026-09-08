package bash_test

import (
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/tool"
	"github.com/unreallabsai/unreal-agent/harness/tool/bash"
)

type recordingContext struct {
	specs []operation.Spec
}

func (ctx *recordingContext) Submit(spec operation.Spec) operation.ID {
	ctx.specs = append(ctx.specs, spec)
	return operation.ID("operation-1")
}

func TestTranslatorSubmitsShellOperation(t *testing.T) {
	config := bash.Config{
		Shell:         "/bin/bash",
		Directory:     "/workspace",
		BaseDirectory: "/operations",
	}
	translator := bash.New(config)
	ctx := &recordingContext{}
	status := translator.Translate(ctx, llm.ToolCall{
		CallID:    "call-1",
		Name:      "Bash",
		Arguments: `{"command":"  printf '%s' \"$HOME\"; exit 7  "}`,
	})

	if status.Error != "" || !reflect.DeepEqual(status.WaitingFor, []operation.ID{"operation-1"}) {
		t.Fatalf("status = %#v", status)
	}
	if len(ctx.specs) != 1 {
		t.Fatalf("submitted specs = %d, want 1", len(ctx.specs))
	}
	spec := ctx.specs[0]
	if spec.Type != operation.TypeShell || spec.Version != operation.VersionShell {
		t.Fatalf("spec type/version = %q/%d", spec.Type, spec.Version)
	}
	var state operation.ShellState
	if err := json.Unmarshal(spec.State, &state); err != nil {
		t.Fatalf("decode shell state: %v", err)
	}
	wantInput := operation.ShellInput{
	}
	if !reflect.DeepEqual(state.Input, wantInput) {
		t.Fatalf("shell input = %#v, want %#v", state.Input, wantInput)
	}
	if state.BaseDirectory != config.BaseDirectory || spec.MaxOutputLength != operation.DefaultMaxOutputLength {
		t.Fatalf("shell configuration = %#v", state)
	}
}

func TestTranslatorTranslatesShellOperationResults(t *testing.T) {
	tests := []struct {
		name   string
		status operation.Status
		state  operation.ShellState
		want   string
	}{
		{
			name:   "completed",
			status: operation.StatusCompleted,
			state: operation.ShellState{
				Input:         operation.ShellInput{Command: "secret command"},
				BaseDirectory: "/secret/path",

				Result: &operation.ShellResult{
					Out:     "ok\n",
					OutSize: 3,
				},
			},
		},
		{
			name:   "empty output",
			status: operation.StatusCompleted,
			state:  operation.ShellState{Result: &operation.ShellResult{}},
		},
		{
			name:   "nonzero exit with stderr",
			status: operation.StatusCompleted,
			state: operation.ShellState{Result: &operation.ShellResult{
				Err: "command failed\n", ErrSize: 15, ExitCode: 7,
			}},
		},
		{
			name:   "truncated stdout",
			status: operation.StatusCompleted,
			}},
		},
		{
			name:   "truncated stderr",
			status: operation.StatusCompleted,
				Out: "output", OutSize: 6,
			}},
		},
		{
			name:   "ready",
			state:  operation.ShellState{OutTruncated: true, ErrTruncated: true},
			status: operation.StatusReady,
		},
		{
			name:   "awaiting",
			state:  operation.ShellState{OutTruncated: true, ErrTruncated: true},
			status: operation.StatusAwaiting,
		},
		{
			name:   "canceling",
			state:  operation.ShellState{OutTruncated: true, ErrTruncated: true},
			status: operation.StatusCanceling,
		},
		{
			name:   "canceled",
			status: operation.StatusCanceled,
			state:  operation.ShellState{OutTruncated: true, ErrTruncated: true, TerminalError: "canceled by user"},
		},
		{
			name:   "failed",
			status: operation.StatusFailed,
			state:  operation.ShellState{OutTruncated: true, ErrTruncated: true, TerminalError: "process failed"},
		},
	}
	translator := bash.New(bash.Config{})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, err := json.Marshal(test.state)
			if err != nil {
				t.Fatal(err)
			}
			result, err := translator.TranslateResult("call-1", tool.CallStatus{}, []operation.Operation{{
				ID:      "operation-1",
				Type:    operation.TypeShell,
				Version: operation.VersionShell, MaxOutputLength: operation.DefaultMaxOutputLength,
				Status: test.status,
				State:  state,
			}})
			if err != nil {
				t.Fatal(err)
			}
			if result.CallID != "call-1" {
				t.Fatalf("call ID = %q", result.CallID)
			}
			}
				}
			}
		})
	}
}

func TestTranslatorTranslatesValidationErrorWithoutOperations(t *testing.T) {
	translator := bash.New(bash.Config{})
	ctx := &recordingContext{}
	status := translator.Translate(ctx, llm.ToolCall{
		CallID:    "call-1",
		Arguments: `{"command":42}`,
	})
	if status.Error == "" || len(ctx.specs) != 0 {
		t.Fatalf("status = %#v, submitted specs = %d", status, len(ctx.specs))
	}

	result, err := translator.TranslateResult("call-1", status, nil)
	if err != nil {
		t.Fatal(err)
	}
	}
}

	translator := bash.New(bash.Config{})
	}
}

func TestTranslatorRejectsInvalidShellOperationResults(t *testing.T) {
	validState, err := json.Marshal(operation.ShellState{})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		operations []operation.Operation
		want       string
	}{
		{
			name: "wrong type",
			operations: []operation.Operation{{
				ID: "operation-1", Type: "other", Version: operation.VersionShell, MaxOutputLength: operation.DefaultMaxOutputLength,
			}},
			want: `has type "other", want "shell"`,
		},
		{
			name: "malformed state",
			operations: []operation.Operation{{
				ID: "operation-1", Type: operation.TypeShell, Version: operation.VersionShell, MaxOutputLength: operation.DefaultMaxOutputLength,
				State: []byte(`{`),
			}},
			want: "decode Bash operation",
		},
		{
			name: "completed without result",
			operations: []operation.Operation{{
				ID: "operation-1", Type: operation.TypeShell, Version: operation.VersionShell, MaxOutputLength: operation.DefaultMaxOutputLength,
				Status: operation.StatusCompleted, State: validState,
			}},
			want: "completed operation",
		},
	}
	translator := bash.New(bash.Config{})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := translator.TranslateResult("call-1", tool.CallStatus{}, test.operations)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestTranslatorAcceptsEmptyCommand(t *testing.T) {
	translator := bash.New(bash.Config{
		Shell:         "/bin/bash",
		BaseDirectory: "/operations",
	})
	ctx := &recordingContext{}
	status := translator.Translate(ctx, llm.ToolCall{Arguments: `{"command":""}`})

	if status.Error != "" || len(status.WaitingFor) != 1 || len(ctx.specs) != 1 {
		t.Fatalf("status = %#v, submitted specs = %d", status, len(ctx.specs))
	}
}

func TestTranslatorRejectsInvalidArguments(t *testing.T) {
	translator := bash.New(bash.Config{
		Shell:         "/bin/bash",
		BaseDirectory: "/operations",
	})
	tests := []struct {
		name      string
		arguments string
		want      string
	}{
		{name: "malformed JSON", arguments: `{`, want: "decode Bash arguments: jsontext: unexpected EOF"},
		{name: "duplicate command", arguments: `{"command":"pwd","command":"ls"}`, want: `duplicate object member name "command"`},
		{name: "missing command", arguments: `{}`, want: `bash argument "command" must be set`},
		{name: "uppercase command", arguments: `{"COMMAND":"pwd"}`, want: `bash argument "command" must be set`},
		{name: "null command", arguments: `{"command":null}`, want: `bash argument "command" must be a string`},
		{name: "non-string command", arguments: `{"command":42}`, want: `decode Bash argument "command": json:`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := &recordingContext{}
			status := translator.Translate(ctx, llm.ToolCall{Arguments: test.arguments})
			if !strings.Contains(status.Error, test.want) {
				t.Fatalf("error = %q, want substring %q", status.Error, test.want)
			}
			if len(status.WaitingFor) != 0 || len(ctx.specs) != 0 {
				t.Fatalf("status = %#v, submitted specs = %d", status, len(ctx.specs))
			}
		})
	}
}

func TestTranslatorReturnsShellConfigurationError(t *testing.T) {
	translator := bash.New(bash.Config{Shell: "bash", BaseDirectory: "/operations"})
	ctx := &recordingContext{}
	status := translator.Translate(ctx, llm.ToolCall{Arguments: `{"command":"pwd"}`})

	if status.Error != "build Bash operation: shell path must be absolute" {
		t.Fatalf("error = %q", status.Error)
	}
	if len(ctx.specs) != 0 {
		t.Fatalf("submitted specs = %d, want 0", len(ctx.specs))
	}
}
