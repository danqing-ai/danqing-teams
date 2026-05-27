package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"danqing-teams/internal/api/mcp"
	"danqing-teams/internal/api/rest"
	"danqing-teams/internal/api/rest/controller"
	"danqing-teams/internal/application/service"
	"danqing-teams/internal/application/service/events"
	"danqing-teams/internal/domain/model"
	"danqing-teams/internal/persistence"
	"danqing-teams/internal/provider/llm/mock"
)

func noopEvents() model.EventPublisher {
	return events.NewNoop()
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	storeKind := env("TEAMS_STORE", "sqlite")
	dbPath := env("TEAMS_DB_PATH", "./data/teams.db")
	instanceID := env("TEAMS_INSTANCE_ID", "")

	reg, kind, closeStore, err := persistence.Open(ctx, storeKind, dbPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer func() { _ = closeStore() }()

	hub := noopEvents()
	llm := mock.New()
	autoApprove := envBool("TEAMS_AUTO_APPROVE", false)

	orch := service.NewOrchestrationService(reg.Teams, reg.Tasks, reg.Approvals, reg.Jobs, llm, hub, autoApprove)
	worker := service.NewOrchestrationWorker(orch, reg.Jobs, reg.Recover, instanceID)
	worker.Start(ctx)

	teamSvc := service.NewTeamService(reg.Teams)
	agentSvc := service.NewAgentService(reg.Agents, reg.Teams)
	taskSvc := service.NewTaskService(reg.Tasks, orch)
	approvalSvc := service.NewApprovalService(reg.Teams, reg.Tasks, reg.Approvals, hub, orch)
	todos := service.NewTodoService(reg.Teams)
	workspace := service.NewWorkspaceService(reg.Workspace)

	h := &controller.Controller{
		Teams: teamSvc, Tasks: taskSvc, Approvals: approvalSvc,
		Todos: todos, Workspace: workspace, Agents: agentSvc,
	}
	mcpTools := &mcp.Tools{Teams: teamSvc, Tasks: taskSvc, Approvals: approvalSvc}

	addr := env("TEAMS_ADDR", "0.0.0.0:7801")
	log.Printf("danqing-teams listening on %s (store=%s llm=mock auto_approve=%v instance=%s)",
		addr, kind, autoApprove, workerInstanceID(instanceID))

	if err := rest.NewRouter(h, mcpTools).Run(addr); err != nil {
		log.Fatal(err)
	}
}

func workerInstanceID(instanceID string) string {
	if instanceID != "" {
		return instanceID
	}
	host, _ := os.Hostname()
	return host
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
