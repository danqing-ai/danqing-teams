package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"danqing-teams/core/port"
)

func hostRunShell(ctx context.Context, opts port.SandboxRunOptions) ([]byte, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", opts.Command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", opts.Command)
	}
	cmd.Dir = opts.WorkDir
	if opts.Env != nil {
		cmd.Env = opts.Env
	}
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("command timed out after %s", timeout)
	}
	return out, err
}
