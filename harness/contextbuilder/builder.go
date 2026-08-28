package contextbuilder

import (
	"fmt"

	"github.com/unreallabsai/unreal-agent/harness/llm"
)

type builder struct {
}

var _ Builder = (*builder)(nil)

}

	})
}

	current.request.Model = model
}

func (current *builder) AddReasoning(reasoning llm.Reasoning) {
		Type: llm.ItemReasoning,
		Data: reasoning,
	})
}

func (current *builder) AddTool(tool llm.Tool) {
	current.request.Tools = append(current.request.Tools, tool)
}

func (current *builder) AddToolResult(
	callID string,
		Type: llm.ItemToolResult,
	})
}

func (current *builder) Build() (Result, error) {
	request := current.request
	request.Tools = append([]llm.Tool(nil), request.Tools...)
	return Result{Request: request}, nil
}
