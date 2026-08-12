package operation

import (
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
}

type RemoteJobState struct {
	Plan             RemoteJobPlan
}
