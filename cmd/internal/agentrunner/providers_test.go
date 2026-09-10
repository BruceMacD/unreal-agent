
import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerProviderRetries(t *testing.T) {
		for _, maxAttempts := range []int{1, 2} {
			t.Run(provider.Name+"/"+strconv.Itoa(maxAttempts), func(t *testing.T) {
				t.Parallel()
				var attempts atomic.Int64
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					var body struct {
						MaxAttempts *int `json:"max_attempts"`
					}
					if err := json.UnmarshalRead(request.Body, &body); err != nil {
						t.Error(err)
					}
					if body.MaxAttempts != nil {
						t.Error("harness retry configuration leaked into provider request")
					}
					attempts.Add(1)
					writer.WriteHeader(http.StatusServiceUnavailable)
				}))
				defer server.Close()
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
					[]string{"-workspace", t.TempDir(), "-session-directory", t.TempDir()},
					func(name string) string {
						switch name {
						case "UNREAL_HARNESS_LLM_PROVIDER":
							return provider.Name
						case "UNREAL_HARNESS_LLM_BASE_URL":
							return server.URL
						case "UNREAL_HARNESS_LLM_API_KEY":
							return "test-key"
						default:
							return ""
						}
					}, func() []string { return nil },
					strings.NewReader(`{"prompt":"hello","model":"test","max_attempts":`+strconv.Itoa(maxAttempts)+`}`),
				if code != 1 || attempts.Load() != int64(maxAttempts) {
					t.Fatalf("exit = %d, attempts = %d, want %d", code, attempts.Load(), maxAttempts)
				}
			})
		}
	}
}

func TestRunnerProviderDefaultModels(t *testing.T) {
		want := ""
		if provider.Name == "openai" {
			want = "gpt-6-astra"
		}
		if provider.DefaultModel != want {
			t.Errorf("%s default model = %q, want %q", provider.Name, provider.DefaultModel, want)
		}
	}
}
