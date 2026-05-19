package join

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

var gumBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gum-join-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp dir: %v\n", err)
		os.Exit(1)
	}
	gumBinary = dir + "/gum"

	cmd := exec.Command("go", "build", "-o", gumBinary, "..")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build gum binary: %v\n%s", err, output)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func runGumJoin(t *testing.T, args ...string) (string, string) {
	t.Helper()

	cmd := exec.Command(gumBinary, append([]string{"join"}, args...)...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("gum join %q failed: %v\nstdout: %q\nstderr: %q", args, err, stdout.String(), stderr.String())
	}

	return stdout.String(), stderr.String()
}

func TestJoinCLIOutput(t *testing.T) {
	for name, tt := range map[string]struct {
		criterion string
		args      []string
		want      string
	}{
		"horizontal default": {
			//harness:criterion=c-join-horizontal-no-separator-default
			criterion: "c-join-horizontal-no-separator-default",
			args:      []string{"left", "right"},
			want:      "leftright\n",
		},
		"horizontal with separator": {
			//harness:criterion=c-join-horizontal-with-separator
			criterion: "c-join-horizontal-with-separator",
			args:      []string{"--separator", " | ", "left", "right"},
			want:      "left | right\n",
		},
		"vertical default": {
			//harness:criterion=c-join-vertical-no-separator-default
			criterion: "c-join-vertical-no-separator-default",
			args:      []string{"--vertical", "top", "bottom"},
			want:      "top\nbottom\n",
		},
		"vertical with separator": {
			//harness:criterion=c-join-vertical-with-separator
			criterion: "c-join-vertical-with-separator",
			args:      []string{"--vertical", "--separator=---", "top", "bottom"},
			want:      "top\n---\nbottom\n",
		},
		"short separator flag": {
			//harness:criterion=c-join-separator-short-flag
			criterion: "c-join-separator-short-flag",
			args:      []string{"-s", "|", "left", "right"},
			want:      "left|right\n",
		},
		"three values": {
			//harness:criterion=c-join-separator-three-or-more-values
			criterion: "c-join-separator-three-or-more-values",
			args:      []string{"--separator", ", ", "a", "b", "c"},
			want:      "a, b, c\n",
		},
		"single value": {
			//harness:criterion=c-join-separator-single-value
			criterion: "c-join-separator-single-value",
			args:      []string{"--separator", "|", "only"},
			want:      "only\n",
		},
		"preserves separator spaces": {
			//harness:criterion=c-join-separator-preserves-internal-spaces
			criterion: "c-join-separator-preserves-internal-spaces",
			args:      []string{"--separator", "  ", "left", "right"},
			want:      "left  right\n",
		},
		"dash prefixed separator": {
			//harness:criterion=c-join-separator-dash-prefixed-value
			criterion: "c-join-separator-dash-prefixed-value",
			args:      []string{"--separator=---", "left", "right"},
			want:      "left---right\n",
		},
		"no trailing separator": {
			//harness:criterion=c-join-separator-no-trailing-separator
			criterion: "c-join-separator-no-trailing-separator",
			args:      []string{"--separator=,", "a", "b"},
			want:      "a,b\n",
		},
		"legacy horizontal two values": {
			//harness:criterion=c-join-existing-invocations-unchanged
			criterion: "c-join-existing-invocations-unchanged",
			args:      []string{"a", "b"},
			want:      "ab\n",
		},
		"legacy vertical two values": {
			//harness:criterion=c-join-existing-invocations-unchanged
			criterion: "c-join-existing-invocations-unchanged",
			args:      []string{"--vertical", "a", "b"},
			want:      "a\nb\n",
		},
		"legacy horizontal three values": {
			//harness:criterion=c-join-existing-invocations-unchanged
			criterion: "c-join-existing-invocations-unchanged",
			args:      []string{"a", "b", "c"},
			want:      "abc\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			out, stderr := runGumJoin(t, tt.args...)
			if stderr != "" {
				t.Fatalf("%s: expected empty stderr, got %q", tt.criterion, stderr)
			}
			if out != tt.want {
				t.Fatalf("%s: expected stdout %q, got %q", tt.criterion, tt.want, out)
			}
		})
	}
}

func TestJoinCLIEmptySeparatorMatchesDefault(t *testing.T) {
	//harness:criterion=c-join-empty-separator-byte-equality
	defaultOut, defaultStderr := runGumJoin(t, "left", "right")
	emptyOut, emptyStderr := runGumJoin(t, "--separator=", "left", "right")

	if defaultStderr != "" {
		t.Fatalf("expected default invocation stderr to be empty, got %q", defaultStderr)
	}
	if emptyStderr != "" {
		t.Fatalf("expected empty separator invocation stderr to be empty, got %q", emptyStderr)
	}
	if emptyOut != "leftright\n" {
		t.Fatalf("expected empty separator stdout to preserve legacy bytes, got %q", emptyOut)
	}
	if emptyOut != defaultOut {
		t.Fatalf("expected empty separator stdout %q to match default stdout %q", emptyOut, defaultOut)
	}
}

func TestOptionsSeparatorField(t *testing.T) {
	//harness:criterion=c-join-options-separator-field-exported
	optionsType := reflect.TypeOf(Options{})
	field, ok := optionsType.FieldByName("Separator")
	if !ok {
		t.Fatal("expected Options to expose a Separator field")
	}
	if field.Type.Kind() != reflect.String {
		t.Fatalf("expected Separator to be a string field, got %s", field.Type.Kind())
	}
	if field.Tag.Get("help") == "" {
		t.Fatalf("expected Separator field to include a Kong help tag, got raw tag %q", string(field.Tag))
	}
	if got := reflect.ValueOf(Options{}).FieldByName("Separator").String(); got != "" {
		t.Fatalf("expected zero-value Separator to be empty, got %q", got)
	}
}

func TestReadmeDocumentsJoinSeparator(t *testing.T) {
	//harness:criterion=c-join-readme-documents-separator
	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}

	content := string(readme)
	joinStart := strings.Index(content, "\n## Join\n")
	if joinStart == -1 {
		t.Fatal("README.md does not contain a Join section")
	}
	afterJoin := content[joinStart+len("\n## Join\n"):]
	nextSection := strings.Index(afterJoin, "\n## ")
	joinSection := afterJoin
	if nextSection != -1 {
		joinSection = afterJoin[:nextSection]
	}

	if !strings.Contains(joinSection, "--separator") {
		t.Fatalf("expected README Join section to mention --separator")
	}
}
