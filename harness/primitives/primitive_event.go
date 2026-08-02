package primitives

type PrimitiveEventType string

type SourceID string

type CorrelationID string


type PrimitiveEvent struct {
	Type          PrimitiveEventType
	Source        SourceID
	CorrelationID CorrelationID
	Result        any
}

type PrimitiveFailureResult struct {
	Error string
}
