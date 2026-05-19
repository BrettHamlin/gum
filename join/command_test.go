package join

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

var (
	buildGumOnce sync.Once
	gumBin       string
	buildGumErr  error
)

func gumBinary(t *testing.T) string {
	t.Helper()

	buildGumOnce.Do(func() {
		dir, err := os.MkdirTemp("", "gum-join-test-")
		if err != nil {
			buildGumErr = err
			return
		}

		gumBin = filepath.Join(dir, "gum")
		cmd := exec.Command("go", "build", "-o", gumBin, "..")
		output, err := cmd.CombinedOutput()
		if err != nil {
			buildGumErr = &commandError{err: err, output: string(output)}
		}
	})
	if buildGumErr != nil {
		t.Fatalf("build gum: %v", buildGumErr)
	}

	return gumBin
}

type commandError struct {
	err    error
	output string
}

func (e *commandError) Error() string {
	return e.err.Error() + "\n" + e.output
}

type gumResult struct {
	stdout string
	stderr string
	err    error
}

func runGum(t *testing.T, env map[string]string, args ...string) gumResult {
	t.Helper()

	cmd := exec.Command(gumBinary(t), args...)
	cmd.Env = cleanEnv(env)

	stdout, err := cmd.Output()
	result := gumResult{stdout: string(stdout), err: err}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.stderr = string(exitErr.Stderr)
	}

	return result
}

func cleanEnv(overrides map[string]string) []string {
	env := os.Environ()
	cleaned := make([]string, 0, len(env)+len(overrides))
	for _, value := range env {
		if strings.HasPrefix(value, "GUM_JOIN_SEPARATOR=") {
			continue
		}
		cleaned = append(cleaned, value)
	}
	for key, value := range overrides {
		cleaned = append(cleaned, key+"="+value)
	}

	return cleaned
}

func stdoutText(stdout string) string {
	return strings.TrimSuffix(stdout, "\n")
}

func requireSuccess(t *testing.T, result gumResult) {
	t.Helper()

	if result.err != nil {
		t.Fatalf("expected success, got err=%v stderr=%q stdout=%q", result.err, result.stderr, result.stdout)
	}
}

func TestJoinSeparatorFlagDeclared(t *testing.T) {
	//harness:criterion=c-separator-flag-declared
	help := runGum(t, nil, "join", "--help")
	requireSuccess(t, help)

	for _, want := range []string{"--separator", "GUM_JOIN_SEPARATOR"} {
		if !strings.Contains(help.stdout, want) {
			t.Fatalf("help output missing %q:\n%s", want, help.stdout)
		}
	}

	flag := runGum(t, nil, "join", "--separator=TEST", "a", "b")
	requireSuccess(t, flag)
}

func TestJoinSeparatorOutput(t *testing.T) {
	//harness:criterion=c-horizontal-no-separator-unchanged,c-separator-empty-string-default,c-horizontal-with-separator,c-vertical-no-separator-unchanged,c-vertical-with-separator,c-separator-env-var-respected,c-separator-flag-overrides-env-var,c-single-value-no-separator-inserted,c-separator-not-appended-trailing,c-horizontal-layout-still-default,c-multichar-separator-supported
	for name, tt := range map[string]struct {
		criteria []string
		env      map[string]string
		args     []string
		want     string
	}{
		"horizontal no separator unchanged": {
			criteria: []string{"c-horizontal-no-separator-unchanged", "c-separator-empty-string-default"},
			args:     []string{"join", "foo", "bar"},
			want:     "foobar",
		},
		"horizontal with separator": {
			criteria: []string{"c-horizontal-with-separator"},
			args:     []string{"join", "--separator", " | ", "foo", "bar", "baz"},
			want:     "foo | bar | baz",
		},
		"vertical no separator unchanged": {
			criteria: []string{"c-vertical-no-separator-unchanged"},
			args:     []string{"join", "--vertical", "foo", "bar"},
			want:     "foo\nbar",
		},
		"vertical with separator": {
			criteria: []string{"c-vertical-with-separator"},
			args:     []string{"join", "--vertical", "--separator", "\n---\n", "foo", "bar", "baz"},
			want:     "foo\n---\nbar\n---\nbaz",
		},
		"separator env var respected": {
			criteria: []string{"c-separator-env-var-respected"},
			env:      map[string]string{"GUM_JOIN_SEPARATOR": " | "},
			args:     []string{"join", "a", "b"},
			want:     "a | b",
		},
		"separator flag overrides env var": {
			criteria: []string{"c-separator-flag-overrides-env-var"},
			env:      map[string]string{"GUM_JOIN_SEPARATOR": "-"},
			args:     []string{"join", "--separator", "+", "a", "b"},
			want:     "a+b",
		},
		"single value no separator inserted": {
			criteria: []string{"c-single-value-no-separator-inserted"},
			args:     []string{"join", "--separator", " | ", "only"},
			want:     "only",
		},
		"separator not appended trailing": {
			criteria: []string{"c-separator-not-appended-trailing"},
			args:     []string{"join", "--separator", " | ", "a", "b"},
			want:     "a | b",
		},
		"horizontal layout still default": {
			criteria: []string{"c-horizontal-layout-still-default"},
			args:     []string{"join", "--separator", "-", "a", "b"},
			want:     "a-b",
		},
		"multichar separator supported": {
			criteria: []string{"c-multichar-separator-supported"},
			args:     []string{"join", "--separator", " <> ", "a", "b", "c"},
			want:     "a <> b <> c",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Logf("harness:criterion=%s", strings.Join(tt.criteria, ","))

			result := runGum(t, tt.env, tt.args...)
			requireSuccess(t, result)
			if got := stdoutText(result.stdout); got != tt.want {
				t.Fatalf("stdout mismatch:\nwant %q\ngot  %q\nraw  %q", tt.want, got, result.stdout)
			}
		})
	}
}

