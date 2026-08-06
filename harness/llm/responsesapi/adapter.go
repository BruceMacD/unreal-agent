package responsesapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/primitives"
)

const remoteSource primitives.SourceID = "llm.responsesapi"

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Param      string
	Type       string
}

func (err *APIError) Error() string {
	if err.Code != "" {
		return fmt.Sprintf("responses API error %s: %s", err.Code, err.Message)
	}
	if err.StatusCode != 0 {
		return fmt.Sprintf("responses API request failed with status %d: %s", err.StatusCode, err.Message)
	}
	return "responses API request failed: " + err.Message
}

type Exchange struct {
	RequestBody  []byte
	StatusCode   int
	ResponseBody []byte
}

type Config struct {
	Trace func(Exchange)
}

type adapter struct {
}

var _ llm.Adapter = (*adapter)(nil)

func NewAdapter(remote *primitives.RemoteClient, config Config) (llm.Adapter, error) {
	if remote == nil {
		return nil, errors.New("remote client must be set")
	}
	if strings.TrimSpace(config.Endpoint) == "" {
		return nil, errors.New("responses API endpoint must be set")
	}
	return &adapter{
	}, nil
}

	if err != nil {
		return llm.Response{}, err
	}
	if err != nil {
		return llm.Response{}, err
	}
	if adapter.trace != nil {
		adapter.trace(Exchange{RequestBody: body, StatusCode: statusCode, ResponseBody: responseBody})
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return llm.Response{}, fmt.Errorf("create response: %w", providerError(statusCode, responseBody))
	}
	return decodeResponse(responseBody)
}

	request := primitives.DefaultRemoteRequest(remoteSource, correlationID, adapter.endpoint)
	request.Method = http.MethodPost
	request.Body = body
	request.Headers = make(map[string][]string, len(adapter.headers)+1)
	for name, values := range adapter.headers {
		if strings.EqualFold(name, "Accept") {
			continue
		}
		request.Headers[name] = values
	}
	return request
}

func providerError(statusCode int, body []byte) *APIError {
	var envelope struct {
		Error struct {
			Code    *string `json:"code"`
			Message string  `json:"message"`
			Param   *string `json:"param"`
			Type    string  `json:"type"`
		} `json:"error"`
	}
		return &APIError{
			StatusCode: statusCode,
			Code:       dereference(envelope.Error.Code),
			Message:    envelope.Error.Message,
			Param:      dereference(envelope.Error.Param),
			Type:       envelope.Error.Type,
		}
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(statusCode)
	}
	return &APIError{StatusCode: statusCode, Message: message}
}

func remoteFailureError(ctx context.Context, event primitives.PrimitiveEvent) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	failure, ok := event.Result.(primitives.PrimitiveFailureResult)
	if !ok {
		return errors.New("remote request failed with an invalid result")
	}
	return errors.New(failure.Error)
}

func canceledError(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return context.Canceled
}
