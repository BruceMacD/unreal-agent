package responsesapi

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/llm"
)

func TestRequestBodyRejectsInvalidItemPayloads(t *testing.T) {
	tests := []struct {
		name string
		item llm.Item
		want string
	}{
		{
			name: "message",
			item: llm.Item{Type: llm.ItemMessage},
			want: "input item 0: message item data must be llm.Message, got <nil>",
		},
		{
			name: "tool call",
			item: llm.Item{Type: llm.ItemToolCall},
			want: "input item 0: tool_call item data must be llm.ToolCall, got <nil>",
		},
		{
			name: "tool result",
			item: llm.Item{Type: llm.ItemToolResult},
			want: "input item 0: tool_result item data must be llm.ToolResult, got <nil>",
		},
		{
			name: "reasoning",
			item: llm.Item{Type: llm.ItemReasoning},
			want: "input item 0: reasoning item data must be llm.Reasoning, got <nil>",
		},
		{
			name: "reasoning without raw",
			item: llm.Item{Type: llm.ItemReasoning, Data: llm.Reasoning{Summary: []string{"thought"}}},
			want: "input item 0: reasoning item must carry the provider item in Raw",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err == nil || err.Error() != test.want {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestRequestInputItemPreservesIdentifiedInputMessageRoles(t *testing.T) {
	roles := []llm.Role{llm.RoleUser, llm.RoleSystem}
	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			item, err := requestInputItem(llm.Item{
				ProviderID: "message-1",
				Type:       llm.ItemMessage,
				Data:       llm.Message{Role: role, Text: "hello"},
			})
			if err != nil {
				t.Fatalf("convert item: %v", err)
			}
			body, err := json.Marshal(item)
			if err != nil {
				t.Fatalf("marshal item: %v", err)
			}
			var got struct {
				Content []struct {
					Text string `json:"text"`
					Type string `json:"type"`
				} `json:"content"`
				ID     *string `json:"id"`
				Role   string  `json:"role"`
				Status string  `json:"status"`
				Type   string  `json:"type"`
			}
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("unmarshal item: %v", err)
			}
			if got.ID != nil || got.Role != string(role) || got.Status != "" || got.Type != "message" {
				t.Fatalf("item = %#v", got)
			}
			if len(got.Content) != 1 || got.Content[0].Text != "hello" || got.Content[0].Type != "input_text" {
				t.Fatalf("content = %#v", got.Content)
			}
		})
	}
}

func TestRequestInputItemReplaysRawReasoningVerbatim(t *testing.T) {
	raw := `{"id":"reasoning-1","type":"reasoning","status":"completed","summary":[],` +
		`"content":[{"type":"reasoning_text","text":"verbatim"}],"encrypted_content":"opaque"}`
	item, err := requestInputItem(llm.Item{
		ProviderID: "ignored",
		Type:       llm.ItemReasoning,
		Data: llm.Reasoning{
			Summary: []string{"stale summary"},
			Raw:     jsontext.Value(raw),
		},
	})
	if err != nil {
		t.Fatalf("convert item: %v", err)
	}
	body, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal item: %v", err)
	}
	var got, want map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal item: %v", err)
	}
	if err := json.Unmarshal([]byte(raw), &want); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("item = %#v, want %#v", got, want)
	}
}

func TestRequestBodyOmitsUnsetMaxOutputTokens(t *testing.T) {
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}
	var request map[string]any
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if _, exists := request["max_output_tokens"]; exists {
		t.Fatalf("request = %#v", request)
	}
}

func TestRequestBodyEncodesReasoningEffort(t *testing.T) {
	body, err := requestBody(llm.Request{
		Model: llm.Model{ID: "gpt-test", ReasoningEffort: llm.ReasoningEffortHigh},
	if err != nil {
		t.Fatal(err)
	}
	var request struct {
		Reasoning struct {
			Effort  string `json:"effort"`
			Summary string `json:"summary"`
		} `json:"reasoning"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatal(err)
	}
	if request.Reasoning.Effort != "high" || request.Reasoning.Summary != "auto" {
		t.Fatalf("reasoning = %#v", request.Reasoning)
	}
}

func TestRequestBodyRejectsUnsupportedReasoningEffort(t *testing.T) {
	_, err := requestBody(llm.Request{
		Model: llm.Model{ID: "gpt-test", ReasoningEffort: "maximum"},
	if err == nil || err.Error() != `unsupported reasoning effort "maximum"` {
		t.Fatalf("error = %v", err)
	}
}

func TestRequestBodyEncodesHostedWebSearch(t *testing.T) {
	body, err := requestBody(llm.Request{
		Model: llm.Model{ID: "gpt-test"},
		Input: []llm.Item{{
			Type: llm.ItemMessage,
			Data: llm.Message{Role: llm.RoleUser, Text: "latest news"},
		}},
		Tools: []llm.Tool{{Type: llm.ToolHosted, Name: "web_search"}},
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}

	var request struct {
		Tools []struct {
			Type string `json:"type"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if len(request.Tools) != 1 || request.Tools[0].Type != "web_search" {
		t.Fatalf("tools = %#v", request.Tools)
	}
}

func TestRequestBodyRejectsUnsupportedHostedTool(t *testing.T) {
	_, err := requestBody(llm.Request{
		Tools: []llm.Tool{{Type: llm.ToolHosted, Name: "unknown"}},
	if err == nil || err.Error() != `tool 0: unsupported hosted tool name "unknown"` {
		t.Fatalf("error = %v", err)
	}
}

func TestRequestBodyRejectsUnsupportedToolType(t *testing.T) {
	_, err := requestBody(llm.Request{
		Tools: []llm.Tool{{Name: "weather"}},
	if err == nil || err.Error() != `tool 0: unsupported tool type ""` {
		t.Fatalf("error = %v", err)
	}
}

func TestRequestBodyIsByteStableAcrossEncodings(t *testing.T) {
	request := validRequest()
	request.Tools = []llm.Tool{{
		Type:        llm.ToolFunction,
		Name:        "Bash",
		Description: "Execute a shell command.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command":          map[string]any{"type": "string", "description": "The shell command."},
				"max_output_chars": map[string]any{"type": "integer", "description": "Inline budget."},
				"timeout_seconds":  map[string]any{"type": "integer", "description": "Deadline."},
			},
			"required": []any{"command"},
		},
	}}
	if err != nil {
		t.Fatal(err)
	}
	for attempt := range 64 {
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != string(first) {
			t.Fatalf("encoding %d differs:\n%s\n%s", attempt, first, body)
		}
	}
}

func TestRequestBodyEncodesMaxReasoningEffort(t *testing.T) {
	body, err := requestBody(llm.Request{
		Model: llm.Model{ID: "gpt-test", ReasoningEffort: llm.ReasoningEffortMax},
	if err != nil {
		t.Fatal(err)
	}
	var request struct {
		Reasoning struct {
			Effort string `json:"effort"`
		} `json:"reasoning"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatal(err)
	}
	if request.Reasoning.Effort != "max" {
		t.Fatalf("reasoning = %#v", request.Reasoning)
	}
}
