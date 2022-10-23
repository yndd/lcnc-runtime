package fnruntime

import (
	"fmt"
	"strings"
)

const (
	FnExecErrorTruncateLines = 4
	// FnExecErrorIndentation is the number of spaces at the beginning of each
	// line of function failure messages.
	FnExecErrorIndentation = 2
)

// ExecError implements an error type that stores information about function failure.
type ExecError struct {
	// OriginalErr is the original error returned from function runtime
	OriginalErr error

	// TruncateOutput indicates should error message be truncated
	TruncateOutput bool

	// Stderr is the content written to function stderr
	Stderr string `yaml:"stderr,omitempty"`

	// ExitCode is the exit code returned from function
	ExitCode int `yaml:"exitCode,omitempty"`
}

// String returns string representation of the failure.
func (fe *ExecError) String() string {
	var b strings.Builder

	errLines := &multiLineFormatter{
		Title:          "Stderr",
		Lines:          strings.Split(fe.Stderr, "\n"),
		UseQuote:       true,
		TruncateOutput: fe.TruncateOutput,
	}
	b.WriteString(errLines.String())
	b.WriteString(fmt.Sprintf("  Exit Code: %d\n", fe.ExitCode))
	return b.String()
}

func (fe *ExecError) Error() string {
	return fe.String()
}
