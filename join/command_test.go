package join

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
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
	left := "hello\nthere"
	right := "world"

	out := runJoin(t, Options{Text: []string{left, right}, Align: "left", Separator: "|"})
	expected := lipgloss.JoinHorizontal(lipgloss.Left, left, "|", right)
	if out != expected {
		t.Fatalf("expected separator to be joined as its own block:\nwant %q\ngot  %q", expected, out)
	}

	lines := strings.Split(out, "\n")
	leftWidth := lipgloss.Width(left)
	leftHeight := lipgloss.Height(left)
	if len(lines) < leftHeight {
		t.Fatalf("expected at least %d rendered lines, got %d in %q", leftHeight, len(lines), out)
	}
	leftLines := make([]string, 0, leftHeight)
	for i := 0; i < leftHeight; i++ {
		if len(lines[i]) < leftWidth {
			t.Fatalf("line %d too short to contain left operand block: %q", i, lines[i])
		}
		leftLines = append(leftLines, lines[i][:leftWidth])
	}
	if got := strings.Join(leftLines, "\n"); got != left {
		t.Fatalf("expected left operand block to remain %q, got %q", left, got)
	}

	rightWidth := lipgloss.Width(right)
	rightHeight := lipgloss.Height(right)
	rightOffset := leftWidth + lipgloss.Width("|")
	rightLines := make([]string, 0, rightHeight)
	for i := 0; i < rightHeight; i++ {
		if len(lines[i]) < rightOffset+rightWidth {
			t.Fatalf("line %d too short to contain right operand block: %q", i, lines[i])
		}
		rightLines = append(rightLines, lines[i][rightOffset:rightOffset+rightWidth])
	}
	if got := strings.Join(rightLines, "\n"); got != right {
		t.Fatalf("expected right operand block to remain %q, got %q", right, got)
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

	var cli struct {
		Join Options `cmd:""`
	}
	parser, err := kong.New(&cli)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.Parse([]string{"join", "--separator", ",", "a", "b"}); err != nil {
		t.Fatal(err)
	}
	if cli.Join.Separator != "," {
		t.Fatalf("expected --separator flag to populate Options.Separator, got %q", cli.Join.Separator)
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
	var cli struct {
		Join Options `cmd:""`
	}
	parser, err := kong.New(&cli)
	if err != nil {
		t.Fatal(err)
	}
	_, err = parser.Parse([]string{"join", "--separatr", ","})
	if err == nil {
		t.Fatal("expected misspelled flag to fail")
	}

	message := err.Error()
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
