package join

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestRunSeparatorOutput(t *testing.T) {
	for name, tt := range map[string]struct {
		options Options
		want    string
	}{
		"horizontal no separator": {
			//harness:criterion=c-horizontal-no-separator-unchanged
			options: Options{Horizontal: true, Text: []string{"foo", "bar"}},
			want:    "foobar\n",
		},
		"vertical no separator": {
			//harness:criterion=c-vertical-no-separator-unchanged
			options: Options{Vertical: true, Text: []string{"foo", "bar"}},
			want:    "foo\nbar\n",
		},
		"horizontal with separator": {
			//harness:criterion=c-horizontal-with-separator
			options: Options{Horizontal: true, Separator: "|", Text: []string{"foo", "bar"}},
			want:    "foo|bar\n",
		},
		"vertical with separator": {
			//harness:criterion=c-vertical-with-separator
			options: Options{Vertical: true, Separator: "---", Text: []string{"foo", "bar"}},
			want:    "foo\n---\nbar\n",
		},
		"horizontal separator only between values": {
			//harness:criterion=c-separator-not-appended-trailing,c-separator-not-prepended-leading
			options: Options{Horizontal: true, Separator: "|", Text: []string{"foo", "bar", "baz"}},
			want:    "foo|bar|baz\n",
		},
		"vertical separator only between values": {
			//harness:criterion=c-separator-not-appended-trailing,c-separator-not-prepended-leading
			options: Options{Vertical: true, Separator: "---", Text: []string{"foo", "bar", "baz"}},
			want:    "foo\n---\nbar\n---\nbaz\n",
		},
		"horizontal single value": {
			//harness:criterion=c-separator-single-value
			options: Options{Horizontal: true, Separator: "|", Text: []string{"solo"}},
			want:    "solo\n",
		},
		"vertical single value": {
			//harness:criterion=c-separator-single-value
			options: Options{Vertical: true, Separator: "|", Text: []string{"solo"}},
			want:    "solo\n",
		},
		"multi-character separator": {
			//harness:criterion=c-separator-arbitrary-string
			options: Options{Horizontal: true, Separator: " | ", Text: []string{"foo", "bar"}},
			want:    "foo | bar\n",
		},
		"tab separator": {
			//harness:criterion=c-separator-arbitrary-string
			options: Options{Horizontal: true, Separator: "\t", Text: []string{"foo", "bar"}},
			want:    "foo\tbar\n",
		},
		"dash separator": {
			//harness:criterion=c-separator-arbitrary-string
			options: Options{Horizontal: true, Separator: "---", Text: []string{"foo", "bar"}},
			want:    "foo---bar\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := captureRunOutput(t, tt.options)
			if got != tt.want {
				t.Fatalf("expected stdout %q, got %q", tt.want, got)
			}
		})
	}
}

func TestRunExplicitEmptySeparatorMatchesDefault(t *testing.T) {
	for name, tt := range map[string]struct {
		defaultOptions Options
		emptyOptions   Options
	}{
		"horizontal": {
			//harness:criterion=c-separator-default-empty,c-separator-empty-string-explicit
			defaultOptions: Options{Horizontal: true, Text: []string{"foo", "bar"}},
			emptyOptions:   Options{Horizontal: true, Separator: "", Text: []string{"foo", "bar"}},
		},
		"vertical": {
			//harness:criterion=c-separator-empty-string-explicit
			defaultOptions: Options{Vertical: true, Text: []string{"foo", "bar"}},
			emptyOptions:   Options{Vertical: true, Separator: "", Text: []string{"foo", "bar"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			gotDefault := captureRunOutput(t, tt.defaultOptions)
			gotEmpty := captureRunOutput(t, tt.emptyOptions)
			if gotDefault != gotEmpty {
				t.Fatalf("expected explicit empty separator stdout %q to match default stdout %q", gotEmpty, gotDefault)
			}
		})
	}
}

func TestJoinSeparatorFlagAccepted(t *testing.T) {
	//harness:criterion=c-separator-flag-exists
	stdout, stderr, err := runGum(t, "join", "--separator=|", "foo", "bar")
	if err != nil {
		t.Fatalf("expected command to exit 0, got %v; stderr=%q stdout=%q", err, stderr, stdout)
	}
	errText := strings.ToLower(stderr)
	if strings.Contains(errText, "unknown flag") || strings.Contains(errText, "unrecognized") {
		t.Fatalf("separator flag was rejected: %q", stderr)
	}
}

func TestJoinHelpIncludesOptionalSeparator(t *testing.T) {
	//harness:criterion=c-options-separator-field
	stdout, stderr, err := runGum(t, "join", "--help")
	if err != nil {
		t.Fatalf("expected help to exit 0, got %v; stderr=%q stdout=%q", err, stderr, stdout)
	}

	separatorLine := ""
	for _, line := range strings.Split(stdout, "\n") {
		if strings.Contains(line, "--separator") {
			separatorLine = line
			break
		}
	}
	if separatorLine == "" {
		t.Fatalf("expected help output to include --separator, got:\n%s", stdout)
	}
	if strings.TrimSpace(strings.ReplaceAll(separatorLine, "--separator", "")) == "" {
		t.Fatalf("expected --separator help line to include a description, got %q", separatorLine)
	}
	if strings.Contains(strings.ToLower(separatorLine), "required") || strings.Contains(separatorLine, "*") {
		t.Fatalf("expected --separator help not to be marked required, got %q", separatorLine)
	}
}

func TestReadmeDocumentsJoinSeparator(t *testing.T) {
	//harness:criterion=c-readme-separator-documented
	data, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}

	readme := string(data)
	start := strings.Index(readme, "## Join")
	if start == -1 {
		t.Fatal("README.md does not contain a Join section")
	}
	rest := readme[start+len("## Join"):]
	end := strings.Index(rest, "\n## ")
	joinSection := rest
	if end != -1 {
		joinSection = rest[:end]
	}
	if !strings.Contains(joinSection, "--separator") {
		t.Fatalf("expected README Join section to document --separator, got:\n%s", joinSection)
	}
}

func captureRunOutput(t *testing.T, options Options) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = originalStdout
	})

	err = options.Run()
	if closeErr := writer.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func runGum(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	cmd := exec.Command("go", append([]string{"run", ".."}, args...)...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
