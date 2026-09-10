package contextbuilder

import (
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
)

func TestBuilderControlMessages(t *testing.T) {
	for _, test := range []struct {
		mode inbox.ControlMode
		role llm.Role
		text string
	}{
		{mode: inbox.Heartbeat, role: llm.RoleUser, text: "requested"},
		{mode: inbox.StopHard},
		{mode: inbox.StopWhenIdle},
	} {
		t.Run(string(test.mode), func(t *testing.T) {
			builder := NewBuilder()
			builder.AddControlMessage(inbox.ControlMessage{Mode: test.mode, Reason: "requested"})
			result, err := builder.Build()
			if err != nil {
				t.Fatal(err)
			}
			input := result.Request.Input[1:]
			if test.text == "" {
				if len(input) != 0 {
					t.Fatal("control added a model message")
				}
				return
			}
			if len(input) != 1 || input[0].Type != llm.ItemMessage {
				t.Fatalf("input = %#v", input)
			}
			message := input[0].Data.(llm.Message)
			if message.Role != test.role || message.Text != test.text {
				t.Fatalf("message = %#v", message)
			}
		})
	}
}
