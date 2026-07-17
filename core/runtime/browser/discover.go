package browser

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func lookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// discoverBrowser finds a system Chrome/Chromium/Edge binary.
func discoverBrowser() (path, engine string) {
	candidates := browserCandidates()
	for _, c := range candidates {
		if c.path == "" {
			continue
		}
		if filepath.IsAbs(c.path) {
			if info, err := os.Stat(c.path); err == nil && !info.IsDir() {
				return c.path, c.engine
			}
			continue
		}
		if p, err := exec.LookPath(c.path); err == nil {
			return p, c.engine
		}
	}
	return "", ""
}

type browserCandidate struct {
	path   string
	engine string
}

func browserCandidates() []browserCandidate {
	switch runtime.GOOS {
	case "darwin":
		return []browserCandidate{
			{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "chrome"},
			{"/Applications/Chromium.app/Contents/MacOS/Chromium", "chromium"},
			{"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge", "edge"},
			{"google-chrome", "chrome"},
			{"chromium", "chromium"},
			{"microsoft-edge", "edge"},
		}
	case "windows":
		localApp := os.Getenv("LOCALAPPDATA")
		progFiles := os.Getenv("ProgramFiles")
		progFilesX86 := os.Getenv("ProgramFiles(x86)")
		var out []browserCandidate
		for _, root := range []string{progFiles, progFilesX86, localApp} {
			if root == "" {
				continue
			}
			out = append(out,
				browserCandidate{filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"), "chrome"},
				browserCandidate{filepath.Join(root, "Chromium", "Application", "chrome.exe"), "chromium"},
				browserCandidate{filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"), "edge"},
			)
		}
		out = append(out,
			browserCandidate{"chrome.exe", "chrome"},
			browserCandidate{"msedge.exe", "edge"},
			browserCandidate{"chromium.exe", "chromium"},
		)
		return out
	default: // linux and others
		return []browserCandidate{
			{"google-chrome-stable", "chrome"},
			{"google-chrome", "chrome"},
			{"chromium-browser", "chromium"},
			{"chromium", "chromium"},
			{"microsoft-edge-stable", "edge"},
			{"microsoft-edge", "edge"},
			{"/usr/bin/google-chrome-stable", "chrome"},
			{"/usr/bin/google-chrome", "chrome"},
			{"/usr/bin/chromium-browser", "chromium"},
			{"/usr/bin/chromium", "chromium"},
		}
	}
}
