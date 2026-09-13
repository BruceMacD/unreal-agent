package operation_test

import (
	"encoding/json/v2"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/primitives"
)

func TestBoundOutputPreservesHeadAndTail(t *testing.T) {
	for _, test := range []struct {
		name, text, want string
		limit            int
		truncated        bool
	}{
		{"empty", "", "", 1, false},
		{"exact", "界é🙂", "界é🙂", 3, false},
		{"even", "abcdefghij", "abc...4 bytes truncated...hij", 6, true},
		{"odd", "abcdefghij", "ab...5 bytes truncated...hij", 5, true},
		{"one", "ab", "...1 bytes truncated...b", 1, true},
		{"two", "abc", "a...1 bytes truncated...c", 2, true},
		{"unicode", "界éab🙂好", "界...4 bytes truncated...🙂好", 3, true},
		{"ellipsis", "…abc…", "…...3 bytes truncated...…", 2, true},
		{"invalid UTF8", "a\xff\xfeb", "a��b", 4, false},
		{"invalid UTF8 truncated", "a\xffbc\xfed", "a�...2 bytes truncated...�d", 4, true},
		{"zero", "abc", "...3 bytes truncated...", 0, true},
		{"negative", "abc", "...3 bytes truncated...", -1, true},
		{"maximum", "HEAD" + strings.Repeat("x", operation.MaxOutputLength) + "TAIL", "HEAD" + strings.Repeat("x", 499996) + "...8 bytes truncated..." + strings.Repeat("x", 499996) + "TAIL", operation.MaxOutputLength, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, truncated := operation.BoundOutput(test.text, test.limit)
			if got != test.want || truncated != test.truncated {
				t.Fatalf("output = %.100q, truncated = %t; want %.100q, truncated %t", got, truncated, test.want, test.truncated)
			}
			if !utf8.ValidString(got) {
				t.Fatal("output is invalid UTF-8")
			}
			retained := regexp.MustCompile(`\.\.\.[0-9]+ bytes truncated\.\.\.`).ReplaceAllString(got, "")
			}
		})
	}
}

	marker := regexp.MustCompile(`\.\.\.([0-9]+) bytes truncated\.\.\.`)
	for _, text := range []string{
		"\"\\/\b\f\n\r\t\x00\x1f",
		`\u0000\\\"abcd\n`,
		"é界🙂…<>&\u2028\u2029",
	} {
			got, truncated := operation.BoundOutput(text, limit)
			if !truncated {
					t.Fatalf("limit %d: unexpected untruncated output %q", limit, got)
				}
				continue
			}
			match := marker.FindStringSubmatchIndex(got)
			if match == nil {
				t.Fatalf("missing marker in %q", got)
			}
			head, tail := got[:match[0]], got[match[1]:]
			if !strings.HasPrefix(text, head) || !strings.HasSuffix(text, tail) {
				t.Fatalf("limit %d: %q is not a head and tail of %q", limit, got, text)
			}
			if got[match[2]:match[3]] != fmt.Sprint(len(text)-len(head)-len(tail)) {
				t.Fatalf("incorrect original byte count in %q", got)
			}
			for i, part := range []string{head, tail} {
				budget := limit / 2
				if i == 1 {
					budget = limit - budget
				}
				}
			}
		}
	}
}

func TestShellRejectsInvalidOutputLimits(t *testing.T) {
	for _, limit := range []int{0, -1, operation.MaxOutputLength + 1, 1_000_000_000} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			if _, err := operation.NewShellSpec(operation.ShellInput{Shell: "/bin/sh"}, t.TempDir(), limit); err == nil {
				t.Fatal("shell spec accepted an invalid limit")
			}
			shell, err := operation.NewShellSpec(operation.ShellInput{Shell: "/bin/sh"}, t.TempDir(), 1)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := operation.DecodeShellState(operation.Operation{Type: shell.Type, Version: shell.Version, State: shell.State, MaxOutputLength: limit}); err == nil {
				t.Fatal("shell operation accepted an invalid limit")
			}
		})
	}
}

func TestShellAcceptsMaximumOutputLength(t *testing.T) {
	shell, err := operation.NewShellSpec(operation.ShellInput{Shell: "/bin/sh"}, t.TempDir(), operation.MaxOutputLength)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := operation.DecodeShellState(operation.Operation{Type: shell.Type, Version: shell.Version, State: shell.State, MaxOutputLength: shell.MaxOutputLength}); err != nil {
		t.Fatal(err)
	}
}

