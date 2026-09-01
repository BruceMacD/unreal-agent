package operation

import (
	"context"
	"fmt"

	"github.com/unreallabsai/unreal-agent/harness/primitives"
)

type LocalOperationManager struct {
	ctx             context.Context
	adds            chan localAddRequest
	cancellations   chan localCancelRequest
	primitiveEvents chan primitives.PrimitiveEvent
	updates         chan Operation
	pendingUpdates  []Operation
}

type localAddRequest struct {
	operation Operation
	result    chan error
}

type localCancelRequest struct {
}

type localRunningOperation struct {
}

var _ Manager = (*LocalOperationManager)(nil)

	manager := &LocalOperationManager{
		ctx:             ctx,
		adds:            make(chan localAddRequest),
		cancellations:   make(chan localCancelRequest),
		primitiveEvents: make(chan primitives.PrimitiveEvent),
		updates:         make(chan Operation),
	}
	go manager.run()
	return manager
}

func (manager *LocalOperationManager) Add(operation Operation) error {
	result := make(chan error, 1)
	request := localAddRequest{operation: operation, result: result}
	select {
	case manager.adds <- request:
	case <-manager.ctx.Done():
		return manager.ctx.Err()
	}
	select {
	case err := <-result:
		return err
	case <-manager.ctx.Done():
		return manager.ctx.Err()
	}
}

	select {
	case manager.cancellations <- request:
	case <-manager.ctx.Done():
		return manager.ctx.Err()
	}
	select {
	case <-manager.ctx.Done():
		return manager.ctx.Err()
	}
}

func (manager *LocalOperationManager) Updates() <-chan Operation {
	return manager.updates
}

func (manager *LocalOperationManager) run() {
	defer close(manager.updates)
	operations := make(map[ID]*localRunningOperation)
	accepted := make(map[ID]struct{})
	activePrimitives := 0

	for {
		var updates chan Operation
		var update Operation
		if len(manager.pendingUpdates) != 0 {
			updates = manager.updates
			update = manager.pendingUpdates[0]
		}

		select {
		case updates <- update:
			manager.pendingUpdates[0] = Operation{}
			manager.pendingUpdates = manager.pendingUpdates[1:]

		case request := <-manager.adds:
			if _, exists := accepted[request.operation.ID]; exists {
				request.result <- nil
				continue
			}
			if err != nil {
				request.result <- err
				continue
			}
			accepted[request.operation.ID] = struct{}{}
			operations[request.operation.ID] = current
			activePrimitives += manager.acceptLocalStep(operations, current, step)
			request.result <- nil

		case request := <-manager.cancellations:
			current, exists := operations[request.id]
				current.cancel()
			}

		case event := <-manager.primitiveEvents:
			completed := localPrimitiveCompleted(event.Type)
			if completed {
				activePrimitives--
			}
			current, exists := operations[ID(event.Source)]
			if !exists {
				continue
			}
			if completed && current.ctx.Err() != nil {
			}
			if err != nil {
				manager.failLocalOperation(operations, current, err)
				continue
			}
			activePrimitives += manager.acceptLocalStep(operations, current, step)

		case <-manager.ctx.Done():
			for _, current := range operations {
				current.cancel()
			}
			manager.drainLocalPrimitives(activePrimitives)
			return
		}
	}
}

func (manager *LocalOperationManager) acceptLocalStep(
	operations map[ID]*localRunningOperation,
	current *localRunningOperation,
	step Step,
) int {
	}
	started := 0
	for _, dispatch := range step.Dispatches {
		if err := startLocalPrimitive(current.ctx, dispatch, manager.primitiveEvents); err != nil {
			if advanceErr != nil {
				manager.failLocalOperation(operations, current, advanceErr)
				return started
			}
			return started + manager.acceptLocalStep(operations, current, failed)
		}
		started++
	}
	return started
}

func (manager *LocalOperationManager) failLocalOperation(
	operations map[ID]*localRunningOperation,
	current *localRunningOperation,
	err error,
) {
	failed := failLocalOperation(current.operation, err)
	manager.appendLocalUpdate(failed)
	current.cancel()
	delete(operations, current.operation.ID)
}

func (manager *LocalOperationManager) drainLocalPrimitives(active int) {
	for active != 0 {
		event := <-manager.primitiveEvents
		if localPrimitiveCompleted(event.Type) {
			active--
		}
	}
}

func (manager *LocalOperationManager) appendLocalUpdate(operation Operation) {
	operation.State = operation.State.Clone()
	operation.Idempotency = operation.Idempotency.Clone()
	manager.pendingUpdates = append(manager.pendingUpdates, operation)
}

func advanceLocalOperation(current Operation, event *primitives.PrimitiveEvent) (Step, error) {
	switch current.Type {
	default:
		return Step{}, fmt.Errorf(
			"local operation manager does not support type %q: %w",
			current.Type,
			ErrUnsupported,
		)
	}
}

func failLocalOperation(current Operation, err error) Operation {
		if stateErr == nil {
			if stepErr == nil {
			}
		}
	}
	current.Status = StatusFailed
	return current
}

func startLocalPrimitive(
	ctx context.Context,
	dispatch PrimitiveDispatch,
	events chan primitives.PrimitiveEvent,
) error {
	switch dispatch.Type {
	case primitives.PrimitiveDispatchIOCreate:
		request, ok := dispatch.Data.(primitives.IOCreateRequest)
		if !ok {
			return fmt.Errorf("io.create dispatch data is %T, want primitives.IOCreateRequest", dispatch.Data)
		}
		primitives.Create(ctx, request, events)

	case primitives.PrimitiveDispatchIORead:
		request, ok := dispatch.Data.(primitives.IOReadRequest)
		if !ok {
			return fmt.Errorf("io.read dispatch data is %T, want primitives.IOReadRequest", dispatch.Data)
		}
		primitives.ReadFile(ctx, request, events)

	case primitives.PrimitiveDispatchProcessStart:
		request, ok := dispatch.Data.(primitives.ProcessStartRequest)
		if !ok {
			return fmt.Errorf("process.start dispatch data is %T, want primitives.ProcessStartRequest", dispatch.Data)
		}
		primitives.StartProcess(ctx, request, events)

	default:
		return fmt.Errorf("local operation manager does not support dispatch %q", dispatch.Type)
	}

	return nil
}

func localOperationFinished(status Status) bool {
	switch status {
	case StatusCompleted, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

func localPrimitiveCompleted(eventType primitives.PrimitiveEventType) bool {
	switch eventType {
	case primitives.PrimitiveEventIOCreateCompleted,
		primitives.PrimitiveEventIOReadCompleted,
		primitives.PrimitiveEventProcessExited,
		primitives.PrimitiveEventFailed,
		return true
	default:
		return false
	}
}
