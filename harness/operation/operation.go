// Package operation defines durable operation values and their actor runtime.
package operation

import (
)

type Type string

type Version uint32

type ID string

type Status string

const (
	StatusReady     Status = "ready"
	StatusAwaiting  Status = "awaiting"
	StatusCanceling Status = "canceling"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

type Spec struct {
}

type Operation struct {
}

type Manager interface {
	Add(Operation) error
	Cancel(ID, string) error
	Updates() <-chan Operation
}
