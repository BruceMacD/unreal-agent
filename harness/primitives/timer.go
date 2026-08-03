package primitives

import (
	"context"
	"time"
)


type TimerRequest struct {
	Source        SourceID
	CorrelationID CorrelationID
	Deadline      time.Time
}

type TimerResult struct {
	Deadline time.Time
	FiredAt  time.Time
}

	}


	}
}

func timerFired(request TimerRequest, firedAt time.Time) PrimitiveEvent {
	return PrimitiveEvent{
		Type:          PrimitiveEventTimerFired,
		Source:        request.Source,
		CorrelationID: request.CorrelationID,
		Result: TimerResult{
			Deadline: request.Deadline,
			FiredAt:  firedAt,
		},
	}
}

func timerCanceled(request TimerRequest) PrimitiveEvent {
	return primitiveCanceled(request.Source, request.CorrelationID)
}
