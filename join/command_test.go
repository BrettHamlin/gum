package join

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var gumPath string

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	tmp, err := os.MkdirTemp("", "gum-join-test-*")
	if err != nil {
		panic(err)
	}

	gumPath = filepath.Join(tmp, "gum")
	cmd := exec.Command("go", "build", "-o", gumPath, ".")
	cmd.Dir = filepath.Dir(wd)
	if out, err := cmd.CombinedOutput(); err != nil {
		panic(string(out))
	}

	code := m.Run()
	if err := os.RemoveAll(tmp); err != nil {
		panic(err)
	}
	os.Exit(code)
}

func TestJoinSeparatorFlagHelpAndParse(t *testing.T) {
	//harness:criterion=c-join-separator-flag-exists,c-join-options-separator-field-kong-tag
	result := runGum(t, "join", "--help")
	if result.exitCode != 0 {
		t.Fatalf("gum join --help exit code = %d, stderr = %q", result.exitCode, result.stderr)
	}

	help := result.stdout + result.stderr
	if !strings.Contains(help, "--separator") {
		t.Fatalf("help output does not contain --separator:\n%s", help)
	}

	helpAroundSeparator := lineWithFollowing(help, "--separator")
	if !strings.Contains(strings.ToLower(helpAroundSeparator), "insert") {
		t.Fatalf("--separator help text has no description: %q", helpAroundSeparator)
	}

	result = runGum(t, "join", "--separator", ", ", "a", "b")
	if result.exitCode != 0 {
		t.Fatalf("gum join --separator exit code = %d, stderr = %q", result.exitCode, result.stderr)
	}
}

func TestJoinSeparatorOutput(t *testing.T) {
	//harness:criterion=c-join-separator-default-empty,c-join-horizontal-default-unchanged,c-join-vertical-default-unchanged,c-join-horizontal-with-separator,c-join-vertical-with-separator,c-join-separator-not-appended-trailing,c-join-separator-not-prepended-leading,c-join-single-value-separator-ignored,c-join-empty-separator-equals-no-flag,c-join-horizontal-flag-unaffected,c-join-vertical-flag-unaffected,c-join-align-flag-unaffected,c-join-intersperse-implementation
	for name, tt := range map[string]struct {
		criteria []string
		args     []string
		want     string
	}{
		"horizontal default": {
			criteria: []string{
				"c-join-separator-default-empty",
				"c-join-horizontal-default-unchanged",
			},
			args: []string{"join", "--horizontal", "a", "b", "c"},
			want: "abc\n",
		},
		"vertical default": {
			criteria: []string{
				"c-join-vertical-default-unchanged",
			},
			args: []string{"join", "--vertical", "a", "b", "c"},
			want: "a\nb\nc\n",
		},
		"horizontal separator": {
			criteria: []string{
				"c-join-horizontal-with-separator",
				"c-join-separator-not-appended-trailing",
				"c-join-separator-not-prepended-leading",
				"c-join-horizontal-flag-unaffected",
				"c-join-intersperse-implementation",
			},
			args: []string{"join", "--horizontal", "--separator", "|", "a", "b", "c"},
			want: "a|b|c\n",
		},
		"horizontal comma separator": {
			criteria: []string{
				"c-join-horizontal-with-separator",
			},
			args: []string{"join", "--horizontal", "--separator", ", ", "a", "b", "c"},
			want: "a, b, c\n",
		},
		"vertical separator": {
			criteria: []string{
				"c-join-vertical-with-separator",
			},
			args: []string{"join", "--vertical", "--separator", "---", "a", "b", "c"},
			want: "a\n---\nb\n---\nc\n",
		},
		"vertical pipe separator": {
			criteria: []string{
				"c-join-vertical-flag-unaffected",
				"c-join-intersperse-implementation",
			},
			args: []string{"join", "--vertical", "--separator", "|", "a", "b", "c"},
			want: "a\n|\nb\n|\nc\n",
		},
		"single value separator ignored": {
			criteria: []string{
				"c-join-single-value-separator-ignored",
			},
			args: []string{"join", "--horizontal", "--separator", "|", "hello"},
			want: "hello\n",
		},
		"horizontal empty separator": {
			criteria: []string{
				"c-join-empty-separator-equals-no-flag",
			},
			args: []string{"join", "--horizontal", "--separator", "", "a", "b", "c"},
			want: "abc\n",
		},
		"vertical empty separator": {
			criteria: []string{
				"c-join-empty-separator-equals-no-flag",
			},
			args: []string{"join", "--vertical", "--separator", "", "a", "b", "c"},
			want: "a\nb\nc\n",
		},
		"horizontal align with separator": {
			criteria: []string{
				"c-join-align-flag-unaffected",
			},
			args: []string{"join", "--horizontal", "--align", "center", "--separator", "|", "a", "b", "c"},
			want: "a|b|c\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Logf("harness:criterion=%s", strings.Join(tt.criteria, ","))
			result := runGum(t, tt.args...)
			if result.exitCode != 0 {
				t.Fatalf("exit code = %d, stderr = %q", result.exitCode, result.stderr)
			}
			if result.stdout != tt.want {
				t.Fatalf("stdout = %q, want %q", result.stdout, tt.want)
			}
			if strings.HasPrefix(result.stdout, "|") {
				t.Fatalf("stdout has leading separator: %q", result.stdout)
			}
			if strings.HasSuffix(result.stdout, "|\n") {
				t.Fatalf("stdout has trailing separator before newline: %q", result.stdout)
			}
			assertExactlyOneTrailingNewline(t, []byte(result.stdout))
		})
	}
}

