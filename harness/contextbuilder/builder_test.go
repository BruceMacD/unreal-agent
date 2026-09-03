package contextbuilder

import (
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
)

// withPreamble expects the items after the preamble every builder starts with.
func withPreamble(items ...llm.Item) []llm.Item {
	return append([]llm.Item{{
		Type: llm.ItemMessage,
		Data: llm.Message{Role: llm.RoleSystem, Text: preamble},
	}}, items...)
}

func TestBuilderAddsExternalInputAsUserMessage(t *testing.T) {
	payload, err := json.Marshal("Hello")
	if err != nil {
		t.Fatal(err)
	}
	current := NewBuilder()
	if err := current.AddExternalInput(inbox.Input{
		ID:      "input-1",
		Kind:    inbox.InputExternal,
		Payload: payload,
	}); err != nil {
		t.Fatal(err)
	}

	result, err := current.Build()
	if err != nil {
		t.Fatal(err)
	}
	want := withPreamble(llm.Item{
		Type: llm.ItemMessage,
		Data: llm.Message{Role: llm.RoleUser, Text: "Hello"},
	})
	if !reflect.DeepEqual(result.Request.Input, want) {
		t.Fatalf("input = %#v, want %#v", result.Request.Input, want)
	}
}

func TestBuilderRejectsInvalidExternalInput(t *testing.T) {
	tests := []struct {
		name  string
		input inbox.Input
		want  string
	}{
		{
			name:  "wrong kind",
			input: inbox.Input{ID: "input-1", Kind: inbox.InputControl},
			want:  `external input "input-1" has input kind "control"`,
		},
		{
			name: "invalid payload",
			input: inbox.Input{
				ID: "input-1", Kind: inbox.InputExternal, Payload: []byte(`{`),
			},
			want: `decode external input "input-1"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := NewBuilder()
			err := current.AddExternalInput(test.input)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
			result, buildErr := current.Build()
			if buildErr != nil {
				t.Fatal(buildErr)
			}
			if !reflect.DeepEqual(result.Request.Input, withPreamble()) {
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
	want := withPreamble(output...)
	if !reflect.DeepEqual(result.Request.Input, want) {
		t.Fatalf("input = %#v, want %#v", result.Request.Input, want)
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
		Input: withPreamble(
			llm.Item{Type: llm.ItemReasoning, Data: reasoning},
			llm.Item{Type: llm.ItemToolCall, Data: call},
			llm.Item{
				Type: llm.ItemToolResult,
			},
		),
	}}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("result = %#v\nwant %#v", result, want)
	}
}

func TestBuilderPreservesToolResultPayload(t *testing.T) {
	current := NewBuilder()
	current.AddToolResult("call-1", output, false)

	result, err := current.Build()
	if err != nil {
		t.Fatal(err)
	}
	got := result.Request.Input[1].Data.(llm.ToolResult)
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
	if got := result.Request.Input[1].Data.(llm.ToolResult); !reflect.DeepEqual(got, want) {
		t.Fatalf("result = %#v, want %#v", got, want)
	}
}

	}
}

func TestBuilderLeadsSystemPromptWithPreamble(t *testing.T) {
	payload, err := json.Marshal("hello")
	if err != nil {
		t.Fatal(err)
	}
	current := NewBuilder()
	current.SetSystemPrompt("Be concise.")
	if err := current.AddExternalInput(inbox.Input{
		ID: "input-1", Kind: inbox.InputExternal, Payload: payload,
	}); err != nil {
		t.Fatal(err)
	}

	result, err := current.Build()
	if err != nil {
		t.Fatal(err)
	}
	want := []llm.Item{
		{Type: llm.ItemMessage, Data: llm.Message{
			Role: llm.RoleSystem, Text: preamble + "\n\nBe concise.",
		}},
		{Type: llm.ItemMessage, Data: llm.Message{Role: llm.RoleUser, Text: "hello"}},
	}
	if !reflect.DeepEqual(result.Request.Input, want) {
		t.Fatalf("input = %#v, want %#v", result.Request.Input, want)
	}
}
