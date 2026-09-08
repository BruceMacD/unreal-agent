package operation_test

import (
	"encoding/json/jsontext"
	"errors"
	"strings"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/operation"
)

func TestNewRemoteJobSpecRoundTripsPlan(t *testing.T) {
	plan := operation.RemoteJobPlan{
		Type:    "test",
		Version: 2,
		Data:    jsontext.Value(`{"value":"preserved"}`),
	}
	spec, err := operation.NewRemoteJobSpec(plan)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Type != operation.TypeRemoteJob || spec.Version != operation.VersionRemoteJob {
		t.Fatalf("spec = %#v", spec)
	}
	state, err := operation.DecodeRemoteJobState(operation.Operation{
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.Plan.Type != plan.Type || state.Plan.Version != plan.Version ||
		string(state.Plan.Data) != string(plan.Data) {
		t.Fatalf("state = %#v", state)
	}
}

	plan := operation.RemoteJobPlan{
		Type:    "test",
		Version: 1,
		Data:    jsontext.Value(`{}`),
	}
	spec, err := operation.NewRemoteJobSpec(plan)
	if err != nil {
		t.Fatal(err)
	}
	current := operation.Operation{
	}
	state, err := operation.DecodeRemoteJobState(current)
	if err != nil {
		t.Fatal(err)
	}
	state.Subscription = jsontext.Value(`{"partial":"response"}`)
	awaiting, err := operation.UpdateRemoteJob(current, state, operation.StatusAwaiting)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		status operation.Status
		apply  func(operation.Operation) (operation.Step, error)
	}{
		{
			name:   "failed",
			status: operation.StatusFailed,
			apply: func(current operation.Operation) (operation.Step, error) {
				return operation.FailRemoteJob(current, errors.New("failed"))
			},
		},
		{
			name:   "canceled",
			status: operation.StatusCanceled,
			apply:  operation.CancelRemoteJob,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			terminal, err := test.apply(*awaiting.Operation)
			if err != nil {
				t.Fatal(err)
			}
			state, err := operation.DecodeRemoteJobState(*terminal.Operation)
			if err != nil {
				t.Fatal(err)
			}
			if terminal.Operation.Status != test.status || len(state.Subscription) != 0 ||
				t.Fatalf("operation = %#v, state = %#v", terminal.Operation, state)
			}
		})
	}
}

func TestNewRemoteJobSpecRejectsInvalidPlans(t *testing.T) {
	tests := []struct {
		name string
		plan operation.RemoteJobPlan
		want string
	}{
		{name: "missing type", plan: operation.RemoteJobPlan{Version: 1, Data: jsontext.Value(`{}`)}, want: "type must be set"},
		{name: "zero version", plan: operation.RemoteJobPlan{Type: "test", Data: jsontext.Value(`{}`)}, want: "version must be positive"},
		{name: "missing data", plan: operation.RemoteJobPlan{Type: "test", Version: 1}, want: "data must be valid JSON"},
		{name: "invalid data", plan: operation.RemoteJobPlan{Type: "test", Version: 1, Data: jsontext.Value(`{`)}, want: "data must be valid JSON"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := operation.NewRemoteJobSpec(test.plan)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestDecodeRemoteJobStateReportsUnsupportedEnvelope(t *testing.T) {
	for _, current := range []operation.Operation{
		{ID: "remote-1", Type: operation.TypeShell, Version: operation.VersionRemoteJob},
		{ID: "remote-1", Type: operation.TypeRemoteJob, Version: operation.VersionRemoteJob + 1},
	} {
		if _, err := operation.DecodeRemoteJobState(current); !errors.Is(err, operation.ErrUnsupported) {
			t.Fatalf("error = %v, want ErrUnsupported", err)
		}
	}
}
