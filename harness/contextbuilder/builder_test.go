package contextbuilder

import (
	"reflect"
	"strings"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/llm"
)

	model := llm.Model{ID: "gpt-test"}
	weather := llm.Tool{
		Type: llm.ToolFunction, Name: "weather", Description: "Get weather",
	}
	if err != nil {
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

	if err != nil {
		t.Fatal(err)
	}
	}
	}
}


	if err != nil {
	}
	}
}

	}
}
