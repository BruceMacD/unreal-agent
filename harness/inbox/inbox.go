package inbox

import (
	"context"
	"encoding/json/jsontext"
)

type ID string

type InputKind string

const (
)

type Input struct {
	ID      ID
	Kind    InputKind
	Payload jsontext.Value `json:",omitzero"`
}

}


const (
)

}

type Writer interface {
}
