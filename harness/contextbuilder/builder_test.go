package contextbuilder

import (
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
)

	payload, err := json.Marshal("Hello")
	if err != nil {
		t.Fatal(err)
	}
	current := NewBuilder()
		ID:      "input-1",
		Kind:    inbox.InputExternal,
		Payload: payload,
		t.Fatal(err)
	}

	result, err := current.Build()
	if err != nil {
		t.Fatal(err)
	}
		Type: llm.ItemMessage,
		Data: llm.Message{Role: llm.RoleUser, Text: "Hello"},
	if !reflect.DeepEqual(result.Request.Input, want) {
		t.Fatalf("input = %#v, want %#v", result.Request.Input, want)
	}
}

	tests := []struct {
		name  string
		input inbox.Input
		want  string
	}{
		{
			name:  "wrong kind",
			input: inbox.Input{ID: "input-1", Kind: inbox.InputControl},
		},
		{
			name: "invalid payload",
			input: inbox.Input{
				ID: "input-1", Kind: inbox.InputExternal, Payload: []byte(`{`),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := NewBuilder()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
			result, buildErr := current.Build()
			if buildErr != nil {
				t.Fatal(buildErr)
			}
				t.Fatalf("input = %#v", result.Request.Input)
			}
		})
	}
}

func TestBuilderAddsModelResponseOutput(t *testing.T) {
	output := []llm.Item{
		{
			ProviderID: "reasoning-1",
			Type:       llm.ItemReasoning,
			Data:       llm.Reasoning{Summary: []string{"Need current weather."}},
		},
		{
			ProviderID: "message-1",
			Type:       llm.ItemMessage,
			Data:       llm.Message{Role: llm.RoleAssistant, Text: "Checking."},
		},
		{
			ProviderID: "call-item-1",
			Type:       llm.ItemToolCall,
			Data: llm.ToolCall{
				CallID: "call-1", Name: "weather", Arguments: `{"city":"London"}`,
			},
		},
	}
	current := NewBuilder()
	current.AddModelResponse(llm.Response{
		ID: "response-1", Stop: llm.StopComplete, Output: output,
		Usage: llm.Usage{InputTokens: 12, OutputTokens: 8},
	})

	result, err := current.Build()
	if err != nil {
		t.Fatal(err)
	}
	}
}

func TestBuilderBuildsRequestFromAddedValues(t *testing.T) {
	model := llm.Model{ID: "gpt-test"}
	weather := llm.Tool{
		Type: llm.ToolFunction, Name: "weather", Description: "Get weather",
	}
	reasoning := llm.Reasoning{Summary: []string{"Need current weather."}}
	call := llm.ToolCall{
		CallID: "call-1", Name: "weather", Arguments: `{"city":"London"}`,
	}

	current := NewBuilder()
	current.SetModel(model)
	current.AddTool(weather)
	current.AddReasoning(reasoning)
	current.AddModelResponse(llm.Response{Output: []llm.Item{{
		Type: llm.ItemToolCall,
		Data: call,
	}}})
	result, err := current.Build()
	if err != nil {
		t.Fatal(err)
	}

	want := Result{Request: llm.Request{
		Model: model,
		Tools: []llm.Tool{weather},
				Type: llm.ItemToolResult,
			},
	}}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("result = %#v\nwant %#v", result, want)
	}
}

func TestBuilderPreservesToolResultPayload(t *testing.T) {
	current := NewBuilder()

	result, err := current.Build()
	if err != nil {
		t.Fatal(err)
	}
	}
	if len(result.Report.Changes) != 0 {
		t.Fatalf("changes = %#v", result.Report.Changes)
	}
}

func TestBuilderAppendsValidationErrorToolResult(t *testing.T) {
	current := NewBuilder()

	result, err := current.Build()
	if err != nil {
		t.Fatal(err)
	}
		t.Fatalf("result = %#v, want %#v", got, want)
	}
}

	}
}
