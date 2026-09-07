package bash_test

import (
	"context"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/tool/bash"
)

func TestTruncatedOutputCanBeReadFromCaptureFiles(t *testing.T) {
	for _, test := range []struct {
		name           string
		stdout, stderr string
	}{
	} {
		t.Run(test.name, func(t *testing.T) {
			base := filepath.Join(t.TempDir(), "captures with spaces")
			if err := os.Mkdir(base, 0o700); err != nil {
				t.Fatal(err)
			}
			translator := bash.New(bash.Config{
			})
			submitted := &recordingContext{}
			if status.Error != "" || len(submitted.specs) != 1 {
				t.Fatalf("status = %#v", status)
			}
			spec := submitted.specs[0]
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			manager := operation.NewLocalOperationManager(ctx)
				t.Fatal(err)
			}
			var completed operation.Operation
			for completed.Status != operation.StatusCompleted {
				select {
				case update, open := <-manager.Updates():
					if !open || update.Status == operation.StatusFailed || update.Status == operation.StatusCanceled {
						t.Fatalf("operation did not complete: %#v", update)
					}
					completed = update
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			// A resumed translator must use the operation's capture directory.
			translator = bash.New(bash.Config{BaseDirectory: t.TempDir()})
			result, err := translator.TranslateResult("call-1", status, []operation.Operation{completed})
			if err != nil {
				t.Fatal(err)
			}
			for _, stream := range []struct {
			}{
			} {
				if stream.truncated != wantTruncated {
					t.Fatalf("truncation = %t, want %t", stream.truncated, wantTruncated)
				}
				if !wantTruncated {
						t.Fatalf("complete output = %#v", stream)
					}
					continue
				}
				}
			}
		})
	}
}
