package seed

import (
	"context"

	"danqing-teams/internal/domain/model"
	"danqing-teams/pkg/id"
)

type TeamWriter interface {
	ListTeams(ctx context.Context) ([]model.Team, error)
	CreateTeam(ctx context.Context, req model.CreateTeamRequest) (*model.TeamDetail, error)
	AddHuman(ctx context.Context, teamID string, h model.HumanMember) error
}

type AgentWriter interface {
	ListAgents(ctx context.Context, role model.AgentRole) ([]model.Agent, error)
	CreateAgent(ctx context.Context, req model.CreateAgentRequest) (*model.Agent, error)
	AddTeamAgent(ctx context.Context, teamID, agentID string) error
}

// DemoTeam creates the SRE demo team and global agents if the store has no teams.
func DemoTeam(ctx context.Context, s interface {
	TeamWriter
	AgentWriter
}) error {
	teams, err := s.ListTeams(ctx)
	if err != nil {
		return err
	}
	if len(teams) > 0 {
		return nil
	}

	if err := seedGlobalAgents(ctx, s); err != nil {
		return err
	}

	detail, err := s.CreateTeam(ctx, model.CreateTeamRequest{
		Name:        "SRE 作战室",
		Description: "Demo team for alert analysis and cluster operations",
	})
	if err != nil {
		return err
	}
	tid := detail.Team.ID

	workers, err := s.ListAgents(ctx, model.AgentRoleTeamWorker)
	if err != nil {
		return err
	}
	for _, w := range workers {
		if err := s.AddTeamAgent(ctx, tid, w.ID); err != nil {
			return err
		}
	}

	return s.AddHuman(ctx, tid, model.HumanMember{
		ID: id.New(), DisplayName: "值班工程师", Role: "approver",
	})
}

func seedGlobalAgents(ctx context.Context, s AgentWriter) error {
	existing, err := s.ListAgents(ctx, "")
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	_, err = s.CreateAgent(ctx, model.CreateAgentRequest{
		ID:           "team-controller",
		Name:         "Team Controller",
		Description:  "负责理解用户意图，按 Worker 人设分派任务，汇总报告并规划 follow-up。",
		Role:         model.AgentRoleTeamController,
		SystemPrompt: "你是 Team Controller，仅依据 Worker 人设匹配，不知道 Worker 的技能与 MCP Tool。",
		MinFunctionCallingRounds: 2,
	})
	if err != nil {
		return err
	}

	workers := []model.CreateAgentRequest{
		{
			ID:          "alert-analyst",
			Name:        "AlertAnalyst",
			Description: "负责告警归因、指标与日志分析、影响评估与止血建议；不执行集群变更或生产写操作",
			Role:        model.AgentRoleTeamWorker,
			MinFunctionCallingRounds: 1,
			Skills: []model.Skill{
				{ID: "alert.triage", Name: "Alert Triage", Keywords: []string{"告警", "alert"}, RiskLevel: model.RiskLow},
				{ID: "metrics.read", Name: "Metrics Read", Keywords: []string{"cpu", "指标"}, RiskLevel: model.RiskLow},
			},
			Tools: []model.ToolBinding{
				{ToolID: "prometheus.query", MCPServer: "observability-mcp", Name: "Prometheus Query", RiskLevel: model.RiskLow},
				{ToolID: "alertmanager.list", MCPServer: "observability-mcp", Name: "Alertmanager List", RiskLevel: model.RiskLow},
			},
			KnowledgeBase: model.KnowledgeBaseRef{ID: "kb-alert", Name: "告警知识库"},
		},
		{
			ID:          "cluster-operator",
			Name:        "ClusterOperator",
			Description: "负责 Kubernetes 集群运维：容量观察、扩容、节点迁移、集群配置变更；变更前须给出计划",
			Role:        model.AgentRoleTeamWorker,
			MinFunctionCallingRounds: 1,
			Skills: []model.Skill{
				{ID: "cluster.inspect", Name: "Cluster Inspect", Keywords: []string{"节点", "inspect", "查看"}, RiskLevel: model.RiskLow},
				{ID: "k8s.scale", Name: "K8s Scale", Keywords: []string{"扩容", "scale"}, RiskLevel: model.RiskHigh},
				{ID: "node.drain", Name: "Node Drain", Keywords: []string{"迁移", "drain"}, RiskLevel: model.RiskHigh},
			},
			Tools: []model.ToolBinding{
				{ToolID: "k8s.nodes.list", MCPServer: "cluster-ops-mcp", Name: "List Nodes", RiskLevel: model.RiskLow},
				{ToolID: "k8s.deployment.scale", MCPServer: "cluster-ops-mcp", Name: "Scale Deployment", RiskLevel: model.RiskHigh},
			},
			KnowledgeBase: model.KnowledgeBaseRef{ID: "kb-cluster", Name: "集群运维手册"},
		},
		{
			ID:          "config-auditor",
			Name:        "ConfigAuditor",
			Description: "负责配置审查、diff、合规检查与回滚建议；以只读分析为主",
			Role:        model.AgentRoleTeamWorker,
			MinFunctionCallingRounds: 1,
			Skills: []model.Skill{
				{ID: "config.diff", Name: "Config Diff", Keywords: []string{"配置", "diff"}, RiskLevel: model.RiskMedium},
				{ID: "policy.check", Name: "Policy Check", Keywords: []string{"合规"}, RiskLevel: model.RiskLow},
			},
			Tools: []model.ToolBinding{
				{ToolID: "git.diff", MCPServer: "config-mcp", Name: "Git Diff", RiskLevel: model.RiskLow},
				{ToolID: "config.apply", MCPServer: "config-mcp", Name: "Apply Config", RiskLevel: model.RiskHigh},
			},
			KnowledgeBase: model.KnowledgeBaseRef{ID: "kb-config", Name: "配置规范"},
		},
	}
	for _, req := range workers {
		if _, err := s.CreateAgent(ctx, req); err != nil {
			return err
		}
	}
	return nil
}
