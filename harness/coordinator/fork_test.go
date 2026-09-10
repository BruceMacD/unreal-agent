package coordinator

import (
	"encoding/json/jsontext"
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
	"github.com/unreallabsai/unreal-agent/harness/tool"
)

func TestCoordinatorForkStopsWhenChildIsIdle(t *testing.T) {
		for _, childTool := range []bool{false, true} {
			t.Run(fmt.Sprintf("parent=%s/child-tool=%t", parentStage, childTool), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					parent := newStopTestRun(t, 1)
					if parentStage == "unscheduled" {
						parent.store.items = parent.store.items[:2]
					}
					store := persistTestRun(t, parent)
					previousTurn := session.TurnID("turn-1")
					if parentStage == "completed" {
						status := parent.store.items[2].Data.(sessionstore.ToolCallStatus)
						status.Operations[0].Status = operation.StatusCompleted
						if err := store.SaveOperation(t.Context(), "session-1", status.Operations[0]); err != nil {
							t.Fatal(err)
						}
						if err := store.AppendToolCallStatus(t.Context(), "session-1", status); err != nil {
							t.Fatal(err)
						}
						previousTurn = "parent-final"
							t.Fatal(err)
						}
						if err := store.AppendModelResponse(t.Context(), "session-1", sessionstore.ModelResponse{TurnID: previousTurn, Response: textResponse("Parent done.")}); err != nil {
							t.Fatal(err)
						}
					}
					if _, err := store.Fork(t.Context(), "child", "session-1", previousTurn); err != nil {
						t.Fatal(err)
					}
					restored, err := store.Resume(t.Context(), "child")
					if err != nil {
						t.Fatal(err)
					}
					child := newStopTestRun(t, 0)
					child.current.dependencies.SessionID = "child"
					child.current.dependencies.Sessions, child.current.dependencies.Restored = store, restored
					spec, err := operation.NewValueSpec(jsontext.Value(`{"value":1}`))
					if err != nil {
						t.Fatal(err)
					}
					child.current.dependencies.Tools = registry
					child.start(t)
					child.assertRunning(t)
					if len(child.operations.adds) != 0 || len(child.current.state.toolCalls) != 0 {
						t.Fatal("inherited tool call became active child work")
					}
					if child.current.state.currentTurnID != previousTurn {
						t.Fatal("fork lost the parent turn boundary")
					}
					child.input(t, externalEvent(t, 0, "child-user", "hello"), stopInput(t, "child-stop", inbox.StopWhenIdle))
					inheritedCall := false
					for _, item := range child.calls[0].request.Input {
						if item.Type == llm.ItemToolCall && item.Data.(llm.ToolCall).CallID == "call-0" {
							inheritedCall = true
						}
					}
					if !inheritedCall {
						t.Fatal("fork lost its parent context")
					}
					if childTool {
						child.respond(t, 0, llm.Response{Output: []llm.Item{{Type: llm.ItemToolCall, Data: llm.ToolCall{
							CallID: "child-call", Name: tool.BashName, Arguments: `{}`,
						}}}})
						child.assertRunning(t)
						if len(child.operations.adds) != 1 || len(child.current.state.toolCalls) != 1 {
							t.Fatal("child tool call was not tracked")
						}
						completed := child.operations.adds[0]
						completed.Status = operation.StatusCompleted
						child.operations.updates <- completed
						synctest.Wait()
						child.assertRunning(t)
						child.respond(t, 1, textResponse("Child tool done."))
					} else {
						child.respond(t, 0, textResponse("Child done."))
					}
					child.assertStopped(t)
				})
			})
		}
	}
}