func TestJoinSeparatorFlagOverridesEnvVarDoesNotLeakEnvValue(t *testing.T) {
	//harness:criterion=c-separator-flag-overrides-env-var
	result := runGum(t, map[string]string{"GUM_JOIN_SEPARATOR": "-"}, "join", "--separator", "+", "a", "b")
	requireSuccess(t, result)

	got := stdoutText(result.stdout)
	if got != "a+b" {
		t.Fatalf("stdout mismatch: want %q, got %q", "a+b", got)
	}
	if strings.Contains(got, "-") {
		t.Fatalf("stdout should not contain env separator: %q", got)
	}
}

func TestJoinSeparatorSingleValueHasNoSeparatorCharacters(t *testing.T) {
	//harness:criterion=c-single-value-no-separator-inserted
	result := runGum(t, nil, "join", "--separator", " | ", "only")
	requireSuccess(t, result)

	got := stdoutText(result.stdout)
	if got != "only" {
		t.Fatalf("stdout mismatch: want %q, got %q", "only", got)
	}
	if strings.Contains(got, "|") {
		t.Fatalf("stdout should not contain separator character: %q", got)
	}
}

func TestJoinSeparatorNotPrependedOrAppended(t *testing.T) {
	//harness:criterion=c-separator-not-appended-trailing
	separator := " | "
	result := runGum(t, nil, "join", "--separator", separator, "a", "b")
	requireSuccess(t, result)

	got := stdoutText(result.stdout)
	if got != "a | b" {
		t.Fatalf("stdout mismatch: want %q, got %q", "a | b", got)
	}
	if !strings.HasPrefix(got, "a") || !strings.HasSuffix(got, "b") {
		t.Fatalf("stdout should start with first value and end with last value: %q", got)
	}
	if strings.HasPrefix(got, separator) || strings.HasSuffix(got, separator) {
		t.Fatalf("stdout should not have leading or trailing separator: %q", got)
	}
}

func TestJoinSeparatorZeroValuesNoPanic(t *testing.T) {
	//harness:criterion=c-zero-values-no-panic
	result := runGum(t, nil, "join", "--separator", " | ")
	requireSuccess(t, result)

	if strings.TrimSpace(result.stdout) != "" {
		t.Fatalf("expected empty or whitespace-only stdout, got %q", result.stdout)
	}
	if strings.Contains(strings.ToLower(result.stderr), "panic") {
		t.Fatalf("stderr should not contain panic: %q", result.stderr)
	}
}

func TestReadmeIncludesJoinSeparatorExample(t *testing.T) {
	//harness:criterion=c-readme-separator-example
	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(string(readme), "\n")
	for i, line := range lines {
		lineNumber := i + 1
		if strings.Contains(line, "--separator") && lineNumber >= 303 && lineNumber <= 343 {
			return
		}
	}

	var matches []string
	for i, line := range lines {
		if strings.Contains(line, "--separator") {
			matches = append(matches, strconv.Itoa(i+1)+":"+line)
		}
	}
	t.Fatalf("README.md missing --separator example within join section near line 323; matches: %v", matches)
}
