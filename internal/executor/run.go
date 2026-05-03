package executor

import (
	"context"
	"io"
	"os"
	"os/exec"
)

// Run executes cmd using the given shell (fallback to "sh") and wires
// stdout/stderr to out/errOut. nil writers default to os.Stdout/os.Stderr.
func Run(ctx context.Context, shell, cmd string, out, errOut io.Writer) error {
	if shell == "" {
		shell = "sh"
	}
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}

	c := exec.CommandContext(ctx, shell, "-c", cmd)
	c.Stdout = out
	c.Stderr = errOut
	c.Stdin = os.Stdin
	return c.Run()
}