func TestJoinEmptySeparatorMatchesNoFlag(t *testing.T) {
	//harness:criterion=c-join-empty-separator-equals-no-flag
	for name, args := range map[string][]string{
		"horizontal": {"join", "--horizontal", "a", "b", "c"},
		"vertical":   {"join", "--vertical", "a", "b", "c"},
	} {
		t.Run(name, func(t *testing.T) {
			without := runGum(t, args...)
			withArgs := append([]string{args[0], args[1], "--separator", ""}, args[2:]...)
			with := runGum(t, withArgs...)
			if without.exitCode != 0 || with.exitCode != 0 {
				t.Fatalf("exit codes without=%d with=%d; stderr without=%q with=%q", without.exitCode, with.exitCode, without.stderr, with.stderr)
			}
			if without.stdout != with.stdout {
				t.Fatalf("empty separator stdout = %q, no-flag stdout = %q", with.stdout, without.stdout)
			}
		})
	}
}

func TestJoinTrailingNewlinePreserved(t *testing.T) {
	//harness:criterion=c-join-trailing-newline-preserved
	for name, args := range map[string][]string{
		"horizontal default":   {"join", "--horizontal", "a", "b", "c"},
		"vertical default":     {"join", "--vertical", "a", "b", "c"},
		"horizontal separator": {"join", "--horizontal", "--separator", "|", "a", "b", "c"},
		"vertical separator":   {"join", "--vertical", "--separator", "---", "a", "b", "c"},
	} {
		t.Run(name, func(t *testing.T) {
			result := runGum(t, args...)
			if result.exitCode != 0 {
				t.Fatalf("exit code = %d, stderr = %q", result.exitCode, result.stderr)
			}
			assertExactlyOneTrailingNewline(t, []byte(result.stdout))
		})
	}
}

type gumResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func runGum(t *testing.T, args ...string) gumResult {
	t.Helper()

	cmd := exec.Command(gumPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			t.Fatalf("failed to run gum %s: %v", strings.Join(args, " "), err)
		}
	}

	return gumResult{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: exitCode,
	}
}

func assertExactlyOneTrailingNewline(t *testing.T, out []byte) {
	t.Helper()

	if len(out) == 0 {
		t.Fatal("stdout is empty")
	}
	if out[len(out)-1] != '\n' {
		t.Fatalf("stdout does not end with newline: %q", string(out))
	}
	if len(out) > 1 && out[len(out)-2] == '\n' {
		t.Fatalf("stdout ends with more than one newline: %q", string(out))
	}
}

func lineWithFollowing(s, substr string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.Contains(line, substr) {
			if i+1 < len(lines) {
				return line + "\n" + lines[i+1]
			}
			return line
		}
	}
	return ""
}
