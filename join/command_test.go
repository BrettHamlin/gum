package join

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

type testCLI struct {
	Join Options `cmd:"" help:"Join text vertically or horizontally"`
}

func runJoin(t *testing.T, args ...string) string {
	t.Helper()

	var cli testCLI
	ctx, err := kong.New(&cli, kong.Exit(func(int) {}))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	parsed, err := ctx.Parse(append([]string{"join"}, args...))
	if err != nil {
		t.Fatalf("failed to parse args %v: %v", args, err)
	}

	stdout := captureStdout(t, func() {
		if err := parsed.Run(); err != nil {
			t.Fatalf("failed to run args %v: %v", args, err)
		}
	})

	return stdout
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = write
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	if err := write.Close(); err != nil {
		t.Fatalf("failed to close stdout writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, read); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	if err := read.Close(); err != nil {
		t.Fatalf("failed to close stdout reader: %v", err)
	}

	return buf.String()
}

//harness:criterion=c-join-horizontal-no-separator-unchanged,c-join-vertical-no-separator-unchanged,c-join-horizontal-empty-separator-unchanged,c-join-vertical-empty-separator-unchanged,c-join-horizontal-with-separator,c-join-vertical-with-separator,c-join-separator-single-value-no-separator-emitted,c-join-separator-preserves-spaces-in-delimiter,c-join-separator-preserves-trailing-newline,c-join-separator-value-looks-like-flag
func TestJoinSeparatorStdoutContract(t *testing.T) {
	for name, tt := range map[string]struct {
		args []string
		want string
	}{
		"horizontal no separator": {
			args: []string{"a", "b", "c"},
			want: "abc\n",
		},
		"vertical no separator": {
			args: []string{"--vertical", "a", "b", "c"},
			want: "a\nb\nc\n",
		},
		"horizontal empty separator": {
			args: []string{"--separator", "", "a", "b", "c"},
			want: "abc\n",
		},
		"horizontal empty separator equals syntax": {
			args: []string{"--separator=", "a", "b", "c"},
			want: "abc\n",
		},
		"vertical empty separator": {
			args: []string{"--vertical", "--separator", "", "a", "b", "c"},
			want: "a\nb\nc\n",
		},
		"vertical empty separator equals syntax": {
			args: []string{"--vertical", "--separator=", "a", "b", "c"},
			want: "a\nb\nc\n",
		},
		"horizontal with separator": {
			args: []string{"--separator", ",", "a", "b", "c"},
			want: "a,b,c\n",
		},
		"vertical with separator": {
			args: []string{"--vertical", "--separator=---", "a", "b"},
			want: "a\n---\nb\n",
		},
		"single value with separator": {
			args: []string{"--separator", ",", "a"},
			want: "a\n",
		},
		"separator preserves spaces": {
			args: []string{"--separator", " | ", "a", "b"},
			want: "a | b\n",
		},
		"separator value looks like flag": {
			args: []string{"--separator=---", "a", "b"},
			want: "a---b\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := runJoin(t, tt.args...)
			if got != tt.want {
				t.Fatalf("stdout mismatch\nwant: %q\n got: %q", tt.want, got)
			}
			if !strings.HasSuffix(got, "\n") {
				t.Fatalf("stdout does not end with newline: %q", got)
			}
			if strings.HasSuffix(strings.TrimSuffix(got, "\n"), "\n") {
				t.Fatalf("stdout has more than one trailing newline: %q", got)
			}
		})
	}
}

//harness:criterion=c-join-separator-not-appended-or-prepended
func TestJoinSeparatorIsOnlyBetweenValues(t *testing.T) {
	horizontal := runJoin(t, "--separator", ",", "a", "b", "c")
	if strings.HasPrefix(horizontal, ",") {
		t.Fatalf("horizontal stdout has leading separator: %q", horizontal)
	}
	if strings.HasSuffix(strings.TrimSuffix(horizontal, "\n"), ",") {
		t.Fatalf("horizontal stdout has trailing separator: %q", horizontal)
	}

	vertical := runJoin(t, "--vertical", "--separator=---", "a", "b", "c")
	if strings.HasPrefix(vertical, "---\n") {
		t.Fatalf("vertical stdout has leading separator line: %q", vertical)
	}
	if strings.HasSuffix(strings.TrimSuffix(vertical, "\n"), "\n---") {
		t.Fatalf("vertical stdout has trailing separator line: %q", vertical)
	}
}

