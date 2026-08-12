// Package sessionstore defines append-only session history and operation state.
package sessionstore

import (
	"context"
	"time"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/tool"
)

type Sequence uint64

// BeforeFirst is the cursor before the first assigned sequence.
const BeforeFirst Sequence = 0

type ItemKind string

const (
	ItemFork           ItemKind = "fork"
	ItemInput          ItemKind = "input"
	ItemTurn           ItemKind = "turn"
	ItemModelResponse  ItemKind = "model_response"
	ItemToolCallStatus ItemKind = "tool_call_status"
)

// ToolCallStatus according to Kind.
type Item struct {
	Sequence   Sequence
	RecordedAt time.Time
	Kind       ItemKind
	Data       any
}

type Fork struct {
	ParentID       session.ID
	PreviousTurnID session.TurnID
}

type ModelResponse struct {
	TurnID   session.TurnID
	Response llm.Response
}

type ToolCallStatus struct {
}

type Snapshot struct {
}

type Page struct {
	Items     []Item
	NextAfter Sequence
	More      bool
}

type ResumeState struct {
}

type Store interface {
	Create(context.Context, session.ID) (Snapshot, error)
	Inspect(context.Context, session.ID) (Snapshot, error)
	Items(context.Context, session.ID, Sequence, int) (Page, error)
	AppendTurn(context.Context, session.ID, session.Turn) error
	// SaveOperation stores a complete state; the latest state for its ID wins.
	Resume(context.Context, session.ID) (ResumeState, error)
	Fork(
		ctx context.Context,
		id session.ID,
		parentID session.ID,
		previousTurnID session.TurnID,
	) (Snapshot, error)
}
