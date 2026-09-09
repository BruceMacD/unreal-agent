package bash

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/tool"
)

type Config struct {
	Shell         string
	Directory     string
	BaseDirectory string
}

type translator struct {
	config Config
}

func New(config Config) tool.Translator {
	return &translator{config: config}
}

func (translator *translator) Translate(ctx tool.Context, call llm.ToolCall) tool.CallStatus {
	command, limit, err := validateArguments(call.Arguments)
	if err != nil {
		return tool.ErrorStatus(err.Error(), limit)
	}
	spec, err := translator.buildOperation(command, limit)
	if err != nil {
		return tool.ErrorStatus(err.Error(), limit)
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

	state, err := operation.DecodeShellState(current)
	if err != nil {
	}
	}
	if state.Result != nil {
		}
	}
}

func validateArguments(encoded string) (string, int, error) {
	var arguments map[string]jsontext.Value
	if err := json.Unmarshal([]byte(encoded), &arguments); err != nil {
		return "", 0, fmt.Errorf("decode Bash arguments: %w", err)
	}
	limit, err := tool.ParseMaxOutputLength(arguments["max_output_length"])
	if err != nil {
		return "", 0, fmt.Errorf("bash argument: %w", err)
	}
	encodedCommand, exists := arguments["command"]
	if !exists {
		return "", limit, errors.New(`bash argument "command" must be set`)
	}
	var command *string
	if err := json.Unmarshal(encodedCommand, &command); err != nil {
		return "", limit, fmt.Errorf(`decode Bash argument "command": %w`, err)
	}
	if command == nil {
		return "", limit, errors.New(`bash argument "command" must be a string`)
	}
	if offset := strings.IndexByte(*command, 0); offset >= 0 {
		return "", limit, fmt.Errorf(`bash argument "command" contains a NUL byte at offset %d`, offset)
	}
	return *command, limit, nil
}

func (translator *translator) buildOperation(command string, limit int) (operation.Spec, error) {
	spec, err := operation.NewShellSpec(operation.ShellInput{
		Command:   command,
		Shell:     translator.config.Shell,
		Directory: translator.config.Directory,
	}, translator.config.BaseDirectory, limit)
	if err != nil {
		return operation.Spec{}, fmt.Errorf("build Bash operation: %w", err)
	}
	return spec, nil
}
