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
	"strings"

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
	if o.Separator != "" && len(o.Text) > 1 {
		text = make([]string, 0, len(o.Text)*2-1)
		for i, t := range o.Text {
			if i > 0 {
				text = append(text, o.Separator)
			}
			text = append(text, t)
		}
		if o.Vertical {
			fmt.Println(joinVerticalWithSeparator(decode.Align[o.Align], o.Separator, o.Text...))
			return nil
		}
	}
	fmt.Println(join(decode.Align[o.Align], text...))
	return nil
}

func joinVerticalWithSeparator(pos lipgloss.Position, separator string, text ...string) string {
	width := 0
	for _, t := range text {
		if w := lipgloss.Width(t); w > width {
			width = w
		}
	}

	var b strings.Builder
	for i, t := range text {
		if i > 0 {
			b.WriteRune('\n')
			b.WriteString(separator)
			b.WriteRune('\n')
		}
		b.WriteString(lipgloss.PlaceHorizontal(width, pos, t))
	}
	return b.String()
}
