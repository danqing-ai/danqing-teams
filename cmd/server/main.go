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
	"danqing-teams/internal/api/rest/handlers"
	"danqing-teams/internal/contract"
	"danqing-teams/internal/persistence"
	"danqing-teams/internal/provider/llm/mock"
	"danqing-teams/internal/service"
	"danqing-teams/internal/service/events"
)

func noopEvents() contract.EventPublisher {
	return events.NewNoop()
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	storeKind := env("TEAMS_STORE", "sqlite")
	dbPath := env("TEAMS_DB_PATH", "./data/teams.db")
	instanceID := env("TEAMS_INSTANCE_ID", "")

	repos, kind, closeStore, err := persistence.Open(ctx, storeKind, dbPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer func() { _ = closeStore() }()

	hub := noopEvents()
	llm := mock.New()
	autoApprove := envBool("TEAMS_AUTO_APPROVE", false)

	teams := repos.(contract.TeamRepository)
	agents := repos.(contract.AgentRepository)
	tasks := repos.(contract.TaskRepository)
	approvals := repos.(contract.ApprovalRepository)
	jobs := repos.(contract.JobRepository)
	recoverStore := repos.(service.RecoverableTaskStore)

	orch := service.NewOrchestrationService(teams, tasks, approvals, jobs, llm, hub, autoApprove)
	worker := service.NewOrchestrationWorker(orch, jobs, recoverStore, instanceID)
	worker.Start(ctx)

	teamSvc := service.NewTeamService(teams)
	agentSvc := service.NewAgentService(agents, teams)
	taskSvc := service.NewTaskService(tasks, orch)
	approvalSvc := service.NewApprovalService(teams, tasks, approvals, hub, orch)
	todos := service.NewTodoService(repos.(contract.TeamRepository))
	workspace := service.NewWorkspaceService(repos.(contract.WorkspaceRepository))

	h := &handlers.Handlers{
		Teams: teamSvc, Tasks: taskSvc, Approvals: approvalSvc,
		Todos: todos, Workspace: workspace, Agents: agentSvc,
	}
	mcpTools := &mcp.Tools{Teams: teamSvc, Tasks: taskSvc, Approvals: approvalSvc}

	addr := env("TEAMS_ADDR", "0.0.0.0:8080")
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
