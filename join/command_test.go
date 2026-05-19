package join

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var gumBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gum-join-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "unable to locate test file")
		os.Exit(1)
	}
	repoRoot := filepath.Dir(filepath.Dir(file))
	gumBinary = filepath.Join(dir, "gum")

	cmd := exec.Command("go", "build", "-o", gumBinary, ".")
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "go build failed: %v\n%s", err, output)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func runGum(t *testing.T, args ...string) (string, string) {
	t.Helper()

	cmd := exec.Command(gumBinary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("gum %s failed: %v\nstderr: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), stderr.String()
}

func TestJoinSeparatorFlagAndHelp(t *testing.T) {
	// harness:criterion=c-join-separator-flag-exists,c-join-options-separator-field,c-join-help-text-separator-documented
	stdout, stderr := runGum(t, "join", "--separator=::", "foo", "bar")
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "foo::bar") {
		t.Fatalf("expected joined output to contain separator, got %q", stdout)
	}

	help, stderr := runGum(t, "join", "--help")
	if stderr != "" {
		t.Fatalf("expected empty help stderr, got %q", stderr)
	}
	if !strings.Contains(help, "--separator") {
		t.Fatalf("expected help to list --separator, got:\n%s", help)
	}
	if !regexp.MustCompile(`--separator(?:=\S+)?\s+\S+`).MatchString(help) {
		t.Fatalf("expected --separator help to include a non-empty description, got:\n%s", help)
	}
}

func TestJoinHorizontalSeparator(t *testing.T) {
	// harness:criterion=c-join-horizontal-separator-inserted,c-join-separator-count-n-minus-1,c-join-separator-multichar-string
	stdout, stderr := runGum(t, "join", "--horizontal", "--separator=::", "foo", "bar", "baz")
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "foo::bar::baz") {
		t.Fatalf("expected separator inserted between every adjacent value, got %q", stdout)
	}
	if got := strings.Count(stdout, "::"); got != 2 {
		t.Fatalf("expected exactly 2 separators for 3 inputs, got %d in %q", got, stdout)
	}

	stdout, stderr = runGum(t, "join", "--horizontal", "--separator= | ", "left", "middle", "right")
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "left | middle | right") {
		t.Fatalf("expected multi-character separator inserted verbatim between values, got %q", stdout)
	}
	if got := strings.Count(stdout, " | "); got != 2 {
		t.Fatalf("expected exactly 2 separators for 3 inputs, got %d in %q", got, stdout)
	}
}

func TestJoinVerticalSeparator(t *testing.T) {
	// harness:criterion=c-join-vertical-separator-inserted,c-join-separator-count-n-minus-1
	stdout, stderr := runGum(t, "join", "--vertical", "--separator=---", "foo", "bar", "baz")
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	want := "foo\n---\nbar\n---\nbaz\n"
	if stdout != want {
		t.Fatalf("expected vertical separator lines\nwant: %q\n got: %q", want, stdout)
	}
	if got := strings.Count(stdout, "---"); got != 2 {
		t.Fatalf("expected exactly 2 separators for 3 inputs, got %d in %q", got, stdout)
	}
}

func TestJoinNoSeparatorOutputUnchanged(t *testing.T) {
	// harness:criterion=c-join-no-separator-horizontal-unchanged,c-join-no-separator-vertical-unchanged,c-join-empty-separator-flag-unchanged,c-join-separator-intersperse-only-nonempty
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{name: "horizontal", args: []string{"join", "--horizontal", "foo", "bar"}, want: "foobar\n"},
		{name: "vertical", args: []string{"join", "--vertical", "foo", "bar"}, want: "foo\nbar\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			baseline, stderr := runGum(t, tt.args...)
			if stderr != "" {
				t.Fatalf("expected empty baseline stderr, got %q", stderr)
			}
			if baseline != tt.want {
				t.Fatalf("expected flagless output to match pre-separator behavior\nwant: %q\n got: %q", tt.want, baseline)
			}

			withEmpty := append([]string{}, tt.args...)
			withEmpty = append(withEmpty, "--separator=")
			emptySeparator, stderr := runGum(t, withEmpty...)
			if stderr != "" {
				t.Fatalf("expected empty separator stderr, got %q", stderr)
			}
			if emptySeparator != baseline {
				t.Fatalf("expected --separator= output to match omitted separator\nwant: %q\n got: %q", baseline, emptySeparator)
			}
		})
	}

	withSeparator, stderr := runGum(t, "join", "--horizontal", "--separator=X", "foo", "bar")
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(withSeparator, "fooXbar") {
		t.Fatalf("expected non-empty separator to be interspersed, got %q", withSeparator)
	}
}

func TestJoinSingleValueWithSeparator(t *testing.T) {
	// harness:criterion=c-join-single-value-no-separator
	stdout, stderr := runGum(t, "join", "--horizontal", "--separator=SEP", "hello")
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if stdout != "hello\n" {
		t.Fatalf("expected single value rendered alone, got %q", stdout)
	}
	if strings.Contains(stdout, "SEP") {
		t.Fatalf("expected no separator for single input, got %q", stdout)
	}
}

func TestJoinSeparatorAlignmentPreserved(t *testing.T) {
	// harness:criterion=c-join-separator-alignment-preserved
	stdout, stderr := runGum(t, "join", "--vertical", "--align=left", "--separator=---", "top", "bottom")
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	want := []string{"top   ", "---   ", "bottom"}
	if len(lines) != len(want) {
		t.Fatalf("expected %d rendered lines, got %d: %q", len(want), len(lines), stdout)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d mismatch: want %q, got %q in %q", i, want[i], lines[i], stdout)
		}
	}
	for _, line := range lines {
		if len(line) != len("bottom") {
			t.Fatalf("expected all lines to share rendered width %d, got %d for %q", len("bottom"), len(line), line)
		}
	}
}

func TestJoinModifiedFilesGofmtClean(t *testing.T) {
	// harness:criterion=c-join-gofmt-clean
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to locate test file")
	}
	repoRoot := filepath.Dir(filepath.Dir(file))
	cmd := exec.Command("gofmt", "-l", "join/options.go", "join/command.go", "join/command_test.go")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gofmt failed: %v\n%s", err, output)
	}
	if len(output) != 0 {
		t.Fatalf("expected gofmt -l to be empty, got:\n%s", output)
	}
}
