package primitives



type TimerRequest struct {
	Source        SourceID
	CorrelationID CorrelationID
	Deadline      time.Time
}

type TimerResult struct {
	Deadline time.Time
	FiredAt  time.Time
}

