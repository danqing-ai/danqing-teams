// Command mcp exposes Teams tools over stdio (JSON lines) for external agents.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"os"

	"danqing-teams/internal/api/mcp"
	"danqing-teams/internal/persistence/memory"
	"danqing-teams/internal/provider/llm/mock"
	"danqing-teams/internal/service"
	"danqing-teams/internal/service/events"
)

func main() {
	log.SetOutput(os.Stderr)
	store := memory.NewStore()
	_ = memory.SeedDemoTeam(context.Background(), store)
	hub := events.NewNoop()
	orch := service.NewOrchestrationService(store, store, store, store, mock.New(), hub, true)
	worker := service.NewOrchestrationWorker(orch, store, store, "test")
	worker.Start(context.Background())
	tools := &mcp.Tools{
		Teams:     service.NewTeamService(store),
		Tasks:     service.NewTaskService(store, orch),
		Approvals: service.NewApprovalService(store, store, store, hub, orch),
	}

	sc := bufio.NewScanner(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	for sc.Scan() {
		var req mcp.CallRequest
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			_ = enc.Encode(map[string]string{"error": err.Error()})
			continue
		}
		resp, err := tools.Call(context.Background(), req)
		if err != nil {
			_ = enc.Encode(map[string]string{"error": err.Error()})
			continue
		}
		_ = enc.Encode(resp)
	}
}
