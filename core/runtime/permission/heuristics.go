package permission

import (
	"regexp"
	"strings"
)

var (
	reRmRf     = regexp.MustCompile(`(?i)\brm\s+(-[a-zA-Z]*r[a-zA-Z]*f|-rf|-fr)\b`)
	reSudo     = regexp.MustCompile(`(?i)\bsudo\b`)
	reMkfs     = regexp.MustCompile(`(?i)\bmkfs(\.\w+)?\b`)
	reDdIf     = regexp.MustCompile(`(?i)\bdd\s+.*\bif=`)
	reCurlSh   = regexp.MustCompile(`(?i)\b(curl|wget)\b[^|;]*\|\s*(ba)?sh\b`)
	reChmod777 = regexp.MustCompile(`(?i)\bchmod\s+(-R\s+)?777\b`)
	reSSHWrite = regexp.MustCompile(`(?i)(~|/home/[^/\s]+)/\.ssh\b`)
)

// LooksDangerous reports commands that should always ask even inside a strong sandbox.
func LooksDangerous(cmd string) bool {
	c := strings.TrimSpace(cmd)
	if c == "" {
		return false
	}
	switch {
	case reSudo.MatchString(c),
		reRmRf.MatchString(c),
		reMkfs.MatchString(c),
		reDdIf.MatchString(c),
		reCurlSh.MatchString(c),
		reChmod777.MatchString(c),
		reSSHWrite.MatchString(c) && looksLikeWrite(c):
		return true
	default:
		return false
	}
}

func looksLikeWrite(cmd string) bool {
	lower := strings.ToLower(cmd)
	for _, tok := range []string{" >", ">>", "tee ", "cp ", "mv ", "install ", "cat >"} {
		if strings.Contains(lower, tok) {
			return true
		}
	}
	return false
}

var networkPatterns = []string{
	"curl ", "curl\t", "wget ", "wget\t",
	"npm install", "npm i ", "npx ",
	"yarn add", "pnpm install", "pnpm i ",
	"pip install", "pip3 install", "uv pip",
	"go get ", "go install ",
	"cargo install",
	"git push", "git pull", "git fetch", "git clone",
	"docker pull", "docker push",
	"apt-get ", "apt install", "brew install",
}

// LooksLikeNetwork reports commands that typically need outbound network.
func LooksLikeNetwork(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	if lower == "" {
		return false
	}
	for _, p := range networkPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	// bare curl/wget as first token
	fields := strings.Fields(lower)
	if len(fields) > 0 {
		switch fields[0] {
		case "curl", "wget":
			return true
		}
	}
	return false
}
