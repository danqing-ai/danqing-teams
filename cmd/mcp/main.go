// Command mcp exposes Teams tools over stdio (JSON lines) for external agents.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"os"

	"danqing-teams/internal/api/mcp"
	"danqing-teams/internal/application/service"
	"danqing-teams/internal/application/service/events"
	"danqing-teams/internal/persistence/memory"
	"danqing-teams/internal/provider/llm/mock"
)

func main() {
	log.SetOutput(os.Stderr)
	store := memory.NewStore()
	_ = memory.SeedDemoTeam(context.Background(), store)
	reg := store.Registry()
	hub := events.NewNoop()
	orch := service.NewOrchestrationService(reg.Teams, reg.Tasks, reg.Approvals, reg.Jobs, mock.New(), hub, true)
	worker := service.NewOrchestrationWorker(orch, reg.Jobs, reg.Recover, "test")
	worker.Start(context.Background())
	tools := &mcp.Tools{
		Teams:     service.NewTeamService(reg.Teams),
		Tasks:     service.NewTaskService(reg.Tasks, orch),
		Approvals: service.NewApprovalService(reg.Teams, reg.Tasks, reg.Approvals, hub, orch),
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
