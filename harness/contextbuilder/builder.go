package contextbuilder

import (
	"encoding/json/v2"
	"fmt"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
)

type builder struct {
}

var _ Builder = (*builder)(nil)

}

		return fmt.Errorf(
		)
	}

	var text string
	}
		Type: llm.ItemMessage,
		Data: llm.Message{Role: llm.RoleUser, Text: text},
	})
	return nil
}

	current.request.Model = model
}

func (current *builder) AddModelResponse(response llm.Response) {
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
