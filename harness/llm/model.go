package llm


type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

type ItemType string

const (
	ItemMessage    ItemType = "message"
	ItemToolCall   ItemType = "tool_call"
	ItemToolResult ItemType = "tool_result"
	ItemReasoning  ItemType = "reasoning"
)

type Item struct {
	ProviderID string
	Type       ItemType
	Data       any
}

type Message struct {
	Role  Role
	Text  string
	Phase string
}

type ToolCall struct {
	CallID    string
	Name      string
	Arguments string
}

type ToolResult struct {
	CallID string
}

// Raw is the provider's verbatim reasoning item. A provider may attach state to
// it that the harness cannot reconstruct, such as encrypted reasoning content, so
// adapters replay Raw unchanged instead of re-encoding Summary.
type Reasoning struct {
}

type ToolType string

const (
	ToolFunction ToolType = "function"
	ToolHosted   ToolType = "hosted"
)

type Tool struct {
	Type        ToolType
	Name        string
	Description string
	Parameters  map[string]any
}

type Model struct {
	ID              string
	MaxOutputTokens *int64
}

type Request struct {
	Model Model
	Input []Item
	Tools []Tool
}

type StopReason string

const (
	StopComplete        StopReason = "complete"
	StopMaxOutputTokens StopReason = "max_output_tokens"
	StopRefused         StopReason = "refused"
)

type Response struct {
	ID      string
	Stop    StopReason
	Usage   Usage
	Failure *Failure
}

// InputTokens includes CachedInputTokens and CacheWriteInputTokens.
// OutputTokens includes ReasoningTokens.
type Usage struct {
	InputTokens           int64
	CachedInputTokens     int64
	CacheWriteInputTokens int64
	OutputTokens          int64
	ReasoningTokens       int64
}

type Failure struct {
	Code    string
	Message string
}
