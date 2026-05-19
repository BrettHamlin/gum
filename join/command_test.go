package join

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"strings"
	"testing"
)

func runJoin(t *testing.T, opts Options) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stdout pipe: %v", err)
	}
	os.Stdout = writer

	runErr := opts.Run()
	closeErr := writer.Close()
	os.Stdout = originalStdout
	if runErr != nil {
		t.Fatalf("running join: %v", runErr)
	}
	if closeErr != nil {
		t.Fatalf("closing stdout pipe: %v", closeErr)
	}

	var output bytes.Buffer
	if _, err := io.Copy(&output, reader); err != nil {
		t.Fatalf("reading stdout: %v", err)
	}
	return strings.TrimSuffix(output.String(), "\n")
}

func trimLineRight(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return strings.Join(lines, "\n")
}

func TestSeparatorOptionField(t *testing.T) {
	//harness:criterion=c-separator-field-in-options
	file, err := parser.ParseFile(token.NewFileSet(), "options.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing options.go: %v", err)
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "Options" {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				t.Fatalf("Options is not a struct")
			}
			for _, field := range structType.Fields.List {
				if len(field.Names) != 1 || field.Names[0].Name != "Separator" {
					continue
				}
				ident, ok := field.Type.(*ast.Ident)
				if !ok || ident.Name != "string" {
					t.Fatalf("Separator field type = %T %v, want string", field.Type, field.Type)
				}
				if field.Tag == nil {
					t.Fatalf("Separator field has no struct tag")
				}
				tag := strings.Trim(field.Tag.Value, "`")
				for _, want := range []string{`help:`, `default:""`, `short:"`} {
					if !strings.Contains(tag, want) {
						t.Fatalf("Separator tag %q does not contain %q", tag, want)
					}
				}
				shortValue := strings.Split(strings.Split(tag, `short:"`)[1], `"`)[0]
				if len(shortValue) != 1 {
					t.Fatalf("Separator short alias = %q, want single character", shortValue)
				}
				return
			}
			t.Fatalf("Options struct does not declare Separator string")
		}
	}
	t.Fatalf("Options struct not found")
}

func TestJoinSeparatorOutput(t *testing.T) {
	for name, tt := range map[string]struct {
		options  Options
		want     string
		contains map[string]int
	}{
		"default horizontal": {
			//harness:criterion=c-separator-default-empty-horizontal
			options: Options{Text: []string{"foo", "bar"}, Align: "left"},
			want:    "foobar",
		},
		"default vertical": {
			//harness:criterion=c-separator-default-empty-vertical
			options: Options{Text: []string{"foo", "bar"}, Align: "left", Vertical: true},
			want:    "foo\nbar",
		},
		"horizontal separator": {
			//harness:criterion=c-separator-horizontal-interleaved,c-separator-not-appended-to-last,c-separator-not-prepended-to-first
			options:  Options{Text: []string{"a", "b", "c"}, Align: "left", Separator: " | "},
			want:     "a | b | c",
			contains: map[string]int{" | ": 2},
		},
		"vertical separator": {
			//harness:criterion=c-separator-vertical-interleaved,c-separator-not-appended-to-last,c-separator-not-prepended-to-first
			options:  Options{Text: []string{"a", "b", "c"}, Align: "left", Vertical: true, Separator: "\n---\n"},
			want:     "a\n\n---\n\nb\n\n---\n\nc",
			contains: map[string]int{"\n---\n": 2},
		},
		"single element": {
			//harness:criterion=c-separator-single-element-no-insertion
			options:  Options{Text: []string{"hello"}, Align: "left", Separator: " | "},
			want:     "hello",
			contains: map[string]int{" | ": 0},
		},
		"multiline horizontal": {
			//harness:criterion=c-separator-multiline-input-horizontal
			options:  Options{Text: []string{"foo\nbar", "baz\nqux"}, Align: "left", Separator: " | "},
			want:     "foo | baz\nbar   qux",
			contains: map[string]int{" | ": 1},
		},
		"alignment left with separator": {
			//harness:criterion=c-separator-alignment-decode-unchanged
			options: Options{Text: []string{"foo", "bar"}, Align: "left", Separator: " | "},
			want:    "foo | bar",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := runJoin(t, tt.options)
			compareGot := got
			compareWant := tt.want
			if tt.options.Vertical {
				compareGot = trimLineRight(got)
				compareWant = trimLineRight(tt.want)
			}
			if compareGot != compareWant {
				t.Fatalf("output = %q, want %q", got, tt.want)
			}
			for needle, count := range tt.contains {
				if gotCount := strings.Count(got, needle); gotCount != count {
					t.Fatalf("count of %q = %d, want %d in %q", needle, gotCount, count, got)
				}
			}
		})
	}
}

