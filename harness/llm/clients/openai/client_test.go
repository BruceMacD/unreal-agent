package openai

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/llm"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	client, err := NewClient(Config{APIKey: " "})
	if err == nil || err.Error() != "OpenAI API key must be set" || client != nil {
		t.Fatalf("client, error = (%#v, %v)", client, err)
	}
}

func TestNewClientRequiresBaseURL(t *testing.T) {
	client, err := NewClient(Config{APIKey: "test-key", BaseURL: " "})
	if err == nil || err.Error() != "OpenAI base URL must be set" || client != nil {
		t.Fatalf("client, error = (%#v, %v)", client, err)
	}
}

func TestClientCallsResponsesAPI(t *testing.T) {
	requestSeen := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/responses" {
			t.Errorf("path = %q", request.URL.Path)
		}
		if authorization := request.Header.Get("Authorization"); authorization != "Bearer test-key" {
			t.Errorf("authorization = %q", authorization)
		}
			t.Errorf("accept = %q", accept)
		}
		var body struct {
		}
		if err := json.UnmarshalRead(request.Body, &body); err != nil {
			t.Errorf("decode request: %v", err)
		}
			t.Errorf("body = %#v", body)
		}
		requestSeen <- struct{}{}
	}))
	defer server.Close()

	client, err := NewClient(Config{APIKey: "test-key", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	})
	request := llm.Request{
		Model: llm.Model{ID: "gpt-test"},
		Input: []llm.Item{{
			Type: llm.ItemMessage,
			Data: llm.Message{Role: llm.RoleUser, Text: "hello"},
		}},
	}
	if err != nil {
		t.Fatalf("respond: %v", err)
	}
	<-requestSeen
	if response.ID != "resp-1" || response.Stop != llm.StopComplete {
		t.Fatalf("response = %#v", response)
	}
}
