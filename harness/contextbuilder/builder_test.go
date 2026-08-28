package contextbuilder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/llm"
)

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
	current.AddTool(weather)
	current.AddReasoning(reasoning)
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
