package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"danqing-teams/core/bootstrap"
	"danqing-teams/core/domain"
	"danqing-teams/core/runtime/sandbox"
)

func main() {
	if sandbox.MaybeReexec() {
		return
	}
	core := bootstrap.New(bootstrap.Config{ConfigPath: os.Getenv("TEAMS_CONFIG")})
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("DanQing Teams CLI. Type 'exit' to quit.")
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "exit" {
			break
		}
		if input == "" {
			continue
		}
		session, err := core.Sessions.Create(context.Background(), domain.CreateSessionRequest{
			Content: input,
		})
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		fmt.Println("session created:", session.ID)
		ch := core.Sessions.Subscribe(session.ID)
		for ev := range ch {
			fmt.Printf("[%s] %s\n", ev.Type, string(ev.Payload))
		}
	}
}
