package tool

import (
	"encoding/json/v2"
	"fmt"
	"strings"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
)

type skillUseTranslator struct {
	registry *registry
}

type skillUseArguments struct {
	Name string `json:"name"`
}

func (translator *skillUseTranslator) Translate(ctx Context, call llm.ToolCall) CallStatus {
	if call.Name != "" && call.Name != SkillUseName {
		return CallStatus{Error: fmt.Sprintf(
			"skill-use call name %q does not match static tool %q",
			call.Name,
			SkillUseName,
		)}
	}
	encodedArguments := call.Arguments
	if strings.TrimSpace(encodedArguments) == "" {
		encodedArguments = "{}"
	}
	var arguments skillUseArguments
	if err := json.Unmarshal([]byte(encodedArguments), &arguments); err != nil {
		return CallStatus{Error: fmt.Sprintf("decode skill-use arguments: %v", err)}
	}
	if strings.TrimSpace(arguments.Name) == "" {
		return CallStatus{Error: `skill-use argument "name" must be set`}
	}
	skill, exists := translator.registry.resolveSkill(arguments.Name)
	if !exists {
		return CallStatus{Error: fmt.Sprintf("skill %q is not registered", arguments.Name)}
	}
	spec, err := operation.NewSkillUseSpec(skill.Path)
	if err != nil {
		return CallStatus{Error: fmt.Sprintf("build skill-use operation: %v", err)}
	}
	id := ctx.Submit(spec)
	return CallStatus{WaitingFor: []operation.ID{id}}
}

func (translator *skillUseTranslator) TranslateResult(
	callID string,
	status CallStatus,
	operations []operation.Operation,
) (llm.ToolResult, error) {
	if status.Error != "" {
	}
	if len(operations) != 1 {
		return llm.ToolResult{}, fmt.Errorf(
			"skill-use call %q has %d operations, want 1",
			callID,
			len(operations),
		)
	}
	current := operations[0]
	state, err := operation.DecodeSkillUse(current)
	if err != nil {
		return llm.ToolResult{}, fmt.Errorf("decode skill-use call %q result: %w", callID, err)
	}
	switch current.Status {
	case operation.StatusCompleted:
		return llm.ToolResult{
			CallID: callID,
		}, nil
	case operation.StatusReady, operation.StatusAwaiting, operation.StatusCanceling:
	case operation.StatusCanceled, operation.StatusFailed:
		if state.TerminalError == "" {
			return llm.ToolResult{}, fmt.Errorf(
				"skill-use call %q terminal operation %q has no error",
				callID,
				current.ID,
			)
		}
	default:
		return llm.ToolResult{}, fmt.Errorf(
			"skill-use call %q operation %q has unsupported status %q",
			callID,
			current.ID,
			current.Status,
		)
	}
}
