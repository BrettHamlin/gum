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
	text := intersperse(o.Text, o.Separator)
	fmt.Println(join(decode.Align[o.Align], text...))
	return nil
}

func intersperse(text []string, separator string) []string {
	if separator == "" || len(text) < 2 {
		return text
	}

	joined := make([]string, 0, len(text)*2-1)
	for i, t := range text {
		if i > 0 {
			joined = append(joined, separator)
		}
		joined = append(joined, t)
	}
	return joined
}
