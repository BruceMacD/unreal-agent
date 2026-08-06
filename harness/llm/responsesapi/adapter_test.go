package responsesapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/primitives"
)

func TestAdapterRemoteRequestsUseUUIDCorrelationIDs(t *testing.T) {
	adapter := &adapter{endpoint: "https://example.com/responses"}

	if _, err := uuid.Parse(string(first)); err != nil {
		t.Fatalf("first correlation ID = %q: %v", first, err)
	}
	if _, err := uuid.Parse(string(second)); err != nil {
		t.Fatalf("second correlation ID = %q: %v", second, err)
	}
	if first == second {
		t.Fatalf("correlation IDs are equal: %q", first)
	}
}

func TestAdapterResponds(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]any
			t.Errorf("decode request: %v", err)
		}
		requestBody <- body

			"id":"resp-1",
			"object":"response",
			"status":"completed",
			"output":[
				{"id":"message-1","type":"message","role":"assistant","status":"completed","phase":"final_answer","content":[{"type":"output_text","text":"hello","annotations":[],"logprobs":[]}]},
				{"id":"reasoning-1","type":"reasoning","status":"completed","summary":[{"type":"summary_text","text":"Used tools."}],"encrypted_content":"opaque"},
				{"id":"function-1","type":"function_call","call_id":"call-2","name":"weather","arguments":"{\"city\":\"Paris\"}","status":"completed"}
			],
			"usage":{"input_tokens":20,"input_tokens_details":{"cached_tokens":8,"cache_write_tokens":3},"output_tokens":10,"output_tokens_details":{"reasoning_tokens":4},"total_tokens":30}
		}`)
	}))
	defer server.Close()

	adapter := newTestAdapter(t, server.URL+"/responses")
	if err != nil {
		t.Fatalf("respond: %v", err)
	}
	gotRequestBody := <-requestBody
	}
	assertRequestBody(t, gotRequestBody)

	want := llm.Response{
		ID:   "resp-1",
		Stop: llm.StopComplete,
		Output: []llm.Item{
			{
				ProviderID: "message-1", Type: llm.ItemMessage,
				Data: llm.Message{
					Role: llm.RoleAssistant, Text: "hello", Phase: "final_answer",
				},
			},
			{
				ProviderID: "reasoning-1",
				Type:       llm.ItemReasoning,
				Data: llm.Reasoning{
					Summary: []string{"Used tools."},
						`"summary":[{"type":"summary_text","text":"Used tools."}],"encrypted_content":"opaque"}`),
				},
			},
			{
				ProviderID: "function-1",
				Type:       llm.ItemToolCall,
				Data: llm.ToolCall{
					CallID:    "call-2",
					Name:      "weather",
					Arguments: `{"city":"Paris"}`,
				},
			},
		},
		Usage: llm.Usage{
			InputTokens: 20, CachedInputTokens: 8, CacheWriteInputTokens: 3,
			OutputTokens: 10, ReasoningTokens: 4,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("response = %#v\nwant %#v", got, want)
	}
}

func TestAdapterReturnsProviderErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(writer, `{"error":{"code":"invalid_request","message":"model is required","param":"model","type":"invalid_request_error"}}`)
	}))
	defer server.Close()

	adapter := newTestAdapter(t, server.URL+"/responses")
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("error = %#v", err)
	}
	if apiError.StatusCode != http.StatusBadRequest || apiError.Code != "invalid_request" ||
		apiError.Param != "model" || apiError.Type != "invalid_request_error" {
		t.Fatalf("API error = %#v", apiError)
	}
}

func TestRespondCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		select {
		case <-request.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)

	adapter := newTestAdapter(t, server.URL+"/responses")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- err
	}()
	waitForSignal(t, started, "provider request did not start")
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("provider request was not canceled")
	}
}

func TestResponseSeparatesStopReasonsFromFailures(t *testing.T) {
	tests := []struct {
		name string
		body string
		want llm.Response
	}{
		{
			name: "truncated",
			body: `{"id":"resp-1","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[],"usage":{"input_tokens":2,"input_tokens_details":{},"output_tokens":1,"output_tokens_details":{}}}`,
			want: llm.Response{
				ID: "resp-1", Stop: llm.StopMaxOutputTokens, Output: []llm.Item{},
			},
		},
		{
			name: "refused",
			body: `{"id":"resp-2","status":"incomplete","incomplete_details":{"reason":"content_filter"},"output":[],"usage":{"input_tokens":2,"input_tokens_details":{},"output_tokens":0,"output_tokens_details":{}}}`,
			want: llm.Response{
				ID: "resp-2", Stop: llm.StopRefused, Output: []llm.Item{},
			},
		},
		{
			name: "failed",
			body: `{"id":"resp-3","status":"failed","error":{"code":"server_error","message":"failed"},"output":[],"usage":{"input_tokens":2,"input_tokens_details":{},"output_tokens":0,"output_tokens_details":{}}}`,
			want: llm.Response{
				ID: "resp-3", Output: []llm.Item{},
				Failure: &llm.Failure{Code: "server_error", Message: "failed"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := decodeResponse([]byte(test.body))
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("response = (%#v, %v), want %#v", got, err, test.want)
			}
		})
	}
}