func TestShellPreparesOutputBeforePublishingCompletion(t *testing.T) {
	for _, test := range []struct {
		name      string
		out       []byte
		size      int64
		limit     int
		want      string
		truncated bool
	}{
		{"default", []byte(strings.Repeat("界", 40001)), 120003, operation.DefaultMaxOutputLength, strings.Repeat("界", 20000) + "...3 bytes truncated; complete output in {path}..." + strings.Repeat("界", 20000), true},
		{"custom", []byte("界éab"), 7, 2, "界...3 bytes truncated; complete output in {path}...b", true},
		{"above default", []byte(strings.Repeat("é", 50000)), 100000, 50000, strings.Repeat("é", 50000), false},
		{"invalid UTF8", []byte{'a', 0xff, 'b'}, 3, 3, "a�b", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			exitCode := 0
			base := t.TempDir()
			encoded, err := json.Marshal(operation.ShellState{
				Input: operation.ShellInput{Shell: "/bin/sh"}, BaseDirectory: base, Phase: operation.ShellPhaseReadErr,

				InlineOut: test.out, OutSize: test.size,
				PendingExitCode: &exitCode,
			})
			if err != nil {
				t.Fatal(err)
			}
			current := operation.Operation{ID: "shell", Type: operation.TypeShell, Version: operation.VersionShell, Status: operation.StatusAwaiting, State: encoded, MaxOutputLength: test.limit}
			shell, err := operation.NewShell(current)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := shell.Handle(&primitives.PrimitiveEvent{
				Source: "shell", CorrelationID: "read_err", Type: primitives.PrimitiveEventIOReadOutput,
				Result: primitives.IOReadOutputResult{Offset: 0, Data: test.out},
			}); err != nil {
				t.Fatal(err)
			}
			step, err := shell.Handle(&primitives.PrimitiveEvent{
				Source: "shell", CorrelationID: "read_err", Type: primitives.PrimitiveEventIOReadCompleted,
				Result: primitives.IOReadCompletedResult{Size: test.size},
			})
			if err != nil {
				t.Fatal(err)
			}
			var state operation.ShellState
			if err := json.Unmarshal(step.Operation.State, &state); err != nil {
				t.Fatal(err)
			}
			wantOut := strings.ReplaceAll(test.want, "{path}", filepath.Join(base, "shell", "out"))
			wantErr := strings.ReplaceAll(test.want, "{path}", filepath.Join(base, "shell", "err"))
			if step.Operation.Status != operation.StatusCompleted || state.Result == nil || state.Result.Out != wantOut || state.Result.Err != wantErr || state.OutTruncated != test.truncated || state.ErrTruncated != test.truncated {
				t.Fatalf("state = %#v, result = %#v", state, state.Result)
			}
			if state.OutPath != filepath.Join(base, "shell", "out") || state.ErrPath != filepath.Join(base, "shell", "err") {
				t.Fatalf("capture paths = %q, %q", state.OutPath, state.ErrPath)
			}
		})
	}
}

func TestShellBoundsOperationErrors(t *testing.T) {
	spec, err := operation.NewShellSpec(operation.ShellInput{Shell: "/bin/sh"}, t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}
	current := operation.Operation{ID: "shell", Type: spec.Type, Version: spec.Version, Status: operation.StatusAwaiting, State: spec.State, MaxOutputLength: 3}
	step, err := advanceShellOnce(t, current, &primitives.PrimitiveEvent{
		Source: "shell", Type: primitives.PrimitiveEventFailed,
		Result: primitives.PrimitiveFailureResult{Error: "éééé"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var state operation.ShellState
	if err := json.Unmarshal(step.Operation.State, &state); err != nil {
		t.Fatal(err)
	}
	if state.TerminalError != "é...2 bytes truncated...éé" || !state.ErrorTruncated || !state.OutTruncated || !state.ErrTruncated {
		t.Fatalf("state = %#v", state)
	}
}

func TestShellReplacesPreviouslyTruncatedErrors(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		state := operation.ShellState{
			Input: operation.ShellInput{Shell: "/bin/sh"}, BaseDirectory: t.TempDir(),
			TerminalError: "o...10 bytes truncated...ld", ErrorTruncated: true,
		}
		encoded, err := json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		current := operation.Operation{ID: "shell", Type: operation.TypeShell, Version: operation.VersionShell, Status: operation.StatusAwaiting, State: encoded, MaxOutputLength: 3}
		event := &primitives.PrimitiveEvent{Source: "shell", Type: primitives.PrimitiveEventFailed, Result: primitives.PrimitiveFailureResult{Error: "abcdef"}}
		want := "a...3 bytes truncated...ef"
		if cancel {
			current.Status, event = operation.StatusCanceling, nil
			want = "s...21 bytes truncated...ed"
		}
		step, err := advanceShellOnce(t, current, event)
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			state, err = operation.DecodeShellState(*step.Operation)
			if err != nil {
				t.Fatal(err)
			}
			if state.TerminalError != want || !state.ErrorTruncated {
				t.Fatalf("cancel %t: error = %q, want %q", cancel, state.TerminalError, want)
			}
		}
	}
}