func TestSeparatorNotAtEdges(t *testing.T) {
	for name, tt := range map[string]struct {
		options   Options
		separator string
	}{
		"horizontal": {
			//harness:criterion=c-separator-not-appended-to-last,c-separator-not-prepended-to-first
			options:   Options{Text: []string{"a", "b"}, Align: "left", Separator: " | "},
			separator: " | ",
		},
		"vertical": {
			//harness:criterion=c-separator-not-appended-to-last,c-separator-not-prepended-to-first
			options:   Options{Text: []string{"a", "b"}, Align: "left", Vertical: true, Separator: "---"},
			separator: "---",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := runJoin(t, tt.options)
			if strings.HasPrefix(got, tt.separator) {
				t.Fatalf("output %q starts with separator %q", got, tt.separator)
			}
			if strings.HasSuffix(got, tt.separator) {
				t.Fatalf("output %q ends with separator %q", got, tt.separator)
			}
		})
	}
}

func TestExplicitEmptySeparatorMatchesDefault(t *testing.T) {
	for name, tt := range map[string]struct {
		base     Options
		explicit Options
	}{
		"horizontal": {
			//harness:criterion=c-separator-default-empty-horizontal,c-separator-empty-string-explicit
			base:     Options{Text: []string{"foo", "bar"}, Align: "left"},
			explicit: Options{Text: []string{"foo", "bar"}, Align: "left", Separator: ""},
		},
		"vertical": {
			//harness:criterion=c-separator-empty-string-explicit
			base:     Options{Text: []string{"foo", "bar"}, Align: "left", Vertical: true},
			explicit: Options{Text: []string{"foo", "bar"}, Align: "left", Vertical: true, Separator: ""},
		},
	} {
		t.Run(name, func(t *testing.T) {
			base := runJoin(t, tt.base)
			explicit := runJoin(t, tt.explicit)
			if explicit != base {
				t.Fatalf("explicit empty separator output = %q, want omitted separator output %q", explicit, base)
			}
		})
	}
}

func TestLipglossJoinCallSignatures(t *testing.T) {
	//harness:criterion=c-separator-no-lipgloss-signature-change
	file, err := parser.ParseFile(token.NewFileSet(), "command.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing command.go: %v", err)
	}

	found := map[string]bool{}
	foundJoinCall := false
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.SelectorExpr:
			pkg, ok := node.X.(*ast.Ident)
			if ok && pkg.Name == "lipgloss" && (node.Sel.Name == "JoinHorizontal" || node.Sel.Name == "JoinVertical") {
				found[node.Sel.Name] = true
			}
		case *ast.CallExpr:
			ident, ok := node.Fun.(*ast.Ident)
			if !ok || ident.Name != "join" {
				return true
			}
			if len(node.Args) != 2 {
				t.Fatalf("join call has %d args, want position plus variadic text", len(node.Args))
			}
			if _, ok := node.Args[0].(*ast.IndexExpr); !ok {
				t.Fatalf("join first arg is %T, want decoded align position", node.Args[0])
			}
			if !node.Ellipsis.IsValid() {
				t.Fatalf("join call does not use variadic text")
			}
			if ident, ok := node.Args[1].(*ast.Ident); !ok || ident.Name != "text" {
				t.Fatalf("join variadic arg = %T, want text...", node.Args[1])
			}
			foundJoinCall = true
		}
		return true
	})
	for _, name := range []string{"JoinHorizontal", "JoinVertical"} {
		if !found[name] {
			t.Fatalf("lipgloss.%s reference not found", name)
		}
	}
	if !foundJoinCall {
		t.Fatalf("join(decode.Align[o.Align], text...) call not found")
	}

	got := runJoin(t, Options{Text: []string{"a", "b", "c"}, Align: "left", Separator: " | "})
	if got != "a | b | c" {
		t.Fatalf("separator output = %q, want %q", got, "a | b | c")
	}
}

func TestAlignmentDecodeStillWorks(t *testing.T) {
	for name, tt := range map[string]struct {
		options Options
		want    string
	}{
		"left": {
			//harness:criterion=c-separator-alignment-decode-unchanged
			options: Options{Text: []string{"foo", "bar"}, Align: "left"},
			want:    "foobar",
		},
		"center": {
			//harness:criterion=c-separator-alignment-decode-unchanged
			options: Options{Text: []string{"foo", "bar"}, Align: "center"},
			want:    "foobar",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := runJoin(t, tt.options); got != tt.want {
				t.Fatalf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadmeJoinSectionMentionsSeparator(t *testing.T) {
	//harness:criterion=c-separator-readme-updated
	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}

	content := string(readme)
	joinStart := strings.Index(content, "\n## Join\n")
	if joinStart == -1 {
		t.Fatalf("README.md does not contain ## Join section")
	}
	remaining := content[joinStart+1:]
	nextSection := strings.Index(remaining[len("## Join\n"):], "\n## ")
	joinSection := remaining
	if nextSection != -1 {
		joinSection = remaining[:len("## Join\n")+nextSection]
	}
	if !strings.Contains(joinSection, "--separator") {
		t.Fatalf("README.md Join section does not mention --separator")
	}
}
