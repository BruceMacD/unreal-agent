package agentrunner

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"uuid"

	"github.com/unreallabsai/unreal-agent/harness/contextbuilder"
	"github.com/unreallabsai/unreal-agent/harness/coordinator"
	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore/localfile"
	"github.com/unreallabsai/unreal-agent/harness/tool"
	"github.com/unreallabsai/unreal-agent/harness/tool/bash"
)

const (
)

const defaultSystemPrompt = `You are an AI agent running inside an isolated sandbox container.

## Guidelines
- Save output files to the workspace root.
- For large datasets, inspect a sample first before processing everything.
`

type Client interface {
	llm.Adapter
	Close() error
}

type Provider struct {
}

}

	Role      string  `json:"role"`
	Content   string  `json:"content"`
	MessageID *string `json:"message_id"`
}

type environmentChange struct {
	name    string
	value   string
	present bool
}

type environmentScope struct {
	changes []environmentChange
}

type errorEvent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type sessionObserver struct {
	sessionID session.ID
	output    io.Writer
	cancel    context.CancelFunc

}

func RunMain(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	environ func() []string,
	input io.Reader,
	output io.Writer,
	stderr io.Writer,
) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if context.Cause(ctx) != nil {
		return 130
	}
	encoded, encodeErr := json.Marshal(errorEvent{Type: "error", Message: err.Error()})
	if encodeErr != nil {
		err = errors.Join(err, fmt.Errorf("encode error event: %w", encodeErr))
	} else if _, writeErr := fmt.Fprintf(output, "%s\n", encoded); writeErr != nil {
		err = errors.Join(err, fmt.Errorf("write error event: %w", writeErr))
	}
		return 1
	}
	return 1
}

