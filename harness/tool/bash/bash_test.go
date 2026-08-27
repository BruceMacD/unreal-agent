package bash_test

import (
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
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
	}
	ctx := &recordingContext{}
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
		t.Fatalf("shell configuration = %#v", state)
	}
}

func TestTranslatorAcceptsEmptyCommand(t *testing.T) {
		Shell:         "/bin/bash",
		BaseDirectory: "/operations",
	})
	ctx := &recordingContext{}

	if status.Error != "" || len(status.WaitingFor) != 1 || len(ctx.specs) != 1 {
		t.Fatalf("status = %#v, submitted specs = %d", status, len(ctx.specs))
	}
}

func TestTranslatorRejectsInvalidArguments(t *testing.T) {
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
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := &recordingContext{}
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
	ctx := &recordingContext{}

	if status.Error != "build Bash operation: shell path must be absolute" {
		t.Fatalf("error = %q", status.Error)
	}
	if len(ctx.specs) != 0 {
		t.Fatalf("submitted specs = %d, want 0", len(ctx.specs))
	}
}
