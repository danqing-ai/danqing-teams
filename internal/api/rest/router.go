package rest

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"danqing-teams/internal/api/mcp"
	"danqing-teams/internal/api/rest/controller"
	"danqing-teams/internal/api/rest/middleware"
)

func NewRouter(h *controller.Controller, mcpTools *mcp.Tools) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.Recovery(), middleware.Logging(), middleware.CORS())

	r.GET("/health", h.Health)
	r.GET("/api/health", h.Health)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/agents", h.ListAgents)
		v1.POST("/agents", h.PostAgent)
		v1.GET("/agents/:agentId", h.GetAgent)
		v1.PATCH("/agents/:agentId", h.PatchAgent)
		v1.DELETE("/agents/:agentId", h.DeleteAgent)

		v1.GET("/teams", h.ListTeams)
		v1.POST("/teams", h.CreateTeam)
		v1.GET("/teams/:teamId", h.GetTeam)
		v1.PATCH("/teams/:teamId", h.UpdateTeam)
		v1.DELETE("/teams/:teamId", h.DeleteTeam)

		v1.GET("/teams/:teamId/controller", h.GetController)
		v1.PUT("/teams/:teamId/controller", h.PutController)

		v1.GET("/teams/:teamId/workers", h.ListWorkers)
		v1.POST("/teams/:teamId/workers", h.PostWorker)
		v1.PATCH("/teams/:teamId/workers/:workerId", h.PatchWorker)
		v1.DELETE("/teams/:teamId/workers/:workerId", h.DeleteWorker)
		v1.GET("/teams/:teamId/agent-members", h.ListTeamAgentMembers)
		v1.POST("/teams/:teamId/agent-members/:agentId", h.PostTeamAgentMember)
		v1.DELETE("/teams/:teamId/agent-members/:agentId", h.DeleteTeamAgentMember)
		v1.GET("/teams/:teamId/workers/:workerId/knowledge", h.ListKnowledge)
		v1.PUT("/teams/:teamId/workers/:workerId/knowledge/docs", h.PutKnowledge)

		v1.GET("/teams/:teamId/humans", h.ListHumans)
		v1.POST("/teams/:teamId/humans", h.PostHuman)

		v1.GET("/teams/:teamId/tasks", h.ListTasks)
		v1.POST("/teams/:teamId/tasks", h.SubmitTask)
		v1.POST("/teams/:teamId/messages", h.SendTeamMessage)
		v1.GET("/teams/:teamId/tasks/:taskId/messages", h.ListTaskMessages)
		v1.GET("/teams/:teamId/tasks/:taskId", h.GetTask)
		v1.GET("/teams/:teamId/tasks/:taskId/timeline", h.GetTimeline)
		v1.POST("/teams/:teamId/tasks/:taskId/cancel", h.CancelTask)
		v1.GET("/teams/:teamId/tasks/:taskId/reports", h.ListReports)
		v1.GET("/teams/:teamId/tasks/:taskId/runs/:runId/plan", h.GetRunPlan)

		v1.GET("/teams/:teamId/approvals", h.ListApprovals)
		v1.GET("/teams/:teamId/approvals/:approvalId", h.GetApproval)
		v1.POST("/teams/:teamId/approvals/:approvalId/approve", h.Approve)
		v1.POST("/teams/:teamId/approvals/:approvalId/reject", h.Reject)

		v1.GET("/teams/:teamId/workspace", h.ListWorkspace)
		v1.POST("/teams/:teamId/workspace/artifacts", h.PostWorkspace)

		v1.GET("/teams/:teamId/todos", h.ListTodos)
		v1.POST("/teams/:teamId/todos", h.PostTodo)
		v1.PATCH("/teams/:teamId/todos/:todoId", h.PatchTodo)

		if mcpTools != nil {
			mcp.RegisterRoutes(v1.Group("/mcp"), mcpTools)
		}
	}

	staticDir := os.Getenv("DQ_STATIC_DIR")
	if staticDir == "" {
		staticDir = "./out/frontend/dist"
	}
	if _, err := os.Stat(staticDir); err == nil {
		r.Static("/app", staticDir)
		r.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/app/")
		})
	}

	return r
}
