package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"danqing-teams/core/domain"
)

type ExecShell struct{}

func (h *ExecShell) Name() string                { return "exec_shell" }
func (h *ExecShell) RiskLevel() domain.RiskLevel { return domain.RiskHigh }
func (h *ExecShell) Describe(args map[string]any) string {
	cmd, _ := args["command"].(string)
	if cmd == "" {
		return "exec_shell"
	}
	if len(cmd) > 100 {
		cmd = cmd[:100] + "..."
	}
	return cmd
}
func (h *ExecShell) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name: "exec_shell",
		Description: "Execute a shell command and return its stdout/stderr. HIGH RISK — requires user approval.\n\n" +
			"**Important**: Commands run in the project root directory. Use relative paths when referencing project files.\n\n" +
			"- Use only for builds, tests, git operations, or commands with no tool alternative.\n" +
			"- Do NOT use for reading/writing files — use read_file, write, edit, or apply_patch instead.\n" +
			"- Do NOT use for searching file contents — use grep or glob instead.\n" +
			"- Avoid destructive commands unless the user explicitly requests them.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string", "description": "Shell command to execute"},
				"timeout": map[string]any{"type": "integer", "description": "Timeout in seconds (default: 30)"},
			},
			"required": []string{"command"},
		},
	}
}
func (h *ExecShell) Execute(_ context.Context, input map[string]any) (domain.ToolResult, error) {
	cmdStr, _ := input["command"].(string)
	if cmdStr == "" {
		return domain.ToolResult{}, fmt.Errorf("command is required")
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}
	if wd := workDirFromInput(input); wd != "" {
		cmd.Dir = wd
	}
	out, err := cmd.CombinedOutput()
	content := strings.TrimSpace(string(out))
	if err != nil {
		if content == "" {
			content = err.Error()
		} else {
			content = content + "\nerror: " + err.Error()
		}
		return domain.ToolResult{Content: content}, nil
	}
	return domain.ToolResult{Content: content}, nil
}
