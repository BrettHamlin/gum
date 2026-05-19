package join

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var gumPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gum-join-tests-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp dir: %v\n", err)
		os.Exit(1)
	}
	gumPath = filepath.Join(dir, "gum")

	cmd := exec.Command("go", "build", "-o", gumPath, ".")
	cmd.Dir = ".."
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build gum: %v\n%s", err, output)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

//harness:criterion=c-join-separator-flag-exists,c-join-options-separator-field
func TestSeparatorFlagExistsInHelpAndRuns(t *testing.T) {
	stdout, stderr := runGum(t, "join", "--separator=::<*>", "foo", "bar")
	if stderr != "" {
		t.Fatalf("expected empty stderr when --separator is supplied, got %q", stderr)
	}
	if got, want := shellOutput(stdout), "foo::<*>bar"; got != want {
		t.Fatalf("expected arbitrary separator value to be accepted:\ngot:  %q\nwant: %q", got, want)
	}

	help, _ := runGum(t, "join", "--help")
	if !strings.Contains(help, "--separator") {
		t.Fatalf("help output does not contain --separator:\n%s", help)
	}
	if !strings.Contains(help, "Separator") && !strings.Contains(help, "separator") {
		t.Fatalf("help output does not include descriptive separator text:\n%s", help)
	}

	single, _ := runGum(t, "join", "--horizontal", "foo")
	if got := shellOutput(single); got != "foo" {
		t.Fatalf("expected default separator to be empty for single value, got %q", got)
	}
}

//harness:criterion=c-join-separator-default-empty,c-join-separator-empty-string-explicit,c-join-command-no-interleave-empty
func TestEmptySeparatorMatchesOmittedSeparator(t *testing.T) {
	for _, mode := range []string{"--horizontal", "--vertical"} {
		t.Run(mode, func(t *testing.T) {
			omitted, _ := runGum(t, "join", mode, "foo", "bar")
			explicit, _ := runGum(t, "join", mode, "--separator=", "foo", "bar")
			if omitted != explicit {
				t.Fatalf("expected omitted and explicit empty separator output to match for %s:\nomitted: %q\nexplicit: %q", mode, omitted, explicit)
			}
		})
	}
}

