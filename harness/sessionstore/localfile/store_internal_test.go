package localfile

import (
	"encoding/json/jsontext"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
	"github.com/unreallabsai/unreal-agent/harness/tool"
)

func TestStoreLoadsGoldenLog(t *testing.T) {
	store, err := New("testdata")
	if err != nil {
		t.Fatal(err)
	}
	got, committedSize, err := store.readState(t.Context(), "golden-session")
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(store.sessionPath("golden-session"))
	if err != nil {
		t.Fatal(err)
	}
	if committedSize != int64(len(contents)) {
		t.Fatalf("committed size = %d, want %d", committedSize, len(contents))
	}

	createdAt := time.Date(2026, time.August, 27, 10, 0, 0, 0, time.UTC)
	recordedAt := time.Date(2026, time.August, 27, 10, 1, 0, 0, time.UTC)
	want := newStoredState("golden-session", createdAt)
	want.Snapshot = sessionstore.Snapshot{
		Session: session.Session{ID: "golden-session", CreatedAt: createdAt},
	}
	want.Items = []sessionstore.Item{
		{
			Sequence: 1, RecordedAt: recordedAt, Kind: sessionstore.ItemInput,
			Data: inbox.Input{
				ID: "input-1", Kind: inbox.InputExternal,
				Payload: jsontext.Value(`{"message":"hello"}`),
			},
		},
		{
			Sequence: 2, RecordedAt: recordedAt, Kind: sessionstore.ItemTurn,
		},
		{
			Sequence: 3, RecordedAt: recordedAt, Kind: sessionstore.ItemModelResponse,
			Data: sessionstore.ModelResponse{
				TurnID: "turn-1",
				Response: llm.Response{
					ID: "response-1", Stop: llm.StopComplete,
					Output: []llm.Item{
						{
							ProviderID: "message-1", Type: llm.ItemMessage,
							Data: llm.Message{
								Role: llm.RoleAssistant, Text: "working", Phase: "commentary",
							},
						},
						{
							ProviderID: "reasoning-1", Type: llm.ItemReasoning,
							Data: llm.Reasoning{
								Summary: []string{"inspect"},
								Raw:     jsontext.Value(`{"encrypted":"opaque"}`),
							},
						},
					},
					Usage: llm.Usage{
						InputTokens: 10, CachedInputTokens: 2, CacheWriteInputTokens: 1,
						OutputTokens: 4, ReasoningTokens: 3,
						Raw: jsontext.Value(`{"provider_total":14}`),
					},
				},
			},
		},
		{
			Sequence: 4, RecordedAt: recordedAt, Kind: sessionstore.ItemToolCallStatus,
			Data: sessionstore.ToolCallStatus{
				TurnID: "turn-1", CallID: "call-1",
				Status: tool.CallStatus{WaitingFor: []operation.ID{"operation-1"}},
				Operations: []operation.Operation{{
					ID: "operation-1", Type: "test", Version: 1, Status: operation.StatusReady,
					State:       jsontext.Value(`{"step":1}`),
					Idempotency: jsontext.Value(`{"key":"one"}`),
				}},
			},
		},
	}
	want.Operations = []operation.Operation{{
		ID: "operation-1", Type: "test", Version: 1, Status: operation.StatusAwaiting,
		State:       jsontext.Value(`{"step":2}`),
		Idempotency: jsontext.Value(`{"key":"one"}`),
	}}
	if !reflect.DeepEqual(got.Snapshot, want.Snapshot) ||
		!reflect.DeepEqual(got.Items, want.Items) ||
		!reflect.DeepEqual(got.Operations, want.Operations) {
		t.Fatalf("golden state = %#v\nwant %#v", got, want)
	}
}

