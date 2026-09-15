package inbox_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
)

func TestInboxControlMessages(t *testing.T) {
	for _, mode := range []inbox.ControlMode{inbox.StopHard, inbox.StopWhenIdle, inbox.Heartbeat} {
		t.Run(string(mode), func(t *testing.T) {
			want := inbox.ControlMessage{Mode: mode, Reason: "stop now"}
			payload, err := json.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			input := inbox.Input{ID: "stop", Kind: inbox.InputControl, Payload: payload}
			got, err := submitAndReceive(t, newInbox(t), input).DecodeControlMessage()
			if err != nil || got != want {
				t.Fatalf("control message = %+v, error = %v", got, err)
			}
		})
	}
}

func TestInboxRejectsInvalidControlMessages(t *testing.T) {
	for _, payload := range []string{
		"", "null", `{}`, `{"Mode":"unknown"}`, `{"Mode":"soft"}`, `{"Mode":42}`, `{"Mode":"hard"`,
		`{"Mode":"hard","extra":true}`,
		`{"Mode":"heartbeat","Reason":"waiting","extra":true}`,
		`{"Mode":"heartbeat"}`,
		`{"Mode":"heartbeat","Reason":""}`,
		`{"Mode":"heartbeat","Reason":null}`,
	} {
		t.Run(payload, func(t *testing.T) {
			input := inbox.Input{ID: "stop", Kind: inbox.InputControl, Payload: jsontext.Value(payload)}
			if err := newInbox(t).Submit(t.Context(), input); err == nil {
				t.Fatal("invalid control message accepted")
			}
		})
	}
	if _, err := (inbox.Input{Kind: inbox.InputExternal, Payload: jsontext.Value(`{"Mode":"hard"}`)}).DecodeControlMessage(); err == nil {
		t.Fatal("external input decoded as control message")
	}
}
