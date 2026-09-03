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

// Item Data is Fork, inbox.Input, session.Turn, ModelResponse, or
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
	TurnID     session.TurnID
	CallID     string
	Status     tool.CallStatus
	Operations []operation.Operation `json:",omitempty"`
}

type Snapshot struct {
	Session session.Session
}

type Page struct {
	Items     []Item
	NextAfter Sequence
	More      bool
}

type ResumeState struct {
	Snapshot         Snapshot
	ExternalInputIDs []inbox.ID
}

// Store does not serialize methods for the same session ID.
type Store interface {
	Create(context.Context, session.ID) (Snapshot, error)
	Inspect(context.Context, session.ID) (Snapshot, error)
	Items(context.Context, session.ID, Sequence, int) (Page, error)
	AppendInput(context.Context, session.ID, inbox.Input) error
	AppendTurn(context.Context, session.ID, session.Turn) error
	AppendModelResponse(context.Context, session.ID, ModelResponse) error
	// AppendToolCallStatus appends the status and its operation snapshots.
	// The first append also initializes those operations.
	AppendToolCallStatus(context.Context, session.ID, ToolCallStatus) error
	// SaveOperation stores a complete state; the latest state for its ID wins.
	SaveOperation(context.Context, session.ID, operation.Operation) error
	Resume(context.Context, session.ID) (ResumeState, error)
	Fork(
		ctx context.Context,
		id session.ID,
		parentID session.ID,
		previousTurnID session.TurnID,
	) (Snapshot, error)
}
