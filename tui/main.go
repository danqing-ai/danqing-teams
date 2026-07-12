package main

import (
	"fmt"
	"os"
	"danqing-teams/core/bootstrap"
)

func main() {
	core := bootstrap.New(bootstrap.Config{ConfigPath: os.Getenv("TEAMS_CONFIG")})
	_ = core
	fmt.Println("DanQing Teams TUI (placeholder)")
}
