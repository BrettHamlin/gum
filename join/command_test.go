package join

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRunSeparatorOutput(t *testing.T) {
	//harness:criterion=c-horizontal-default-no-separator,c-vertical-default-no-separator,c-horizontal-with-separator-two-operands,c-vertical-with-separator-two-operands,c-horizontal-with-separator-three-operands,c-vertical-with-separator-three-operands,c-horizontal-single-operand-no-change,c-vertical-single-operand-no-change,c-horizontal-separator-multichar,c-horizontal-separator-whitespace
	for name, tt := range map[string]struct {
		options  Options
		expected string
		criteria string
	}{
		"default horizontal": {
			options:  Options{Text: []string{"a", "b"}, Align: "left"},
			expected: "ab",
			criteria: "c-horizontal-default-no-separator,c-separator-empty-string-default",
		},
		"default vertical": {
			options:  Options{Text: []string{"a", "b"}, Align: "left", Vertical: true},
			expected: "a\nb",
			criteria: "c-vertical-default-no-separator",
		},
		"horizontal separator two operands": {
			options:  Options{Text: []string{"a", "b"}, Align: "left", Separator: ","},
			expected: "a,b",
			criteria: "c-horizontal-with-separator-two-operands",
		},
		"vertical separator two operands": {
			options:  Options{Text: []string{"a", "b"}, Align: "left", Vertical: true, Separator: "---"},
			expected: "a\n---\nb",
			criteria: "c-vertical-with-separator-two-operands",
		},
		"horizontal separator three operands": {
			options:  Options{Text: []string{"a", "b", "c"}, Align: "left", Separator: ","},
			expected: "a,b,c",
			criteria: "c-horizontal-with-separator-three-operands",
		},
		"vertical separator three operands": {
			options:  Options{Text: []string{"a", "b", "c"}, Align: "left", Vertical: true, Separator: "---"},
			expected: "a\n---\nb\n---\nc",
			criteria: "c-vertical-with-separator-three-operands",
		},
		"horizontal separator single operand": {
			options:  Options{Text: []string{"a"}, Align: "left", Separator: ","},
			expected: "a",
			criteria: "c-horizontal-single-operand-no-change",
		},
		"vertical separator single operand": {
			options:  Options{Text: []string{"a"}, Align: "left", Vertical: true, Separator: "---"},
			expected: "a",
			criteria: "c-vertical-single-operand-no-change",
		},
		"horizontal multichar separator": {
			options:  Options{Text: []string{"a", "b", "c"}, Align: "left", Separator: " | "},
			expected: "a | b | c",
			criteria: "c-horizontal-separator-multichar",
		},
		"horizontal whitespace separator": {
			options:  Options{Text: []string{"a", "b"}, Align: "left", Separator: " "},
			expected: "a b",
			criteria: "c-horizontal-separator-whitespace",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Logf("harness:criterion=%s", tt.criteria)

			out := runJoin(t, tt.options)
			if out != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, out)
			}
		})
	}
}

func TestSeparatorDefaultsToEmptyString(t *testing.T) {
	//harness:criterion=c-separator-empty-string-default
	var options Options
	if options.Separator != "" {
		t.Fatalf("expected zero-value separator to be empty, got %q", options.Separator)
	}

	defaultOut := runJoin(t, Options{Text: []string{"a", "b"}, Align: "left"})
	explicitEmptyOut := runJoin(t, Options{Text: []string{"a", "b"}, Align: "left", Separator: ""})
	if defaultOut != explicitEmptyOut {
		t.Fatalf("expected default and explicit empty separator output to match byte-for-byte: %q != %q", defaultOut, explicitEmptyOut)
	}
}

func TestSeparatorDoesNotAffectOperandBlockDimensions(t *testing.T) {
	//harness:criterion=c-separator-does-not-affect-lipgloss-block-dimensions
	out := strings.TrimSuffix(runJoin(t, Options{Text: []string{"hello", "world"}, Align: "left", Separator: "|"}), "\n")
	parts := strings.Split(out, "|")
	if len(parts) != 2 {
		t.Fatalf("expected two operand blocks split by separator, got %q", out)
	}

	for i, want := range []string{"hello", "world"} {
		if parts[i] != want {
			t.Fatalf("expected operand block %d to remain %q, got %q", i, want, parts[i])
		}
		if lipgloss.Width(parts[i]) != lipgloss.Width(want) {
			t.Fatalf("expected operand block %d width %d, got %d", i, lipgloss.Width(want), lipgloss.Width(parts[i]))
		}
		if lipgloss.Height(parts[i]) != lipgloss.Height(want) {
			t.Fatalf("expected operand block %d height %d, got %d", i, lipgloss.Height(want), lipgloss.Height(parts[i]))
		}
	}
}

func TestSeparatorFlagIsExposed(t *testing.T) {
	//harness:criterion=c-separator-flag-exists
	field, ok := reflect.TypeOf(Options{}).FieldByName("Separator")
	if !ok {
		t.Fatal("expected Options to expose a Separator field")
	}
	if field.Type.Kind() != reflect.String {
		t.Fatalf("expected Separator to be a string, got %s", field.Type)
	}

	source := readRepoFile(t, "join", "options.go")
	var separatorLine string
	for _, line := range strings.Split(source, "\n") {
		if strings.Contains(line, "Separator") && strings.Contains(line, "string") {
			separatorLine = line
			break
		}
	}
	if !strings.Contains(separatorLine, "--separator") {
		t.Fatalf("expected Separator field line to include the --separator Kong flag/help text, got %q", separatorLine)
	}
}

func TestReadmeDocumentsSeparator(t *testing.T) {
	//harness:criterion=c-readme-documents-separator
	readme := readRepoFile(t, "README.md")
	if !strings.Contains(readme, "--separator") {
		t.Fatal("expected README.md to document --separator")
	}
	if !strings.Contains(readme, "gum join --separator") {
		t.Fatal("expected README.md to include a gum join --separator usage example")
	}
}

func TestUnknownSeparatorFlagRejected(t *testing.T) {
	//harness:criterion=c-unknown-flag-rejected
	cmd := exec.Command("go", "run", ".", "join", "--separatr", ",")
	cmd.Dir = filepath.Join("..")

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected misspelled flag to fail, got success with output %q", out)
	}

	message := string(out)
	if !strings.Contains(message, "--separatr") &&
		!strings.Contains(strings.ToLower(message), "unknown flag") &&
		!strings.Contains(strings.ToLower(message), "unexpected flag") {
		t.Fatalf("expected error to reference unknown flag, got %q", message)
	}
}

func runJoin(t *testing.T, options Options) string {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer

	runErr := options.Run()
	closeErr := writer.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}

	return buf.String()
}

func readRepoFile(t *testing.T, elems ...string) string {
	t.Helper()

	path := filepath.Join(append([]string{".."}, elems...)...)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}
