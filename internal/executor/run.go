package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/biddan606/asksh/internal/safety"
)

// Run executes cmd using the given shell (fallback to "sh") and wires
// stdout/stderr/stdin to out/errOut/in. nil writers/reader default to os.Stdout/os.Stderr/os.Stdin.
// Defense-in-depth: Blocked commands are rejected before any exec call.
func Run(ctx context.Context, shell, cmd string, out, errOut io.Writer, in io.Reader) error {
	if v, reason := safety.Check(cmd); v == safety.Blocked {
		return fmt.Errorf("blocked command refused: %s", reason)
	}

	if shell == "" {
		shell = "sh"
	}
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}
	if in == nil {
		in = os.Stdin
	}

	c := exec.CommandContext(ctx, shell, "-c", cmd)
	c.Stdout = out
	c.Stderr = errOut
	c.Stdin = in
	return c.Run()
}
