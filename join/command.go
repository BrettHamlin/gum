// Package join provides a shell script interface for the lipgloss
// JoinHorizontal and JoinVertical commands. It allows you to join multi-line
// text to build different layouts.
//
// For example, you can place two bordered boxes next to each other: Note: We
// wrap the variable in quotes to ensure the new lines are part of a single
// argument. Otherwise, the command won't work as expected.
//
// $ gum join --horizontal "$BUBBLE_BOX" "$GUM_BOX"
//
// ╔══════════════════════╗╔═════════════╗
// ║                      ║║             ║
// ║        Bubble        ║║     Gum     ║
// ║                      ║║             ║
// ╚══════════════════════╝╚═════════════╝
package join

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/gum/internal/decode"
)

// Run is the command-line interface for the joining strings through lipgloss.
func (o Options) Run() error {
	join := lipgloss.JoinHorizontal
	if o.Vertical {
		join = lipgloss.JoinVertical
	}
	text := o.Text
	if o.Separator != "" {
		text = intersperse(text, o.Separator)
	}
	fmt.Println(join(decode.Align[o.Align], text...))
	return nil
}

func intersperse(values []string, separator string) []string {
	if len(values) < 2 {
		return values
	}

	text := make([]string, 0, len(values)*2-1)
	for i, value := range values {
		if i > 0 {
			text = append(text, separator)
		}
		text = append(text, value)
	}
	return text
}
