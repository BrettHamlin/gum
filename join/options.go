package join

import (
	"fmt"
	"reflect"

	"github.com/alecthomas/kong"
)

// Options is the set of options that can configure a join.
type Options struct {
	Text []string `arg:"" help:"Text to join."`

	Align      string `help:"Text alignment" enum:"left,center,right,bottom,middle,top" default:"left"`
	Separator  string `help:"Separator to place between each joined value" default:"" type:"separator"`
	Horizontal bool   `help:"Join (potentially multi-line) strings horizontally"`
	Vertical   bool   `help:"Join (potentially multi-line) strings vertically"`
}

// SeparatorMapper preserves separator strings that look like flags.
type SeparatorMapper struct{}

func (SeparatorMapper) Decode(ctx *kong.DecodeContext, target reflect.Value) error {
	token := ctx.Scan.Pop()
	if token.IsEOL() {
		return fmt.Errorf("expected separator value but got %q", token.String())
	}
	target.SetString(token.String())
	return nil
}
