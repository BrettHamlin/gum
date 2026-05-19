package join

import (
	"bytes"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var gumPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gum-join-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	gumPath = filepath.Join(dir, "gum")
	cmd := exec.Command("go", "build", "-o", gumPath, "..")
	cmd.Env = cleanEnv(nil)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		os.Stderr.Write(output.Bytes())
		panic(err)
	}

	os.Exit(m.Run())
}

//harness:criterion=c-join-horizontal-no-separator-unchanged
func TestJoinHorizontalNoSeparatorUnchanged(t *testing.T) {
	got := runGum(t, nil, "join", "--horizontal", "A", "B", "C")
	if got != "ABC\n" {
		t.Fatalf("expected horizontal join without separator to be %q, got %q", "ABC\n", got)
	}
}

//harness:criterion=c-join-vertical-no-separator-unchanged
func TestJoinVerticalNoSeparatorUnchanged(t *testing.T) {
	got := runGum(t, nil, "join", "--vertical", "A", "B", "C")
	if got != "A\nB\nC\n" {
		t.Fatalf("expected vertical join without separator to be %q, got %q", "A\nB\nC\n", got)
	}
}

//harness:criterion=c-join-horizontal-with-separator-inserts-delimiter,c-join-separator-not-appended-to-last-item
func TestJoinHorizontalWithSeparatorInsertsDelimiterBetweenItemsOnly(t *testing.T) {
	got := runGum(t, nil, "join", "--horizontal", "--separator=|", "A", "B", "C")
	if got != "A|B|C\n" {
		t.Fatalf("expected horizontal join with separator to be %q, got %q", "A|B|C\n", got)
	}

	rendered := strings.TrimSuffix(got, "\n")
	if strings.HasPrefix(rendered, "|") {
		t.Fatalf("separator was prepended before first item: %q", got)
	}
	if strings.HasSuffix(rendered, "|") {
		t.Fatalf("separator was appended after last item: %q", got)
	}
	if count := strings.Count(rendered, "|"); count != 2 {
		t.Fatalf("expected separator to appear exactly twice, got %d in %q", count, got)
	}
}

//harness:criterion=c-join-vertical-with-separator-inserts-delimiter
func TestJoinVerticalWithSeparatorInsertsDelimiterBetweenItems(t *testing.T) {
	got := runGum(t, nil, "join", "--vertical", "--separator=---", "A", "B", "C")
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	want := []string{"A", "---", "B", "---", "C"}
	if strings.Join(lines, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("expected vertical separator lines %#v, got %#v from %q", want, lines, got)
	}
	if lines[0] == "---" || lines[len(lines)-1] == "---" {
		t.Fatalf("separator should not be first or last line: %#v", lines)
	}
}

//harness:criterion=c-join-separator-empty-string-is-no-op,c-join-command-intersperse-only-when-nonempty
func TestJoinEmptySeparatorIsNoOp(t *testing.T) {
	without := []byte(runGum(t, nil, "join", "--horizontal", "A", "B", "C"))
	withEmpty := []byte(runGum(t, nil, "join", "--horizontal", "--separator=", "A", "B", "C"))

	withoutSum := sha256.Sum256(without)
	withEmptySum := sha256.Sum256(withEmpty)
	if withoutSum != withEmptySum {
		t.Fatalf("expected empty separator output checksum %x to match no-separator checksum %x", withEmptySum, withoutSum)
	}
	if !bytes.Equal(without, withEmpty) {
		t.Fatalf("expected empty separator output %q to match no-separator output %q", withEmpty, without)
	}
}

//harness:criterion=c-join-separator-env-var-accepted
func TestJoinSeparatorEnvironmentVariableAccepted(t *testing.T) {
	fromEnv := runGum(t, map[string]string{"GUM_JOIN_SEPARATOR": "|"}, "join", "--horizontal", "A", "B", "C")
	fromFlag := runGum(t, nil, "join", "--horizontal", "--separator=|", "A", "B", "C")
	if fromEnv != fromFlag {
		t.Fatalf("expected env separator output %q to match flag separator output %q", fromEnv, fromFlag)
	}
	if count := strings.Count(strings.TrimSuffix(fromEnv, "\n"), "|"); count != 2 {
		t.Fatalf("expected env separator to appear exactly twice, got %d in %q", count, fromEnv)
	}
}