//harness:criterion=c-join-separator-horizontal-interleaved,c-join-separator-preserves-spaces
func TestHorizontalSeparatorIsInterleavedAndPreservesSpaces(t *testing.T) {
	stdout, _ := runGum(t, "join", "--horizontal", "--separator= | ", "foo", "bar", "baz")
	if got, want := shellOutput(stdout), "foo | bar | baz"; got != want {
		t.Fatalf("horizontal separator output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
	if !strings.Contains(shellOutput(stdout), " | ") {
		t.Fatalf("expected separator spaces to be preserved in %q", stdout)
	}
}

//harness:criterion=c-join-separator-vertical-interleaved
func TestVerticalSeparatorIsInterleavedOnSeparateLines(t *testing.T) {
	stdout, _ := runGum(t, "join", "--vertical", "--separator=---", "foo", "bar", "baz")
	got := shellOutput(stdout)
	want := "foo\n---\nbar\n---\nbaz"
	if got != want {
		t.Fatalf("vertical separator output mismatch:\ngot:\n%q\nwant:\n%q", got, want)
	}
	if count := countExactLine(got, "---"); count != 2 {
		t.Fatalf("expected separator line to appear twice, got %d in %q", count, got)
	}
}

//harness:criterion=c-join-separator-two-values-one-delimiter
func TestTwoValuesUseOneDelimiter(t *testing.T) {
	stdout, _ := runGum(t, "join", "--horizontal", "--separator=|", "alpha", "beta")
	if got := strings.Count(shellOutput(stdout), "|"); got != 1 {
		t.Fatalf("expected exactly one separator, got %d in %q", got, stdout)
	}
}

//harness:criterion=c-join-separator-single-value-no-delimiter
func TestSingleValueDoesNotUseDelimiter(t *testing.T) {
	stdout, _ := runGum(t, "join", "--horizontal", "--separator=|", "solo")
	got := shellOutput(stdout)
	if got != "solo" {
		t.Fatalf("expected single value output to be unchanged, got %q", got)
	}
	if strings.Contains(got, "|") {
		t.Fatalf("expected no separator in single value output, got %q", got)
	}
}

//harness:criterion=c-join-separator-horizontal-default-mode
func TestSeparatorAppliesInDefaultHorizontalMode(t *testing.T) {
	defaultMode, _ := runGum(t, "join", "--separator=-", "foo", "bar")
	explicitHorizontal, _ := runGum(t, "join", "--horizontal", "--separator=-", "foo", "bar")
	if defaultMode != explicitHorizontal {
		t.Fatalf("default mode should match explicit horizontal mode:\ndefault:  %q\nexplicit: %q", defaultMode, explicitHorizontal)
	}
	if got := shellOutput(defaultMode); got != "foo-bar" {
		t.Fatalf("expected separator to apply in default mode, got %q", got)
	}
}

//harness:criterion=c-join-separator-multiline-values
func TestSeparatorWithMultilineValues(t *testing.T) {
	stdout, _ := runGum(t, "join", "--vertical", "--separator", "===", "line1\nline2", "plain")
	got := shellOutput(stdout)
	if count := countExactLine(got, "==="); count != 1 {
		t.Fatalf("expected one separator line between two values, got %d in %q", count, got)
	}
	if !strings.Contains(got, "line1\nline2") || !strings.Contains(got, "plain") {
		t.Fatalf("expected multiline value and plain value to remain intact, got %q", got)
	}
}

//harness:criterion=c-join-separator-dash-prefixed-value
func TestDashPrefixedSeparatorValueIsNotMisparsed(t *testing.T) {
	stdout, stderr := runGum(t, "join", "--horizontal", "--separator=---", "foo", "bar", "baz")
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if got := shellOutput(stdout); got != "foo---bar---baz" {
		t.Fatalf("expected dash-prefixed separator to be interleaved, got %q", got)
	}
}

//harness:criterion=c-join-command-interleave-logic
func TestSeparatorInterleaveLogicForHorizontalAndVertical(t *testing.T) {
	horizontal, _ := runGum(t, "join", "--horizontal", "--separator=X", "a", "b", "c")
	if got := shellOutput(horizontal); got != "aXbXc" {
		t.Fatalf("expected horizontal separator interleaving, got %q", got)
	}

	vertical, _ := runGum(t, "join", "--vertical", "--separator=X", "a", "b", "c")
	if got, want := shellOutput(vertical), "a\nX\nb\nX\nc"; got != want {
		t.Fatalf("expected vertical separator interleaving:\ngot:  %q\nwant: %q", got, want)
	}
}

//harness:criterion=c-join-readme-documents-separator
func TestReadmeDocumentsSeparatorInJoinSection(t *testing.T) {
	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	section := joinReadmeSection(string(readme))
	if !strings.Contains(section, "--separator") {
		t.Fatalf("Join section does not document --separator:\n%s", section)
	}
	if !strings.Contains(section, `--separator " | "`) && !strings.Contains(section, `--separator=' | '`) {
		t.Fatalf("Join section does not include a non-empty separator example:\n%s", section)
	}
}

func runGum(t *testing.T, args ...string) (string, string) {
	t.Helper()

	cmd := exec.Command(gumPath, args...)
	cmd.Dir = ".."
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("gum %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String()
}

func shellOutput(stdout string) string {
	return strings.TrimRight(stdout, "\n")
}

func countExactLine(output, line string) int {
	count := 0
	for _, got := range strings.Split(output, "\n") {
		if got == line {
			count++
		}
	}
	return count
}

func joinReadmeSection(readme string) string {
	start := strings.Index(readme, "\n## Join\n")
	if start == -1 {
		return ""
	}
	section := readme[start+1:]
	if end := strings.Index(section[len("## Join\n"):], "\n## "); end != -1 {
		return section[:len("## Join\n")+end]
	}
	return section
}
