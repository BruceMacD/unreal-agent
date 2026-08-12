package contextbuilder

import (
	"github.com/unreallabsai/unreal-agent/harness/llm"
)

type ChangeKind string

const (
	ChangeOmitted   ChangeKind = "omitted"
	ChangeTruncated ChangeKind = "truncated"
	ChangeCompacted ChangeKind = "compacted"
)

type Change struct {
	Kind   ChangeKind
	Source string
	Reason string
}

type Report struct {
	Changes []Change
}

type Result struct {
	Request llm.Request
	Report  Report
}

type Builder interface {
}
