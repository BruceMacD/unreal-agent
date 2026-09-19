package agentrunner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore/localfile"
)

func TestRunResumesAfterOutputFailure(t *testing.T) {
			workspace, sessions := t.TempDir(), t.TempDir()
			request := `{"session_id":"output-failure","messages":[{"role":"user","content":"hello","message_id":"69621f8d-4f4d-49a5-8f7d-3b24fd855c01"}]}`
			requests := make(chan context.Context, 4)
			client := &fakeClient{respond: func(ctx context.Context, _ llm.Request) (llm.Response, error) {
				requests <- ctx
				return llm.Response{Output: []llm.Item{{Type: llm.ItemMessage, Data: llm.Message{Role: llm.RoleAssistant, Text: "Hello."}}}}, nil
			}}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			run := func(output io.Writer) error {
				return Run(ctx, []string{"-workspace", workspace, "-session-directory", sessions},
					func(name string) string {
						if name == llmAPIKeyEnvironment {
							return "secret"
						}
						return ""
			}
			want := errors.New("output unavailable")
				t.Fatalf("Run error = %v, want original output error", err)
			}
			store, err := localfile.New(sessions)
			if err != nil {
				t.Fatal(err)
			}
			page, err := store.Items(ctx, "output-failure", sessionstore.BeforeFirst, 100)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, item := range page.Items {
			}
			if !found {
				t.Fatal("item whose output failed was not committed")
			}
			var output bytes.Buffer
			if err := run(&output); err != nil {
				t.Fatal(err)
			}
			}
			if len(requests) != 1 {
				t.Fatalf("model calls across failure and retry = %d, want 1", len(requests))
			}
			if err := (<-requests).Err(); !errors.Is(err, context.Canceled) {
				t.Fatalf("model context error = %v, want cancellation", err)
			}
			if err := run(io.Discard); err != nil {
				t.Fatal(err)
			}
			if len(requests) != 0 {
				t.Fatal("settled retry started another model request")
			}
		})
	}
}

func TestRunMainResumesInterruptedDeliveryWithDuplicateInput(t *testing.T) {
	for _, withTool := range []bool{false, true} {
		name := "first-response"
		if withTool {
			name = "tool-result"
		}
		t.Run(name, func(t *testing.T) {
			workspace, sessions := t.TempDir(), t.TempDir()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			run := func(client Client) (int, string, string) {
				var stdout, stderr bytes.Buffer
				code := RunMain(ctx, []string{"-workspace", workspace, "-session-directory", sessions},
					func(name string) string {
						switch name {
						case llmAPIKeyEnvironment:
							return "secret"
						case "SHELL":
							return "/bin/sh"
						default:
							return ""
						}
					}, func() []string { return []string{"PATH=/usr/bin:/bin"} },
				return code, stdout.String(), stderr.String()
			}
			hasResult := func(request llm.Request) bool {
				for _, item := range request.Input {
					if item.Type == llm.ItemToolResult {
						result := item.Data.(llm.ToolResult)
						if result.CallID == "call-1" && result.Output[0].Value == "hello" {
							return true
						}
					}
				}
				return false
			}
			interrupted := &fakeClient{}
			interrupted.respond = func(_ context.Context, request llm.Request) (llm.Response, error) {
				interrupted.calls++
				if withTool && interrupted.calls == 1 {
					return llm.Response{Output: []llm.Item{{Type: llm.ItemToolCall, Data: llm.ToolCall{
						CallID: "call-1", Name: "Bash", Arguments: `{"command":"printf hello"}`,
					}}}}, nil
				}
				if withTool && !hasResult(request) {
					return llm.Response{}, errors.New("missing completed result")
				}
				return llm.Response{}, errors.New("interrupted delivery")
			}
			code, _, stderr := run(interrupted)
			wantCalls := 1
			if withTool {
				wantCalls = 2
			}
			if code != 1 || interrupted.calls != wantCalls || !strings.Contains(stderr, "interrupted delivery") {
				t.Fatalf("first run: exit=%d, calls=%d, stderr=%s", code, interrupted.calls, stderr)
			}
			resumed := &fakeClient{}
			resumed.respond = func(_ context.Context, request llm.Request) (llm.Response, error) {
				resumed.calls++
				if withTool && !hasResult(request) {
					return llm.Response{}, errors.New("missing restored result")
				}
				return llm.Response{Output: []llm.Item{{Type: llm.ItemMessage, Data: llm.Message{Role: llm.RoleAssistant, Text: "Done."}}}}, nil
			}
			code, stdout, stderr := run(resumed)
			if code != 0 || resumed.calls != 1 {
				t.Fatalf("resumed run: exit=%d, calls=%d, stderr=%s", code, resumed.calls, stderr)
			}
			if ids := inputIDs(t, stdout); len(ids) != 0 {
				t.Fatalf("duplicate input was recorded again: %v", ids)
			}
			responses := 0
			for _, kind := range itemKinds(t, stdout) {
				if kind == sessionstore.ItemModelResponse {
					responses++
				}
			}
			if responses != 1 {
				t.Fatalf("recorded responses = %d, want 1", responses)
			}
			resumed.calls = 0
			code, _, stderr = run(resumed)
			if code != 0 || resumed.calls != 0 {
				t.Fatalf("completed retry: exit=%d, calls=%d, stderr=%s", code, resumed.calls, stderr)
			}
		})
	}
}
