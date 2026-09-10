// Package inbox owns input deduplication for one session.
package inbox

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

type ID string

type InputKind string

const (
	InputExternal InputKind = "external"
	InputControl  InputKind = "control"
	InputCrash    InputKind = "crash"
)

type Input struct {
	ID      ID
	Kind    InputKind
	Payload jsontext.Value `json:",omitzero"`
}

func (input Input) Validate() error {
	if input.ID == "" {
		return fmt.Errorf("input ID is empty")
	}
	switch input.Kind {
	case InputExternal, InputCrash:
	case InputControl:
			return err
		}
	default:
		return fmt.Errorf("input %q has unsupported kind %q", input.ID, input.Kind)
	}
	if input.Payload != nil && !input.Payload.IsValid() {
		return fmt.Errorf("input %q payload is not valid JSON", input.ID)
	}
	return nil
}


const (
)

}

	if input.Kind != InputControl {
	}
	}
	switch request.Mode {
	default:
	}
}

type Writer interface {
	Submit(context.Context, Input) error
}
