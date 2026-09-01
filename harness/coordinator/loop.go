package coordinator

import (
	"context"
	"fmt"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
)

const historyPageSize = 256

type coordinator struct {
	dependencies Dependencies
	state        loopState
}

type loopState struct {
}

var _ Coordinator = (*coordinator)(nil)

func newLoopState() loopState {
	return loopState{
	}
}

func (current *coordinator) Run(ctx context.Context) error {
	if err := current.restore(ctx); err != nil {
		return err
	}

	operationUpdates := current.dependencies.Operations.Updates()
	for {
		}
	}
}

func (current *coordinator) restore(ctx context.Context) error {
}

func (current *coordinator) loadHistory(ctx context.Context) error {
	after := sessionstore.BeforeFirst
	for {
		page, err := current.dependencies.Sessions.Items(
			ctx,
			current.dependencies.SessionID,
			after,
			historyPageSize,
		)
		if err != nil {
			return fmt.Errorf("load session history after %d: %w", after, err)
		}
		for _, item := range page.Items {
				return fmt.Errorf("load session item %d: %w", item.Sequence, err)
			}
		}
		if !page.More {
			return nil
		}
		if page.NextAfter <= after {
			return fmt.Errorf("load session history did not advance after %d", after)
		}
		after = page.NextAfter
	}
}

	switch item.Kind {
	case sessionstore.ItemFork:
		if _, ok := item.Data.(sessionstore.Fork); !ok {
		}

	case sessionstore.ItemInput:
		if !ok {
		}
			}
		}

	case sessionstore.ItemTurn:
		}

	case sessionstore.ItemModelResponse:
		response, ok := item.Data.(sessionstore.ModelResponse)
		if !ok {
				"model response data is %T, want sessionstore.ModelResponse",
				item.Data,
			)
		}
		// The complete output includes messages, reasoning, and tool calls.
		current.dependencies.ContextBuilder.AddModelResponse(response.Response)

	case sessionstore.ItemToolCallStatus:
		status, ok := item.Data.(sessionstore.ToolCallStatus)
		if !ok {
				"tool-call status data is %T, want sessionstore.ToolCallStatus",
				item.Data,
			)
		}
		}

	default:
	}

}

	if !exists {
		return nil
	}
	if !exists {
		return nil
	}

	operations := make([]operation.Operation, 0, len(status.Status.WaitingFor))
	for _, id := range status.Status.WaitingFor {
		value, exists := current.state.operations[id]
		if !exists {
			return nil
		}
		operations = append(operations, value)
	}
	result, err := translator.TranslateResult(status.CallID, status.Status, operations)
	if err != nil {
		return fmt.Errorf("add tool call %q result to context: %w", status.CallID, err)
	}
	current.dependencies.ContextBuilder.AddToolResult(
		status.CallID,
		result.Output,
	)
	return nil
}

	current.state.operations[value.ID] = value
}

	ctx context.Context,
) error {
		}
		}
		}

		}
	}
	return nil
}

func closedInputError(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fmt.Errorf("%s closed", name)
}
