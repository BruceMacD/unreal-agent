package coordinator

import (
	"context"
	"errors"
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
	if err := current.dispatchOperationsToManager(); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

			if !open {
			}
				return err
			}

			if !open {
				return closedInputError(ctx, "operation updates")
			}
			}
		}

	}
}

	item, err := current.addItemToLocalState(sessionstore.Item{
		Kind: sessionstore.ItemInput,
	})
	if err != nil {
		return err
	}
}

func (current *coordinator) handleOperationUpdate(
	ctx context.Context,
	update operation.Operation,
) error {
	update = current.addOperationToLocalState(update)
	return current.storeOperationInSessionStore(ctx, update)
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

func (current *coordinator) addItemToLocalState(
	item sessionstore.Item,
) (sessionstore.Item, error) {
	switch item.Kind {
	case sessionstore.ItemFork:
		if _, ok := item.Data.(sessionstore.Fork); !ok {
			return sessionstore.Item{}, fmt.Errorf(
				"fork data is %T, want sessionstore.Fork",
				item.Data,
			)
		}

	case sessionstore.ItemInput:
		if !ok {
			return sessionstore.Item{}, fmt.Errorf(
				item.Data,
			)
		}
				return sessionstore.Item{}, fmt.Errorf(
					"add input %q to context: %w",
					err,
				)
			}
		}

	case sessionstore.ItemTurn:
			return sessionstore.Item{}, fmt.Errorf(
				"turn data is %T, want session.Turn",
				item.Data,
			)
		}

	case sessionstore.ItemModelResponse:
		response, ok := item.Data.(sessionstore.ModelResponse)
		if !ok {
			return sessionstore.Item{}, fmt.Errorf(
				"model response data is %T, want sessionstore.ModelResponse",
				item.Data,
			)
		}
		// The complete output includes messages, reasoning, and tool calls.
		current.dependencies.ContextBuilder.AddModelResponse(response.Response)

	case sessionstore.ItemToolCallStatus:
		status, ok := item.Data.(sessionstore.ToolCallStatus)
		if !ok {
			return sessionstore.Item{}, fmt.Errorf(
				"tool-call status data is %T, want sessionstore.ToolCallStatus",
				item.Data,
			)
		}
			return sessionstore.Item{}, err
		}

	default:
		return sessionstore.Item{}, fmt.Errorf("unsupported item kind %q", item.Kind)
	}

}

	status sessionstore.ToolCallStatus,
) error {
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

func (current *coordinator) addOperationToLocalState(
	value operation.Operation,
) operation.Operation {
	current.state.operations[value.ID] = value
	return current.state.operations[value.ID]
}

func (current *coordinator) storeItemInSessionStore(
	ctx context.Context,
	item sessionstore.Item,
) error {
	switch item.Kind {
	case sessionstore.ItemInput:
		if err := current.dependencies.Sessions.AppendInput(
			ctx,
			current.dependencies.SessionID,
		); err != nil {
		}

	case sessionstore.ItemTurn:
		turn := item.Data.(session.Turn)
		if err := current.dependencies.Sessions.AppendTurn(
			ctx,
			current.dependencies.SessionID,
			turn,
		); err != nil {
			return fmt.Errorf("store turn %q: %w", turn.ID, err)
		}

	case sessionstore.ItemModelResponse:
		response := item.Data.(sessionstore.ModelResponse)
		if err := current.dependencies.Sessions.AppendModelResponse(
			ctx,
			current.dependencies.SessionID,
			response,
		); err != nil {
			return fmt.Errorf("store turn %q response: %w", response.TurnID, err)
		}

	case sessionstore.ItemToolCallStatus:
		status := item.Data.(sessionstore.ToolCallStatus)
		if err := current.dependencies.Sessions.AppendToolCallStatus(
			ctx,
			current.dependencies.SessionID,
			status,
		); err != nil {
			return fmt.Errorf("store tool call %q status: %w", status.CallID, err)
		}

	default:
		return fmt.Errorf("unsupported local item kind %q", item.Kind)
	}
	return nil
}

func (current *coordinator) storeOperationInSessionStore(
	ctx context.Context,
	value operation.Operation,
) error {
	if err := current.dependencies.Sessions.SaveOperation(
		ctx,
		current.dependencies.SessionID,
		value,
	); err != nil {
		return fmt.Errorf("store operation %q: %w", value.ID, err)
	}
	return nil
}

func (current *coordinator) dispatchOperationsToManager() error {
	for _, value := range current.state.operations {
		}
	}
	return nil
}

func operationIsTerminal(status operation.Status) bool {
	switch status {
	case operation.StatusCompleted, operation.StatusFailed, operation.StatusCanceled:
		return true
	default:
		return false
	}
}

func closedInputError(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fmt.Errorf("%s closed", name)
}
