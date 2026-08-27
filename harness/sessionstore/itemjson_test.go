package sessionstore

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/tool"
)

func TestItemJSONRoundTrip(t *testing.T) {
	recordedAt := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	tests := []Item{
		{Sequence: 1, RecordedAt: recordedAt, Kind: ItemFork, Data: Fork{
			ParentID: "parent", PreviousTurnID: "turn-1",
		}},
		}},
		{Sequence: 1, RecordedAt: recordedAt, Kind: ItemModelResponse, Data: ModelResponse{
			TurnID: "turn-1",
			Response: llm.Response{Output: []llm.Item{{
				Type: llm.ItemMessage, Data: llm.Message{Role: llm.RoleAssistant, Text: "hello"},
			}}},
		}},
		{Sequence: 1, RecordedAt: recordedAt, Kind: ItemToolCallStatus, Data: ToolCallStatus{
		}},
	}

	for _, want := range tests {
		t.Run(string(want.Kind), func(t *testing.T) {
			encoded, err := json.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			var got Item
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("round trip = %#v, want %#v", got, want)
			}
		})
	}
}

func TestItemJSONPreservesNilResponseOutput(t *testing.T) {
	want := Item{
		Sequence: 1,
		Kind:     ItemModelResponse,
		Data:     ModelResponse{TurnID: "turn-1", Response: llm.Response{}},
	}
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got Item
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}

func TestItemJSONRejectsInvalidTagAndData(t *testing.T) {
	invalid := []struct {
		item Item
		want string
	}{
		{item: Item{Kind: ItemFork, Data: session.Turn{}}, want: "sessionstore.Fork"},
		{item: Item{Kind: ItemModelResponse, Data: session.Turn{}}, want: "sessionstore.ModelResponse"},
		{item: Item{Kind: ItemToolCallStatus, Data: session.Turn{}}, want: "sessionstore.ToolCallStatus"},
		{item: Item{Kind: "unknown", Data: struct{}{}}, want: `unsupported item kind "unknown"`},
	}
	for _, test := range invalid {
		if _, err := json.Marshal(test.item); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("marshal error = %v, want %q", err, test.want)
		}
	}

	for _, encoded := range []string{
		`{"Kind":[],"Data":{}}`,
		`{"Sequence":[],"Kind":"turn","Data":{}}`,
		`{"Kind":"unknown","Data":{}}`,
		`{"Kind":"fork","Data":[]}`,
		`{"Kind":"input","Data":[]}`,
		`{"Kind":"turn","Data":[]}`,
		`{"Kind":"model_response","Data":[]}`,
		`{"Kind":"tool_call_status","Data":[]}`,
		`{"Kind":"fork","Data":null}`,
		`{"Kind":"input","Data":null}`,
		`{"Kind":"turn","Data":null}`,
		`{"Kind":"model_response","Data":null}`,
		`{"Kind":"tool_call_status","Data":null}`,
	} {
		var item Item
		if err := json.Unmarshal([]byte(encoded), &item); err == nil {
			t.Fatalf("decoded invalid item %s", encoded)
		}
	}
}
