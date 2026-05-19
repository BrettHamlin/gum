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
	if o.Separator != "" {
		separator := o.Separator
		if !o.Vertical {
			separator = horizontalSeparator(text, separator)
		}
		text = intersperse(text, separator)
	}
	output := join(decode.Align[o.Align], text...)
	if o.Vertical && o.Separator != "" {
		output = trimTrailingLineSpaces(output)
	}
	fmt.Println(output)
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

func horizontalSeparator(values []string, separator string) string {
	if lipgloss.Height(separator) != 1 {
		return separator
	}

	height := 1
	for _, value := range values {
		if h := lipgloss.Height(value); h > height {
			height = h
		}
	}
	if height == 1 {
		return separator
	}

	lines := make([]string, height)
	for i := range lines {
		lines[i] = separator
	}
	return strings.Join(lines, "\n")
}

func trimTrailingLineSpaces(value string) string {
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}
