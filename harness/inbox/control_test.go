package inbox_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
)

		t.Run(string(mode), func(t *testing.T) {
			payload, err := json.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			input := inbox.Input{ID: "stop", Kind: inbox.InputControl, Payload: payload}
			if err != nil || got != want {
			}
		})
	}
}

		t.Run(payload, func(t *testing.T) {
			input := inbox.Input{ID: "stop", Kind: inbox.InputControl, Payload: jsontext.Value(payload)}
			if err := newInbox(t).Submit(t.Context(), input); err == nil {
			}
		})
	}
	}
}
