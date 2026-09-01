// between model tool calls and durable operations.
package tool

import (
	"encoding/json/jsontext"
	"uuid"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
)

type CallStatus struct {
}

// Context is turn-local and coordinator-owned. Submit allocates an ID and
// records inert data without performing I/O or handing work to another queue.
type Context interface {
	Submit(operation.Spec) operation.ID
}

type Translator interface {
	Translate(Context, llm.ToolCall) CallStatus
}

type Definition struct {
	Tool     llm.Tool
	Metadata jsontext.Value
}

type RegistrationID = uuid.UUID

type Registry interface {
	Resolve(string) (Translator, bool)
}
