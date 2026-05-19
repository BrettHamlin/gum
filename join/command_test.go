package join

import (
	"bytes"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/gum/table"
)

func TestOptionsSeparatorField(t *testing.T) {
	//harness:criterion=c-separator-field-exists,c-separator-default-is-empty
	field, ok := reflect.TypeOf(Options{}).FieldByName("Separator")
	if !ok {
		t.Fatal("Options.Separator field is missing")
	}
	if field.Type.Kind() != reflect.String {
		t.Fatalf("Options.Separator type = %v, want string", field.Type)
	}
	if help, ok := field.Tag.Lookup("help"); !ok || help == "" {
		t.Fatalf("Options.Separator help tag is empty; full tag: %q", field.Tag)
	}
	if def, ok := field.Tag.Lookup("default"); !ok || def != "" {
		t.Fatalf("Options.Separator default tag = %q, want empty string", def)
	}

	tableField, ok := reflect.TypeOf(table.Options{}).FieldByName("Separator")
	if !ok {
		t.Fatal("table.Options.Separator field is missing")
	}
	if tableField.Name != field.Name || tableField.Type != field.Type {
		t.Fatalf("Options.Separator = %s %v, want naming parity with table.Options.Separator = %s %v", field.Name, field.Type, tableField.Name, tableField.Type)
	}
}

func TestJoin(t *testing.T) {
	//harness:criterion=c-horizontal-no-separator-unchanged,c-vertical-no-separator-unchanged,c-horizontal-separator-interleaved,c-vertical-separator-interleaved,c-separator-not-appended-trailing,c-multichar-separator-supported,c-single-value-no-separator-inserted,c-empty-separator-is-noop,c-run-interleaves-before-lipgloss,c-separator-default-is-empty,c-horizontal-default-mode-separator
	for name, tt := range map[string]struct {
		criteria []string
		args     []string
		want     string
	}{
		"horizontal_no_separator": {
			criteria: []string{"c-horizontal-no-separator-unchanged", "c-separator-default-is-empty"},
			args:     []string{"--horizontal", "A", "B", "C"},
			want:     "ABC",
		},
		"vertical_no_separator": {
			criteria: []string{"c-vertical-no-separator-unchanged"},
			args:     []string{"--vertical", "A", "B", "C"},
			want:     "A\nB\nC",
		},
		"horizontal_with_separator": {
			criteria: []string{"c-horizontal-separator-interleaved", "c-separator-not-appended-trailing", "c-run-interleaves-before-lipgloss"},
			args:     []string{"--horizontal", "--separator", "|", "A", "B", "C"},
			want:     "A|B|C",
		},
		"vertical_with_separator": {
			criteria: []string{"c-vertical-separator-interleaved", "c-separator-not-appended-trailing", "c-run-interleaves-before-lipgloss"},
			args:     []string{"--vertical", "--separator", "---", "A", "B", "C"},
			want:     "A\n---\nB\n---\nC",
		},
		"multichar_separator": {
			criteria: []string{"c-multichar-separator-supported"},
			args:     []string{"--horizontal", "--separator", " | ", "A", "B", "C"},
			want:     "A | B | C",
		},
		"single_value_no_separator": {
			criteria: []string{"c-single-value-no-separator-inserted"},
			args:     []string{"--horizontal", "--separator", "|", "A"},
			want:     "A",
		},
		"empty_separator_noop": {
			criteria: []string{"c-empty-separator-is-noop"},
			args:     []string{"--horizontal", "--separator", "", "A", "B", "C"},
			want:     "ABC",
		},
		"default_mode_with_separator": {
			criteria: []string{"c-horizontal-default-mode-separator"},
			args:     []string{"--separator", "|", "A", "B", "C"},
			want:     "A|B|C",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Logf("harness:criterion=%s", strings.Join(tt.criteria, ","))
			got := runGumJoin(t, tt.args...)
			if got != tt.want {
				t.Fatalf("gum join %q stdout = %q, want %q", tt.args, got, tt.want)
			}
			if name == "single_value_no_separator" && strings.Contains(got, "|") {
				t.Fatalf("single value stdout %q contains separator", got)
			}
		})
	}
}

func TestJoinEmptySeparatorNoop(t *testing.T) {
	//harness:criterion=c-empty-separator-is-noop
	horizontal := runGumJoin(t, "--horizontal", "A", "B", "C")
	horizontalEmpty := runGumJoin(t, "--horizontal", "--separator", "", "A", "B", "C")
	if horizontalEmpty != horizontal {
		t.Fatalf("horizontal stdout with empty separator = %q, want baseline %q", horizontalEmpty, horizontal)
	}

	vertical := runGumJoin(t, "--vertical", "A", "B", "C")
	verticalEmpty := runGumJoin(t, "--vertical", "--separator", "", "A", "B", "C")
	if verticalEmpty != vertical {
		t.Fatalf("vertical stdout with empty separator = %q, want baseline %q", verticalEmpty, vertical)
	}
}

func TestJoinSeparatorIsNotLeadingOrTrailing(t *testing.T) {
	//harness:criterion=c-separator-not-appended-trailing
	for name, tt := range map[string]struct {
		args      []string
		separator string
	}{
		"horizontal": {
			args:      []string{"--horizontal", "--separator", "|", "A", "B", "C"},
			separator: "|",
		},
		"vertical": {
			args:      []string{"--vertical", "--separator", "---", "A", "B", "C"},
			separator: "---",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := runGumJoin(t, tt.args...)
			if strings.HasPrefix(got, tt.separator) {
				t.Fatalf("stdout %q starts with separator %q", got, tt.separator)
			}
			if strings.HasSuffix(got, tt.separator) {
				t.Fatalf("stdout %q ends with separator %q", got, tt.separator)
			}
		})
	}
}

func TestJoinHelpDocumentsSeparator(t *testing.T) {
	//harness:criterion=c-gum-go-help-documents-separator
	cmd := exec.Command("go", "run", "..", "join", "--help")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("gum join --help failed: %v\nstderr:\n%s", err, stderr.String())
	}
	if !strings.Contains(strings.ToLower(stdout.String()), "separator") {
		t.Fatalf("gum join --help stdout = %q, want it to mention separator", stdout.String())
	}
}

func TestReadmeDocumentsSeparator(t *testing.T) {
	//harness:criterion=c-readme-documents-separator
	b, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	readme := string(b)
	joinStart := strings.Index(readme, "## Join")
	if joinStart == -1 {
		t.Fatal("README.md does not contain a Join section")
	}
	joinSection := readme[joinStart:]
	if next := strings.Index(joinSection[len("## Join"):], "\n## "); next != -1 {
		joinSection = joinSection[:len("## Join")+next]
	}
	if count := strings.Count(joinSection, "--separator"); count < 2 {
		t.Fatalf("README.md Join section contains --separator %d times, want at least 2", count)
	}
	if !strings.Contains(joinSection, "gum join --separator") {
		t.Fatalf("README.md Join section does not include a gum join --separator usage example")
	}
}

func runGumJoin(t *testing.T, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"run", "..", "join"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("gum join %q failed: %v\nstderr:\n%s", args, err, stderr.String())
	}
	return strings.TrimSuffix(stdout.String(), "\n")
}