//harness:criterion=c-join-env-var-gum-join-separator,c-join-flag-overrides-env-var
func TestJoinSeparatorEnvironmentContract(t *testing.T) {
	t.Setenv("GUM_JOIN_SEPARATOR", ",")

	if got, want := runJoin(t, "a", "b", "c"), "a,b,c\n"; got != want {
		t.Fatalf("stdout with GUM_JOIN_SEPARATOR mismatch\nwant: %q\n got: %q", want, got)
	}

	got := runJoin(t, "--separator", ";", "a", "b", "c")
	if want := "a;b;c\n"; got != want {
		t.Fatalf("stdout with flag overriding GUM_JOIN_SEPARATOR mismatch\nwant: %q\n got: %q", want, got)
	}
	if strings.Contains(got, ",") {
		t.Fatalf("stdout used env separator despite explicit flag: %q", got)
	}
}

//harness:criterion=c-join-options-separator-field-exists
func TestJoinOptionsDocumentsSeparatorFlagForKong(t *testing.T) {
	source := readFile(t, "options.go")
	re := regexp.MustCompile("(?s)Separator\\s+string\\s+`[^`]*help:\"[^\"]+\"[^`]*default:\"\"[^`]*env:\"GUM_JOIN_SEPARATOR\"[^`]*`")
	if !re.MatchString(source) {
		t.Fatalf("Separator string field must include help, default:\"\", and env:\"GUM_JOIN_SEPARATOR\" tags")
	}
}

//harness:criterion=c-join-command-interleaves-separator
func TestJoinCommandInterleavesSeparatorBeforeLipglossJoin(t *testing.T) {
	source := readFile(t, "command.go")
	if !strings.Contains(source, "o.Separator") {
		t.Fatalf("Run must reference Options.Separator")
	}
	if !strings.Contains(source, "append(text, o.Separator)") {
		t.Fatalf("Run must append the separator into the joined text sequence")
	}

	separatorIndex := strings.Index(source, "append(text, o.Separator)")
	joinIndex := strings.Index(source, "fmt.Println(join(")
	if joinIndex == -1 {
		t.Fatalf("Run must print the lipgloss join output")
	}
	if separatorIndex == -1 || separatorIndex > joinIndex {
		t.Fatalf("Run must interleave the separator before invoking the join function")
	}
}

//harness:criterion=c-join-readme-documents-separator-flag
func TestReadmeDocumentsJoinSeparatorFlag(t *testing.T) {
	source := readFile(t, "../README.md")
	joinSection := section(t, source, "## Join", "## Format")

	for _, want := range []string{"--separator", "default is", `""`, "gum join --separator"} {
		if !strings.Contains(joinSection, want) {
			t.Fatalf("README join section missing %q", want)
		}
	}
}

//harness:criterion=c-join-command-contracts-documents-separator
func TestCommandContractsDocumentJoinSeparatorFlag(t *testing.T) {
	source := readFile(t, "../.harness/agents/command-contracts.md")
	joinSection := section(t, source, "## Join command contract", "## Scope note")

	for _, want := range []string{"--separator", "defaults to", `""`, "between adjacent joined values"} {
		if !strings.Contains(joinSection, want) {
			t.Fatalf("command contract join entry missing %q", want)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	return string(content)
}

func section(t *testing.T, source, start, end string) string {
	t.Helper()

	startIndex := strings.Index(source, start)
	if startIndex == -1 {
		t.Fatalf("missing section start %q", start)
	}
	endIndex := strings.Index(source[startIndex:], end)
	if endIndex == -1 {
		t.Fatalf("missing section end %q", end)
	}

	return source[startIndex : startIndex+endIndex]
}