func TestResponseKeepsOutputWhenTruncated(t *testing.T) {
	got, err := decodeResponse([]byte(`{
		"id":"resp-1",
		"status":"incomplete",
		"incomplete_details":{"reason":"max_output_tokens"},
		"output":[{"id":"message-1","type":"message","role":"assistant","status":"incomplete","content":[{"type":"output_text","text":"partial","annotations":[],"logprobs":[]}]}]
	}`))
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := llm.Response{
		ID:   "resp-1",
		Stop: llm.StopMaxOutputTokens,
		Output: []llm.Item{{
			ProviderID: "message-1",
			Type:       llm.ItemMessage,
			Data:       llm.Message{Role: llm.RoleAssistant, Text: "partial"},
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("response = %#v\nwant %#v", got, want)
	}
}

func TestResponseReadsRefusalAsMessageText(t *testing.T) {
	got, err := decodeResponse([]byte(`{
		"id":"resp-1",
		"status":"completed",
		"output":[{"id":"message-1","type":"message","role":"assistant","status":"completed","content":[{"type":"refusal","refusal":"I cannot help with that."}]}]
	}`))
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := llm.Message{Role: llm.RoleAssistant, Text: "I cannot help with that."}
	if len(got.Output) != 1 || !reflect.DeepEqual(got.Output[0].Data, want) {
		t.Fatalf("output = %#v", got.Output)
	}
}

func TestResponseCarriesUnknownMessagePhase(t *testing.T) {
	got, err := decodeResponse([]byte(`{
		"id":"resp-1",
		"status":"completed",
		"output":[{"id":"message-1","type":"message","role":"assistant","status":"completed","phase":"analysis","content":[{"type":"output_text","text":"hello","annotations":[],"logprobs":[]}]}]
	}`))
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	message, ok := got.Output[0].Data.(llm.Message)
	if !ok || message.Phase != "analysis" {
		t.Fatalf("output = %#v", got.Output[0].Data)
	}
}

func TestResponseRejectsUnfinishedStatus(t *testing.T) {
	_, err := decodeResponse([]byte(`{"id":"resp-1","status":"queued","output":[]}`))
	if err == nil || !strings.Contains(err.Error(), `unsupported response status "queued"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestResponseRejectsUnsupportedIncompleteReason(t *testing.T) {
	_, err := decodeResponse([]byte(`{"id":"resp-1","status":"incomplete","incomplete_details":{"reason":"unknown"},"output":[]}`))
	if err == nil || !strings.Contains(err.Error(), `unsupported incomplete reason "unknown"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestResponseRejectsUnsupportedOutput(t *testing.T) {
	_, err := decodeResponse([]byte(`{
		"id":"resp-1",
		"status":"completed",
		"output":[{"type":"file_search_call","id":"search-1"}]
	}`))
	if err == nil || !strings.Contains(err.Error(), `unsupported output item type "file_search_call"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestAdapterTracesProviderExchange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
	}))
	defer server.Close()

	remote := primitives.NewRemoteClient()
	t.Cleanup(func() {
		if err := remote.Close(); err != nil {
			t.Errorf("close remote client: %v", err)
		}
	})
	var traced []Exchange
	adapter, err := NewAdapter(remote, Config{
		Endpoint: server.URL + "/responses",
		Trace:    func(exchange Exchange) { traced = append(traced, exchange) },
	})
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}
		t.Fatalf("respond: %v", err)
	}

	if len(traced) != 1 {
		t.Fatalf("traced = %#v", traced)
	}
	if traced[0].StatusCode != http.StatusOK ||
		!strings.Contains(string(traced[0].RequestBody), `"model":"gpt-test"`) ||
		!strings.Contains(string(traced[0].ResponseBody), `"id":"resp-1"`) {
		t.Fatalf("exchange = %#v", traced[0])
	}
}

func TestNewAdapterRequiresDependencies(t *testing.T) {
	adapter, err := NewAdapter(nil, Config{Endpoint: "https://example.com/responses"})
	if err == nil || adapter != nil {
		t.Fatalf("adapter, error = (%#v, %v)", adapter, err)
	}

	remote := primitives.NewRemoteClient()
	t.Cleanup(func() { _ = remote.Close() })
	adapter, err = NewAdapter(remote, Config{})
	if err == nil || adapter != nil {
		t.Fatalf("adapter, error = (%#v, %v)", adapter, err)
	}
}

func TestRequestInputRejectsUnsupportedType(t *testing.T) {
	_, err := requestInputItem(llm.Item{Type: "image"})
	if err == nil || err.Error() != `unsupported input item type "image"` {
		t.Fatalf("error = %v", err)
	}
}

func newTestAdapter(t *testing.T, endpoint string) llm.Adapter {
	t.Helper()
		Endpoint: endpoint,
		Headers: map[string][]string{
			"Authorization": {"Bearer test-key"},
			"Content-Type":  {"application/json"},
		},
	})
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}
	return adapter
}

func validRequest() llm.Request {
	return llm.Request{
		Model: llm.Model{ID: "gpt-test"},
		Input: []llm.Item{{
			Type: llm.ItemMessage,
			Data: llm.Message{Role: llm.RoleUser, Text: "hello"},
		}},
	}
}

func detailedRequest() llm.Request {
	tool := llm.Tool{
		Type:        llm.ToolFunction,
		Name:        "weather",
		Description: "Get weather",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"city": map[string]any{"type": "string"},
			},
			"required":             []any{"city"},
			"additionalProperties": false,
		},
	}
	maxOutputTokens := int64(128)
	return llm.Request{
		Model: llm.Model{ID: "gpt-test", MaxOutputTokens: &maxOutputTokens},
		Input: []llm.Item{
			{
				Type: llm.ItemMessage,
				Data: llm.Message{Role: llm.RoleSystem, Text: "Be concise."},
			},
			{
				Type: llm.ItemMessage,
				Data: llm.Message{Role: llm.RoleUser, Text: "weather?"},
			},
			{
				Type: llm.ItemMessage,
				Data: llm.Message{Role: llm.RoleSystem, Text: "Use tools."},
			},
			{
				ProviderID: "previous-message", Type: llm.ItemMessage,
				Data: llm.Message{
					Role: llm.RoleAssistant, Text: "Checking.", Phase: "commentary",
				},
			},
			{
				ProviderID: "previous-reasoning",
				Type:       llm.ItemReasoning,
				Data: llm.Reasoning{
					Summary: []string{"Checked the request."},
						`"summary":[{"type":"summary_text","text":"Checked the request."}],` +
						`"content":[{"type":"reasoning_text","text":"verbatim"}],` +
						`"encrypted_content":"previous-opaque"}`),
				},
			},
			{
				ProviderID: "previous-tool-call",
				Type:       llm.ItemToolCall,
				Data: llm.ToolCall{
					CallID:    "call-1",
					Name:      "weather",
					Arguments: `{"city":"London"}`,
				},
			},
			{
				Type: llm.ItemToolResult,
			},
		},
		Tools: []llm.Tool{tool},
	}
}

func assertRequest(t *testing.T, request *http.Request, accept string) {
	t.Helper()
	if request.Method != http.MethodPost || request.URL.Path != "/responses" {
		t.Errorf("request = %s %s", request.Method, request.URL.Path)
	}
	if authorization := request.Header.Get("Authorization"); authorization != "Bearer test-key" {
		t.Errorf("authorization = %q", authorization)
	}
	if contentType := request.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("content type = %q", contentType)
	}
	if got := request.Header.Get("Accept"); got != accept {
		t.Errorf("accept = %q, want %q", got, accept)
	}
}

func assertRequestBody(t *testing.T, got map[string]any) {
	t.Helper()
	wantJSON := `{
		"max_output_tokens":128,
		"store":false,
		"include":["reasoning.encrypted_content"],
		"input":[
			{"content":"Be concise.","role":"system"},
			{"content":"weather?","role":"user"},
			{"content":"Use tools.","role":"system"},
			{"content":[{"annotations":[],"logprobs":[],"text":"Checking.","type":"output_text"}],"id":"previous-message","phase":"commentary","role":"assistant","status":"completed","type":"message"},
			{"id":"previous-reasoning","type":"reasoning","status":"completed","summary":[{"type":"summary_text","text":"Checked the request."}],"content":[{"type":"reasoning_text","text":"verbatim"}],"encrypted_content":"previous-opaque"},
			{"arguments":"{\"city\":\"London\"}","call_id":"call-1","name":"weather","id":"previous-tool-call","type":"function_call"},
		],
		"model":"gpt-test",
		"tools":[{
			"description":"Get weather",
			"name":"weather",
			"parameters":{
				"type":"object",
				"properties":{"city":{"type":"string"}},
				"required":["city"],
				"additionalProperties":false
			},
			"strict":false,
			"type":"function"
		}]
	}`
	var want map[string]any
	if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
		t.Fatalf("unmarshal expected request: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("request body =\n%s\nwant\n%s", gotJSON, wantJSON)
	}
}

func waitForSignal(t *testing.T, signal <-chan struct{}, failure string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal(failure)
	}
}