func Run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	environ func() []string,
	input io.Reader,
	output io.Writer,
	flagOutput io.Writer,
) (runErr error) {
	flags.SetOutput(flagOutput)
	sessionDirectory := flags.String("session-directory", defaultSessionDirectory, "directory containing session files")
	workspaceDirectory := flags.String("workspace", ".", "agent workspace and Bash working directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}

	if err != nil {
		return err
	}
	messages, err := validateRequest(parsed)
	if err != nil {
		return err
	}
	workspace, err := filepath.Abs(strings.TrimSpace(*workspaceDirectory))
	if err != nil {
		return fmt.Errorf("resolve workspace: %w", err)
	}
	workspaceInfo, err := os.Stat(workspace)
	if err != nil {
		return fmt.Errorf("inspect workspace: %w", err)
	}
	if !workspaceInfo.IsDir() {
		return fmt.Errorf("workspace %q is not a directory", workspace)
	}
	environment, err := loadDotEnv(filepath.Join(workspace, ".env"))
	if err != nil {
		return err
	}
	defer func() {
		if err := environment.Close(); err != nil {
			runErr = errors.Join(runErr, err)
		}
	}()
	providerName := strings.TrimSpace(getenv(llmProviderEnvironment))
	if providerName == "" {
		providerName = defaultProvider
	}
	if err != nil {
		return err
	}
	configuredBaseURL := strings.TrimSpace(getenv(llmBaseURLEnvironment))
	if configuredBaseURL == "" {
		configuredBaseURL = selected.BaseURL
	}

	model := strings.TrimSpace(parsed.Model)
	if model == "" {
		model = strings.TrimSpace(getenv(llmModelEnvironment))
	}
	if model == "" {
		model = selected.DefaultModel
	}
	if model == "" {
		return fmt.Errorf("model must be set in the request or %s", llmModelEnvironment)
	}
	}
	if err != nil {
		return fmt.Errorf("create %s client: %w", selected.Name, err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close %s client: %w", selected.Name, err))
		}
	}()

	storeDirectory, err := filepath.Abs(strings.TrimSpace(*sessionDirectory))
	if err != nil {
		return fmt.Errorf("resolve session directory: %w", err)
	}
	store, err := localfile.New(storeDirectory)
	if err != nil {
		return fmt.Errorf("open session store: %w", err)
	}
	sessionID, restored, err := openSession(ctx, store, parsed.SessionID)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close session log: %w", err))
		}
	}()
	observedOutput := io.MultiWriter(logFile, output)

	runContext, cancel := context.WithCancel(ctx)
	defer cancel()
	operationDirectory := filepath.Join(storeDirectory, "operations", string(sessionID))
	if err := os.MkdirAll(operationDirectory, 0o700); err != nil {
		return fmt.Errorf("create operation directory: %w", err)
	}
	shell := strings.TrimSpace(getenv("SHELL"))
	if shell == "" {
		shell = "/bin/sh"
	}
		if _, err := fmt.Fprintf(flagOutput, "skill error> %s\n", skillErr); err != nil {
			return fmt.Errorf("write skill error: %w", err)
		}
	}

	inputs, err := inbox.New(runContext, restored.ExternalInputIDs)
	if err != nil {
		return fmt.Errorf("open inbox: %w", err)
	}
	for index, message := range messages {
		payload, err := json.Marshal(message.Content)
		if err != nil {
			return fmt.Errorf("encode message %d: %w", index, err)
		}
		var messageID inbox.ID
		if message.MessageID != nil {
			messageID = inbox.ID(strings.TrimSpace(*message.MessageID))
		} else {
			messageID = inbox.ID(uuid.New().String())
		}
		if err := inputs.Submit(runContext, inbox.Input{
			ID: messageID, Kind: inbox.InputExternal, Payload: payload,
		}); err != nil {
			return fmt.Errorf("submit message %d: %w", index, err)
		}
	}

	builder := contextbuilder.NewBuilder(registry.Skills()...)
	builder.SetModel(llm.Model{
		ID:              model,
		ReasoningEffort: reasoningEffort(parsed.ThinkingLevel),
	})
	systemPrompt := defaultSystemPrompt
	if parsed.SystemPrompt != nil {
		systemPrompt = *parsed.SystemPrompt
	}
	builder.SetSystemPrompt(systemPrompt)
	for _, definition := range registry.StaticDefinitions() {
	}

	observer := &sessionObserver{
		sessionID: sessionID,
		output:    observedOutput,
		cancel:    cancel,
	}
	observerID := store.AddObserver(observer.Observe)
	defer store.RemoveObserver(observerID)
	current := coordinator.New(coordinator.Dependencies{
	})
	coordinatorErr := current.Run(runContext)
	if observerErr := observer.Err(); observerErr != nil {
		return observerErr
	}
	if coordinatorErr != nil {
		return fmt.Errorf("run coordinator: %w", coordinatorErr)
	}
}

func selectProvider(providers []Provider, name string) (Provider, error) {
	for _, provider := range providers {
		if provider.Name == name {
			if provider.NewClient == nil {
				return Provider{}, fmt.Errorf("provider %q has no client factory", name)
			}
			return provider, nil
		}
	}
	names := make([]string, 0, len(providers))
	for _, provider := range providers {
		names = append(names, provider.Name)
	}
	return Provider{}, fmt.Errorf("unsupported provider %q; available providers: %s", name, strings.Join(names, ", "))
}

	raw, err := io.ReadAll(input)
	if err != nil {
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
	}
	}
}

	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	path := filepath.Join(directory, now.UTC().Format("20060102-150405")+".jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open session log: %w", err)
	}
	return file, nil
}

func loadDotEnv(path string) (*environmentScope, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &environmentScope{}, nil
		}
		return nil, fmt.Errorf("read environment file: %w", err)
	}
	values := make(map[string]string)
	for line := range strings.Lines(string(encoded)) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, exists := strings.Cut(line, "=")
		if !exists {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		values[name] = strings.TrimSpace(value)
	}
	scope := &environmentScope{}
	for name, value := range values {
		_, present := os.LookupEnv(name)
		if present && name != "SANDBOX_EGRESS_PROXY" {
			continue
		}
		if err := scope.set(name, value); err != nil {
			return nil, errors.Join(err, scope.Close())
		}
	}
	if proxy := os.Getenv("SANDBOX_EGRESS_PROXY"); proxy != "" {
		if err := scope.set("HTTPS_PROXY", proxy); err != nil {
			return nil, errors.Join(err, scope.Close())
		}
	}
	return scope, nil
}

