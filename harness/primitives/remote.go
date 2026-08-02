package primitives


const (
	PrimitiveEventRemoteResponseStarted PrimitiveEventType = "remote.response_started"
	PrimitiveEventRemoteOutput          PrimitiveEventType = "remote.output"
	PrimitiveEventRemoteStreamFailed    PrimitiveEventType = "remote.stream_failed"
	PrimitiveEventRemoteCompleted       PrimitiveEventType = "remote.completed"
)

type RemoteRequest struct {
}

func DefaultRemoteRequest(source SourceID, correlationID CorrelationID, url string) RemoteRequest {
	return RemoteRequest{
		RetryPolicy: RemoteRetryPolicy{
			InitialBackoff: 2 * time.Second,
			MaxBackoff:     30 * time.Second,
		},
	}
}

type RemoteRetryPolicy struct {
}

type RemoteResponseStartedResult struct {
	StatusCode int
	Headers    map[string][]string
}

type RemoteOutputResult struct {
}

type RemoteStreamFailureResult struct {
}

}
