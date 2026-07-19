package runtime

import (
	"strings"
	"testing"

	"danqing-teams/core/domain"
)

func TestBuildRuntimeEnvironmentGitBash(t *testing.T) {
	out := buildRuntimeEnvironment(domain.SandboxStatus{
		Backend:   domain.SandboxBackendWinToken,
		Shell:     "bash (Git for Windows)",
		ShellPath: `C:\Program Files\Git\bin\bash.exe`,
	})
	for _, want := range []string{
		"Shell: bash (Git for Windows)",
		`Shell path: C:\Program Files\Git\bin\bash.exe`,
		"Sandbox backend: win-token",
		"Prefer POSIX shell syntax",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestBuildRuntimeEnvironmentWSL2(t *testing.T) {
	out := buildRuntimeEnvironment(domain.SandboxStatus{
		Backend: domain.SandboxBackendWSL2,
		Shell:   "bash (WSL2)",
	})
	if !strings.Contains(out, "Shell: bash (WSL2)") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "via: wsl -e bash -lc") {
		t.Fatal(out)
	}
	if strings.Contains(out, "Shell path:") {
		t.Fatalf("unexpected shell path:\n%s", out)
	}
}

func TestBuildRuntimeEnvironmentCmdFallback(t *testing.T) {
	out := buildRuntimeEnvironment(domain.SandboxStatus{
		Backend: domain.SandboxBackendWinToken,
		Shell:   "cmd",
	})
	if !strings.Contains(out, "Git Bash not detected") {
		t.Fatal(out)
	}
}