//harness:criterion=c-join-separator-flag-overrides-env-var
func TestJoinSeparatorFlagOverridesEnvironmentVariable(t *testing.T) {
	got := runGum(t, map[string]string{"GUM_JOIN_SEPARATOR": "|"}, "join", "--horizontal", "--separator=-", "A", "B", "C")
	if got != "A-B-C\n" {
		t.Fatalf("expected flag separator to override env separator with %q, got %q", "A-B-C\n", got)
	}
	rendered := strings.TrimSuffix(got, "\n")
	if strings.Contains(rendered, "|") {
		t.Fatalf("expected env separator not to appear when flag is set: %q", got)
	}
	if count := strings.Count(rendered, "-"); count != 2 {
		t.Fatalf("expected flag separator to appear exactly twice, got %d in %q", count, got)
	}
}

//harness:criterion=c-join-separator-single-item-no-separator-inserted
func TestJoinSeparatorSingleItemNoSeparatorInserted(t *testing.T) {
	got := runGum(t, nil, "join", "--horizontal", "--separator=|", "A")
	if got != "A\n" {
		t.Fatalf("expected single item output to be %q, got %q", "A\n", got)
	}
	if strings.Contains(got, "|") {
		t.Fatalf("expected no separator for single item, got %q", got)
	}
}

//harness:criterion=c-join-separator-multiline-alignment-unaffected
func TestJoinSeparatorMultilineAlignmentUnaffected(t *testing.T) {
	got := runGum(t, nil, "join", "--horizontal", "--separator=|", "foo\nbar\nbaz", "one\ntwo\nthr")
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected three rendered lines, got %d from %q", len(lines), got)
	}

	separatorColumn := -1
	for i, line := range lines {
		if count := strings.Count(line, "|"); count != 1 {
			t.Fatalf("expected line %d to contain exactly one separator, got %d in %q", i, count, line)
		}
		column := strings.Index(line, "|")
		if separatorColumn == -1 {
			separatorColumn = column
		} else if column != separatorColumn {
			t.Fatalf("expected separator column %d on line %d, got %d in %q", separatorColumn, i, column, line)
		}
	}
}

//harness:criterion=c-join-options-separator-field-defined
func TestJoinHelpDocumentsSeparatorFlagAndEnvironmentVariable(t *testing.T) {
	got := runGum(t, nil, "join", "--help")
	if !strings.Contains(got, "--separator") {
		t.Fatalf("expected join help to include --separator flag, got:\n%s", got)
	}
	if !strings.Contains(strings.ToLower(got), "separator") && !strings.Contains(strings.ToLower(got), "delimiter") {
		t.Fatalf("expected join help to describe separator or delimiter behavior, got:\n%s", got)
	}
	if !strings.Contains(got, "GUM_JOIN_SEPARATOR") {
		t.Fatalf("expected join help to include GUM_JOIN_SEPARATOR env var, got:\n%s", got)
	}
}

//harness:criterion=c-join-readme-documents-separator-flag
func TestReadmeDocumentsJoinSeparatorFlag(t *testing.T) {
	content := readFile(t, filepath.Join("..", "README.md"))
	joinSection := markdownSection(t, content, "## Join")
	if !strings.Contains(joinSection, "--separator") {
		t.Fatalf("expected README join section to document --separator, got:\n%s", joinSection)
	}
	if !strings.Contains(joinSection, "GUM_JOIN_SEPARATOR") {
		t.Fatalf("expected README join section to document GUM_JOIN_SEPARATOR, got:\n%s", joinSection)
	}
	if !strings.Contains(joinSection, "gum join") || !strings.Contains(joinSection, "--separator") {
		t.Fatalf("expected README join section to include a gum join --separator usage example, got:\n%s", joinSection)
	}
}

//harness:criterion=c-join-examples-test-sh-includes-separator
func TestExamplesIncludeJoinSeparatorInvocation(t *testing.T) {
	content := readFile(t, filepath.Join("..", "examples", "test.sh"))
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "gum join") && strings.Contains(line, "--separator") {
			return
		}
	}
	t.Fatalf("expected examples/test.sh to include a gum join invocation using --separator")
}

func runGum(t *testing.T, env map[string]string, args ...string) string {
	t.Helper()
	cmd := exec.Command(gumPath, args...)
	cmd.Env = cleanEnv(env)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("gum %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func cleanEnv(extra map[string]string) []string {
	env := make([]string, 0, len(os.Environ())+len(extra))
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GUM_JOIN_SEPARATOR=") {
			continue
		}
		env = append(env, entry)
	}
	for key, value := range extra {
		env = append(env, key+"="+value)
	}
	return env
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func markdownSection(t *testing.T, content, heading string) string {
	t.Helper()
	start := strings.Index(content, heading)
	if start == -1 {
		t.Fatalf("could not find README section %q", heading)
	}
	rest := content[start+len(heading):]
	end := strings.Index(rest, "\n## ")
	if end == -1 {
		return rest
	}
	return rest[:end]
}
