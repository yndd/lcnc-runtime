package fnruntime

import (
	"fmt"
	"strings"
)

// multiLineFormatter knows how to format multiple lines in pretty format
// that can be displayed to an end user.
type multiLineFormatter struct {
	// Title under which lines need to be printed
	Title string

	// Lines to be printed on the CLI.
	Lines []string

	// TruncateOuput determines if output needs to be truncated or not.
	TruncateOutput bool

	// MaxLines to be printed if truncation is enabled.
	MaxLines int

	// UseQuote determines if line needs to be quoted or not
	UseQuote bool
}

// String returns multiline string.
func (ri *multiLineFormatter) String() string {
	if ri.MaxLines == 0 {
		ri.MaxLines = FnExecErrorTruncateLines
	}
	strInterpolator := "%s"
	if ri.UseQuote {
		strInterpolator = "%q"
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s:\n", ri.Title))
	lineIndent := strings.Repeat(" ", FnExecErrorIndentation+2)
	if !ri.TruncateOutput {
		// stderr string should have indentations
		for _, s := range ri.Lines {
			// suppress newlines to avoid poor formatting
			s = strings.ReplaceAll(s, "\n", " ")
			b.WriteString(fmt.Sprintf(lineIndent+strInterpolator+"\n", s))
		}
		return b.String()
	}
	printedLines := 0
	for i, s := range ri.Lines {
		if i >= ri.MaxLines {
			break
		}
		// suppress newlines to avoid poor formatting
		s = strings.ReplaceAll(s, "\n", " ")
		b.WriteString(fmt.Sprintf(lineIndent+strInterpolator+"\n", s))
		printedLines++
	}
	truncatedLines := len(ri.Lines) - printedLines
	if truncatedLines > 0 {
		b.WriteString(fmt.Sprintf(lineIndent+"...(%d line(s) truncated, use '--truncate-output=false' to disable)\n", truncatedLines))
	}
	return b.String()
}