func TestStoreLoadsGoldenForkLog(t *testing.T) {
	store, err := New("testdata")
	if err != nil {
		t.Fatal(err)
	}
	got, committedSize, err := store.readState(t.Context(), "golden-fork")
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(store.sessionPath("golden-fork"))
	if err != nil {
		t.Fatal(err)
	}
	if committedSize != int64(len(contents)) {
		t.Fatalf("committed size = %d, want %d", committedSize, len(contents))
	}

	createdAt := time.Date(2026, time.August, 27, 11, 0, 0, 0, time.UTC)
	recordedAt := time.Date(2026, time.August, 27, 11, 1, 0, 0, time.UTC)
	want := newStoredState("golden-fork", createdAt)
	want.Snapshot = sessionstore.Snapshot{
		Session: session.Session{ID: "golden-fork", CreatedAt: createdAt},
	}
	want.Items = []sessionstore.Item{
		{
			Sequence: 1, RecordedAt: recordedAt, Kind: sessionstore.ItemTurn,
		},
		{
			Sequence: 2, RecordedAt: recordedAt, Kind: sessionstore.ItemFork,
			Data: sessionstore.Fork{
				ParentID: "parent-session", PreviousTurnID: "parent-turn",
			},
		},
		{
			Sequence: 3, RecordedAt: recordedAt, Kind: sessionstore.ItemInput,
			Data: inbox.Input{
				ID: "child-input", Kind: inbox.InputExternal,
				Payload: jsontext.Value(`{"message":"child"}`),
			},
		},
		{
			Sequence: 4, RecordedAt: recordedAt, Kind: sessionstore.ItemTurn,
		},
		{
			Sequence: 5, RecordedAt: recordedAt, Kind: sessionstore.ItemModelResponse,
			Data: sessionstore.ModelResponse{
				TurnID: "child-turn-1",
				Response: llm.Response{
					ID: "response-1", Stop: llm.StopComplete,
					Output: []llm.Item{
						{
							ProviderID: "message-1", Type: llm.ItemMessage,
							Data: llm.Message{
								Role: llm.RoleAssistant, Text: "working", Phase: "commentary",
							},
						},
						{
							ProviderID: "call-1", Type: llm.ItemToolCall,
							Data: llm.ToolCall{CallID: "call-1", Name: "test", Arguments: `{}`},
						},
						{
							ProviderID: "reasoning-1", Type: llm.ItemReasoning,
							Data: llm.Reasoning{
								Summary: []string{"inspect"},
								Raw:     jsontext.Value(`{"encrypted":"opaque"}`),
							},
						},
					},
					Usage: llm.Usage{
						InputTokens: 10, CachedInputTokens: 2, CacheWriteInputTokens: 1,
						OutputTokens: 4, ReasoningTokens: 3,
						Raw: jsontext.Value(`{"provider_total":14}`),
					},
				},
			},
		},
		{
			Sequence: 6, RecordedAt: recordedAt, Kind: sessionstore.ItemTurn,
		},
		{
			Sequence: 7, RecordedAt: recordedAt, Kind: sessionstore.ItemModelResponse,
			Data: sessionstore.ModelResponse{
				TurnID: "child-turn-2",
				Response: llm.Response{
					ID: "response-2", Stop: llm.StopRefused,
					Failure: &llm.Failure{Code: "server_error", Message: "failed"},
				},
			},
		},
		{
			Sequence: 8, RecordedAt: recordedAt, Kind: sessionstore.ItemToolCallStatus,
			Data: sessionstore.ToolCallStatus{
				TurnID: "child-turn-1", CallID: "call-error",
				Status: tool.CallStatus{Error: "invalid arguments"},
			},
		},
	}
	if !reflect.DeepEqual(got.Snapshot, want.Snapshot) ||
		!reflect.DeepEqual(got.Items, want.Items) ||
		!reflect.DeepEqual(got.Operations, want.Operations) {
		t.Fatalf("golden state = %#v\nwant %#v", got, want)
	}
}