func (scope *environmentScope) set(name, value string) error {
	previous, present := os.LookupEnv(name)
	if err := os.Setenv(name, value); err != nil {
		return fmt.Errorf("set environment variable %q: %w", name, err)
	}
	scope.changes = append(scope.changes, environmentChange{
		name: name, value: previous, present: present,
	})
	return nil
}

func (scope *environmentScope) Close() error {
	var closeErr error
	for _, change := range slices.Backward(scope.changes) {
		var err error
		if change.present {
			err = os.Setenv(change.name, change.value)
		} else {
			err = os.Unsetenv(change.name)
		}
		if err != nil {
			closeErr = errors.Join(
				closeErr,
				fmt.Errorf("restore environment variable %q: %w", change.name, err),
			)
		}
	}
	scope.changes = nil
	return closeErr
}

	if parsed.SessionID != nil && strings.TrimSpace(*parsed.SessionID) == "" {
		return nil, errors.New("session_id must not be empty")
	}
	if parsed.ThinkingLevel != "" {
		switch parsed.ThinkingLevel {
		default:
		}
	}
	for _, name := range append(parsed.ExtraAllowedTools, parsed.DisallowedTools...) {
		if strings.TrimSpace(name) == "" {
			return nil, errors.New("tool names must not be empty")
		}
	}

	if parsed.Messages == nil {
		if parsed.Prompt == nil {
			return nil, errors.New("messages must be set")
		}
	}
	if len(parsed.Messages) == 0 {
		return nil, errors.New("messages must not be empty")
	}
	for index, message := range parsed.Messages {
		if message.Role != "" && message.Role != "user" {
			return nil, fmt.Errorf("messages[%d].role must be user", index)
		}
		if message.MessageID != nil && strings.TrimSpace(*message.MessageID) == "" {
			return nil, fmt.Errorf("messages[%d].message_id must not be empty", index)
		}
		if message.MessageID != nil {
			if _, err := uuid.Parse(strings.TrimSpace(*message.MessageID)); err != nil {
				return nil, fmt.Errorf("messages[%d].message_id must be a UUID", index)
			}
		}
	}
	return parsed.Messages, nil
}

func reasoningEffort(level string) llm.ReasoningEffort {
	switch level {
	case "low":
		return llm.ReasoningEffortLow
	case "medium":
		return llm.ReasoningEffortMedium
	default:
		return llm.ReasoningEffortHigh
	}
}

func openSession(
	ctx context.Context,
	store *localfile.Store,
	requested *string,
) (session.ID, sessionstore.ResumeState, error) {
	id := session.ID(uuid.New().String())
	if requested != nil {
		id = session.ID(strings.TrimSpace(*requested))
	}
	restored, err := store.Resume(ctx, id)
	if err == nil {
		return id, restored, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", sessionstore.ResumeState{}, fmt.Errorf("open session %q: %w", id, err)
	}
	snapshot, err := store.Create(ctx, id)
	if err != nil {
		return "", sessionstore.ResumeState{}, fmt.Errorf("create session %q: %w", id, err)
	}
	return id, sessionstore.ResumeState{Snapshot: snapshot}, nil
}

func (observer *sessionObserver) Observe(sessionID session.ID, item sessionstore.Item) {
	if sessionID != observer.sessionID {
		return
	}
	observer.mu.Lock()
	defer observer.mu.Unlock()
	if observer.err != nil {
		return
	}
	if err := writeSessionItem(observer.output, item); err != nil {
		observer.fail(err)
		return
	}
}

func writeSessionItem(output io.Writer, item sessionstore.Item) error {
	encoded, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("encode session item %d: %w", item.Sequence, err)
	}
	if _, err := fmt.Fprintf(output, "%s\n", encoded); err != nil {
		return fmt.Errorf("write session item %d: %w", item.Sequence, err)
	}
	return nil
}

func (observer *sessionObserver) fail(err error) {
	observer.err = err
	observer.cancel()
}

func (observer *sessionObserver) Err() error {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	return observer.err
}
