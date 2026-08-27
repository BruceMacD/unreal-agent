package operation

import (
	"encoding/json/jsontext"
	"time"
)

const (
	TypeRemoteJob    Type    = "remote_job"
)

type RemoteJobPlanType string

type RemoteJobPlanVersion uint32

type RemoteJobPlan struct {
	Type    RemoteJobPlanType
	Version RemoteJobPlanVersion
	Data    jsontext.Value
}

type RemoteJobState struct {
	Plan             RemoteJobPlan
}