func TestStoreCachesWriteStateForCreatedAndLoadedSessions(t *testing.T) {
	directory := t.TempDir()
	store, err := New(directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(t.Context(), "session-1"); err != nil {
		t.Fatal(err)
	}
	created, createdSize, ok := store.getCachedWriteState("session-1")
	if !ok || created.Snapshot.Session.ID != "session-1" {
		t.Fatalf("created cache entry = %#v, present %t", created, ok)
	}
	contents, err := os.ReadFile(store.sessionPath("session-1"))
	if err != nil {
		t.Fatal(err)
	}
	if createdSize != int64(len(contents)) {
		t.Fatalf("cached size = %d, want %d", createdSize, len(contents))
	}

	reopened, err := New(directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := reopened.getCachedWriteState("session-1"); ok {
		t.Fatal("new store started with cached write state")
	}
	if _, err := reopened.Inspect(t.Context(), "session-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := reopened.getCachedWriteState("session-1"); ok {
		t.Fatal("read cached the session write state")
	}
		t.Fatal(err)
	}
	if _, _, ok := reopened.getCachedWriteState("session-1"); !ok {
		t.Fatal("first write did not cache the validated session head")
	}
}

func TestStoreLoadRejectsInvalidSessionLogs(t *testing.T) {
	t.Run("invalid completed record", func(t *testing.T) {
		directory := t.TempDir()
		store, err := New(directory)
		if err != nil {
			t.Fatal(err)
		}
		encoded := append(emptyLog(t), []byte("{\n")...)
		if err := os.WriteFile(store.sessionPath("session-1"), encoded, 0o600); err != nil {
			t.Fatal(err)
		}

		if _, err := store.Inspect(t.Context(), "session-1"); err == nil ||
			!strings.Contains(err.Error(), `read session "session-1": decode record 2`) {
			t.Fatalf("error = %v, want corrupt-log error", err)
		}
	})

	t.Run("filename and header disagree", func(t *testing.T) {
		directory := t.TempDir()
		store, err := New(directory)
		if err != nil {
			t.Fatal(err)
		}
		other := newStoredState("other", stateCreatedAt)
		encoded, err := encodeInitialLog(other.Snapshot.Session, other.Items)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(store.sessionPath("session-1"), encoded, 0o600); err != nil {
			t.Fatal(err)
		}

		if _, err := store.Inspect(t.Context(), "session-1"); err == nil ||
			!strings.Contains(err.Error(), `file contains session "other"`) {
			t.Fatalf("error = %v, want session mismatch", err)
		}
	})
}

func TestAppendFileReportsOpenFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "session.jsonl")
	if err := appendFile(path, 0, []byte("{}\n")); err == nil ||
		!strings.Contains(err.Error(), "open session log for append") {
		t.Fatalf("error = %v, want append-open failure", err)
	}
}

func TestPublishFileFailuresCleanTemporaryFiles(t *testing.T) {
	t.Run("replacement target is a directory", func(t *testing.T) {
		directory := t.TempDir()
		target := filepath.Join(directory, "target")
		if err := os.Mkdir(target, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, "entry"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := publishFile(directory, target, []byte("replacement")); err == nil {
			t.Fatal("replacement succeeded")
		}
		if _, err := os.Stat(filepath.Join(target, "entry")); err != nil {
			t.Fatalf("target changed: %v", err)
		}
		assertNoTemporaryFiles(t, directory)
	})

	t.Run("temporary directory is a file", func(t *testing.T) {
		directory := t.TempDir()
		file := filepath.Join(directory, "file")
		if err := os.WriteFile(file, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := publishFile(file, filepath.Join(file, "target"), nil); err == nil ||
			!strings.Contains(err.Error(), "create temporary session file") {
			t.Fatalf("error = %v, want temporary-file creation failure", err)
		}
	})
}

func TestPersistTemporaryReportsClosedFileFailures(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "closed-")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	err = persistTemporary(file, []byte("state"))
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("error = %v, want os.ErrClosed", err)
	}
	for _, want := range []string{"write temporary session file", "close temporary session file"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err, want)
		}
	}
}

func TestPersistTemporaryReportsSyncFailure(t *testing.T) {
	file, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = persistTemporary(file, []byte("state"))
	if err == nil {
		t.Skip("platform permits syncing the null device")
	}
	if !strings.Contains(err.Error(), "sync temporary session file") {
		t.Fatalf("error = %v, want temporary-file sync failure", err)
	}
}

func TestSyncDirectoryFileReportsClosedHandle(t *testing.T) {
	directory, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Close(); err != nil {
		t.Fatal(err)
	}

	err = syncDirectoryFile(directory)
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("error = %v, want os.ErrClosed", err)
	}
	for _, want := range []string{"sync session store directory", "close session store directory"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err, want)
		}
	}
}

func TestSyncDirectoryReportsOpenFailure(t *testing.T) {
	if err := syncDirectory(filepath.Join(t.TempDir(), "missing")); err == nil ||
		!strings.Contains(err.Error(), "open session store directory for sync") {
		t.Fatalf("error = %v, want directory-open failure", err)
	}
}

func TestNewRejectsInvalidDirectories(t *testing.T) {
	if _, err := New("  "); err == nil {
		t.Fatal("empty directory accepted")
	}
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(path); err == nil || !strings.Contains(err.Error(), "create session store directory") {
		t.Fatalf("error = %v, want directory-creation failure", err)
	}
}

func assertNoTemporaryFiles(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".session-") {
			t.Fatalf("temporary file remains: %q", entry.Name())
		}
	}
}
