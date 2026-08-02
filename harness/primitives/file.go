package primitives


const (
	PrimitiveEventIOCreateCompleted PrimitiveEventType = "io.create_completed"
	PrimitiveEventIOReadCompleted   PrimitiveEventType = "io.read_completed"
)

type IOReadRequest struct {
	Source        SourceID
	CorrelationID CorrelationID
	Path          string
	Offset        int64
	Count         int64
}

}

type IOCreateRequest struct {
	Source        SourceID
	CorrelationID CorrelationID
	Path          string
}

type IOCreateResult struct {
}

