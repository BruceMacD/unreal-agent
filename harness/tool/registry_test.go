package tool

import (
	"reflect"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
)

type fixedTranslator struct {
	status CallStatus
func (translator *fixedTranslator) Translate(Context, llm.ToolCall) CallStatus {
	return translator.status
}

func (translator *fixedTranslator) TranslateResult(
) (llm.ToolResult, error) {
	}
	}
	}
		}
	}
	if translator, exists := registry.Resolve("unknown"); exists || translator != nil {
		t.Fatalf("resolve unknown = (%#v, %t), want (nil, false)", translator, exists)
	}
}

