package bash

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/tool"
)

type Config struct {
}

type translator struct {
	config Config
}

}

func (translator *translator) Translate(ctx tool.Context, call llm.ToolCall) tool.CallStatus {
	if err != nil {
	}

	id := ctx.Submit(spec)
	return tool.CallStatus{WaitingFor: []operation.ID{id}}
}

func (translator *translator) TranslateResult(
	callID string,
	status tool.CallStatus,
	operations []operation.Operation,
) (llm.ToolResult, error) {
		}
	}

	if err != nil {
	}
}

func translateOperationResult(
	callID string,
	current operation.Operation,
	if current.Type != operation.TypeShell {
			"bash tool call %q operation %q has type %q, want %q",
			callID,
			current.ID,
			current.Type,
			operation.TypeShell,
		)
	}

	}
	}
	if state.Result != nil {
}

	var arguments map[string]jsontext.Value
	if err := json.Unmarshal([]byte(encoded), &arguments); err != nil {
	}
	encodedCommand, exists := arguments["command"]
	if !exists {
	}
	var command *string
	if err := json.Unmarshal(encodedCommand, &command); err != nil {
	}
	if command == nil {
	}
}

	spec, err := operation.NewShellSpec(operation.ShellInput{
	if err != nil {
		return operation.Spec{}, fmt.Errorf("build Bash operation: %w", err)
	}
	return spec, nil
}
