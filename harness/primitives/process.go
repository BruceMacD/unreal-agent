package primitives


const (
	PrimitiveEventProcessStarted      PrimitiveEventType = "process.started"
	PrimitiveEventProcessOutput       PrimitiveEventType = "process.output"
	PrimitiveEventProcessStreamFailed PrimitiveEventType = "process.stream_failed"
	PrimitiveEventProcessExited       PrimitiveEventType = "process.exited"



)

type ProcessStartRequest struct {
	Source        SourceID
	CorrelationID CorrelationID
}

type ProcessStartedResult struct {
	PID int
}

type ProcessStream int

const (
	ProcessStdout ProcessStream = 1
	ProcessStderr ProcessStream = 2
)

type ProcessOutputResult struct {
	Stream ProcessStream
	Offset int64
	Data   []byte
}

type ProcessStreamFailureResult struct {
	Stream ProcessStream
	Error  string
}

type ProcessExitResult struct {
	ExitCode int
	Signal   syscall.Signal
}

	CorrelationID CorrelationID
}

}

	Count int
}


type ProcessSignalRequest struct {
}
