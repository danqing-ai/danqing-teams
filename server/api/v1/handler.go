package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"danqing-teams/core/service"
	"danqing-teams/core/domain"
	"danqing-teams/core/port"

	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldmarkHtml "github.com/yuin/goldmark/renderer/html"
)

// Version is set at build time via -ldflags.
var Version = "dev"

type Handler struct {
	Sessions     *service.SessionManager
	Projects     *service.ProjectManager
	LLMConfig    *service.LLMConfigManager
	Config       *service.ConfigManager
	SearchConfig *service.SearchConfigManager
	Agents       *service.AgentManager
	Skills       *service.SkillManager
	SkillHandler *SkillHandler
	TurnLogs     *service.TurnLogManager
	MCPServers   *service.MCPManager
	Store        port.Repository
}

type RouterConfig struct {
	FrontendDir string
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		// Allow Tauri desktop (tauri://localhost), web dev (localhost:*), and same-origin
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func NewRouter(h *Handler, cfg RouterConfig) *gin.Engine {
	r := gin.New()
	r.Use(corsMiddleware())
	api := r.Group("/api/v1")
	api.POST("/sessions", createSession(h))
	api.GET("/sessions", listSessions(h))
	api.GET("/sessions/:id", getSession(h))
	api.PATCH("/sessions/:id", updateSession(h))
	api.DELETE("/sessions/:id", deleteSession(h))
	api.POST("/sessions/:id/turns", sendMessage(h))
	api.GET("/sessions/:id/turns", listTurns(h))
	api.POST("/sessions/:id/turns/:turnID/resume", resumeTurn(h))
	api.DELETE("/sessions/:id/turns/:turnID", cancelTurn(h))
	api.GET("/sessions/:id/turns/:turnID/log", downloadTurnLog(h))
	api.GET("/sessions/:id/events", streamEvents(h))
	api.GET("/sessions/:id/events/poll", pollEvents(h))
	api.POST("/projects", createProject(h))
	api.GET("/projects", listProjects(h))
	api.GET("/projects/:id", getProject(h))
	api.PATCH("/projects/:id", updateProject(h))
	api.DELETE("/projects/:id", deleteProject(h))
	api.GET("/projects/:id/sessions", listProjectSessions(h))
	api.GET("/projects/:id/files", listProjectFiles(h))
	api.GET("/projects/:id/files/content", readProjectFile(h))
	api.GET("/projects/:id/raw/*filepath", serveProjectFile(h))
	api.GET("/proxy/*target", proxyDevServer(h))
	api.GET("/projects/:id/git-changes", getProjectGitChanges(h))
	api.GET("/llm/configs", getLLMConfigs(h))
	api.POST("/llm/configs", createLLMConfig(h))
	api.PUT("/llm/configs/:id", updateLLMConfig(h))
	api.DELETE("/llm/configs/:id", deleteLLMConfig(h))
	api.POST("/llm/configs/fetch-models", fetchLLMModelsFromRequest(h))
	api.POST("/llm/configs/:id/refresh-models", refreshLLMModels(h))
	api.PATCH("/llm/configs/:id/models/:modelName", toggleLLMModel(h))
	api.GET("/llm/models", listLLMModels(h))
	api.GET("/llm/presets", getLLMPresets(h))
	api.POST("/approvals/:id/decide", decideApproval(h))
	api.POST("/asks/:id/resolve", resolveAskUser(h))
	api.GET("/config", getConfig(h))
	api.PUT("/config", updateConfig(h))
	api.GET("/model-configs", getModelConfigs(h))
	api.PUT("/model-configs", updateModelConfigs(h))
	api.GET("/search/config", getSearchConfig(h))
	api.PUT("/search/config", updateSearchConfig(h))
	api.GET("/agents", listAgents(h))
	api.POST("/agents", createAgent(h))
	api.GET("/agents/:id", getAgent(h))
	api.PUT("/agents/:id", updateAgent(h))
	api.POST("/agents/:id/reset", resetAgent(h))
	api.DELETE("/agents/:id", deleteAgent(h))
	api.GET("/mcp/servers", listMCPServers(h))
	api.POST("/mcp/servers", createMCPServer(h))
	api.GET("/mcp/servers/:id", getMCPServer(h))
	api.PUT("/mcp/servers/:id", updateMCPServer(h))
	api.DELETE("/mcp/servers/:id", deleteMCPServer(h))
	api.POST("/mcp/servers/:id/refresh-tools", refreshMCPTools(h))
	api.PATCH("/mcp/servers/:id/tools/:toolName", toggleMCPTool(h))

	api.GET("/skills", listSkills(h.SkillHandler))
	api.POST("/skills", createSkill(h.SkillHandler))
	api.GET("/skills/:id", getSkill(h.SkillHandler))
	api.PUT("/skills/:id", updateSkill(h.SkillHandler))
	api.DELETE("/skills/:id", deleteSkill(h.SkillHandler))
	api.POST("/skills/import", importSkillDir(h.SkillHandler))
	api.POST("/skills/:id/reset", resetSkill(h.SkillHandler))
	api.GET("/skills/:id/export", exportSkillMD(h.SkillHandler))
	api.GET("/skills/:id/files", listSkillFiles(h.SkillHandler))
	api.GET("/skills/:id/files/*path", getSkillFile(h.SkillHandler))

	api.GET("/version", getVersion())

	if cfg.FrontendDir != "" {
		r.Static("/app", cfg.FrontendDir)
		r.NoRoute(func(c *gin.Context) {
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/app" {
				c.File(cfg.FrontendDir + "/index.html")
				return
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		})
	}

	return r
}

func createSession(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.CreateSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if cfgs, _ := h.LLMConfig.GetAll(c); len(cfgs) > 0 {
			if _, _, err := h.LLMConfig.ResolveProvider(c, req.ModelID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
		session, err := h.Sessions.Create(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, session)
	}
}

func listSessions(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessions, err := h.Sessions.List(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, sessions)
	}
}

func getSession(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := h.Sessions.Get(c, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, session)
	}
}

func updateSession(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.UpdateSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		session, err := h.Sessions.Update(c, c.Param("id"), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, session)
	}
}

func deleteSession(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.Sessions.Delete(c, c.Param("id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func listTurns(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		turns := h.Sessions.ListTurns(c.Param("id"))
		c.JSON(http.StatusOK, turns)
	}
}

func resumeTurn(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		h.Sessions.ResumeTurn(c, c.Param("id"), c.Param("turnID"))
		c.JSON(http.StatusOK, gin.H{"status": "resumed"})
	}
}

func streamEvents(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")
		ch := h.Sessions.Subscribe(sessionID)
		defer h.Sessions.Unsubscribe(sessionID, ch)
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Stream(func(w io.Writer) bool {
			ev, ok := <-ch
			if !ok {
				return false
			}
			data, _ := json.Marshal(ev)
			c.SSEvent("message", string(data))
			return true
		})
	}
}

func decideApproval(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Approved bool `json:"approved"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := h.Sessions.DecideApproval(c, c.Param("id"), req.Approved); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func resolveAskUser(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Answer string `json:"answer"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Answer == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "answer required"})
			return
		}
		h.Sessions.ResolveAskUser(c.Param("id"), req.Answer)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func pollEvents(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		since, _ := strconv.ParseInt(c.DefaultQuery("since", "0"), 10, 64)
		events := h.Sessions.StreamEvents(c.Param("id"), since)
		c.JSON(http.StatusOK, events)
	}
}

func listAgents(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		agents, err := h.Agents.List(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, agents)
	}
}

func createAgent(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var a domain.Agent
		if err := c.ShouldBindJSON(&a); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if a.ID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "agent id required"})
			return
		}
		if a.Mode == "" {
			a.Mode = domain.AgentModePrimary
		}
		if err := h.Agents.Upsert(c, a); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, a)
	}
}

func getAgent(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		agent, err := h.Agents.Get(c, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, agent)
	}
}

func updateAgent(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var a domain.Agent
		if err := c.ShouldBindJSON(&a); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		a.ID = id
		if a.Mode == "" {
			a.Mode = domain.AgentModePrimary
		}
		if err := h.Agents.Upsert(c, a); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, a)
	}
}

func deleteAgent(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.Agents.Delete(c, c.Param("id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func resetAgent(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		agent, err := h.Agents.ResetFromTemplate(c, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, agent)
	}
}

func sendMessage(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.SendMessageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		modelID := req.ModelID
		if modelID == "" {
			session, err := h.Sessions.Get(c, c.Param("id"))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			modelID = session.ModelID
		}
		if cfgs, _ := h.LLMConfig.GetAll(c); len(cfgs) > 0 {
			if _, _, err := h.LLMConfig.ResolveProvider(c, modelID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
		turnID, _ := h.Sessions.StartTurn(c, c.Param("id"), req)
		c.JSON(http.StatusOK, gin.H{"turnId": turnID})
	}
}

func cancelTurn(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		h.Sessions.CancelTurn(c, c.Param("turnID"))
		c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
	}
}

func downloadTurnLog(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")
		turnID := c.Param("turnID")
		events := h.Sessions.StreamEvents(sessionID, 0)
		data, err := h.TurnLogs.LoadTurnLogZip(turnID, events)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "turn log not found"})
			return
		}
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", turnID))
		c.Header("Content-Type", "application/zip")
		c.Data(http.StatusOK, "application/zip", data)
	}
}

func createProject(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.CreateProjectRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		p, err := h.Projects.Create(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, p)
	}
}

func listProjects(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		projects, err := h.Projects.List(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, projects)
	}
}

func getProject(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := h.Projects.Get(c, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if p.ID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

func updateProject(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.UpdateProjectRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		p, err := h.Projects.Update(c, c.Param("id"), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

func deleteProject(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.Projects.Delete(c, c.Param("id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func listProjectSessions(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessions, err := h.Projects.SessionsForProject(c, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, sessions)
	}
}

func listLLMModels(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		models := h.LLMConfig.ListModels(c)
		c.JSON(http.StatusOK, models)
	}
}

func getLLMPresets(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := h.Config.Get(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, cfg.LLM.Providers)
	}
}

func getLLMConfigs(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfgs, err := h.LLMConfig.GetAll(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, cfgs)
	}
}

func createLLMConfig(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.UpsertLLMProviderConfigRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cfg, err := h.LLMConfig.Upsert(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, cfg)
	}
}

func updateLLMConfig(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.UpsertLLMProviderConfigRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cfg, err := h.LLMConfig.Upsert(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, cfg)
	}
}

func deleteLLMConfig(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.LLMConfig.Delete(c, c.Param("id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func fetchLLMModelsFromRequest(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.UpsertLLMProviderConfigRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		models, err := h.LLMConfig.FetchModelsFromRequest(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"models": models})
	}
}

func refreshLLMModels(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		models, err := h.LLMConfig.FetchModels(c, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"models": models})
	}
}

func toggleLLMModel(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cfg, err := h.LLMConfig.ToggleModel(c, c.Param("id"), c.Param("modelName"), req.Enabled)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, cfg)
	}
}

func getSearchConfig(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := h.SearchConfig.Get(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, cfg)
	}
}

func updateSearchConfig(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.UpsertSearchConfigRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cfg, err := h.SearchConfig.Upsert(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, cfg)
	}
}

func getConfig(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := h.Config.Get(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, cfg)
	}
}

func updateConfig(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.UpdateConfigFileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cfg, err := h.Config.Update(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, cfg)
	}
}

func listProjectFiles(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Query("path")
		nodes, err := h.Projects.ListFiles(c, c.Param("id"), path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, nodes)
	}
}

func readProjectFile(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
			return
		}

		// Raw mode: serve file directly with Content-Type header
		if c.Query("raw") == "true" {
			data, ct, err := h.Projects.ReadFileRaw(c, c.Param("id"), path)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// Strip charset param to avoid mime type parsing issues
			if i := strings.Index(ct, ";"); i != -1 {
				ct = ct[:i]
			}
			c.Header("Content-Type", ct)
			c.Data(http.StatusOK, ct, data)
			return
		}

		fc, err := h.Projects.ReadFileContent(c, c.Param("id"), path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, fc)
	}
}

// dqInspectScript is injected into HTML responses so the browser tab can select elements.
const dqInspectScript = `<script>(function(){var o=document.createElement('div');o.id='__dq_inspect_overlay';o.style.cssText='position:fixed;pointer-events:none;z-index:99999;border:2px solid #3b82f6;background:rgba(59,130,246,0.1);transition:all 0.08s ease;display:none';document.body.appendChild(o);function u(e){var el=e.target;if(!el||el===document.body||el===document.documentElement||el.id==='__dq_inspect_overlay')return;var r=el.getBoundingClientRect();o.style.display='block';o.style.top=r.top+'px';o.style.left=r.left+'px';o.style.width=r.width+'px';o.style.height=r.height+'px'}function s(e){e.preventDefault();e.stopPropagation();var el=e.target;if(!el||el===document.body||el===document.documentElement)return;var h=el.outerHTML.length>2000?el.outerHTML.slice(0,2000)+'...':el.outerHTML;var t=(el.textContent||'').trim().slice(0,2000);window.parent.postMessage({type:'dq-inspect-selected',html:h,text:t,tag:el.tagName.toLowerCase()},'*');c()}function c(){document.removeEventListener('mouseover',u,true);document.removeEventListener('click',s,true);document.body.style.cursor='';o.remove()}window.addEventListener('message',function(e){if(e.data&&e.data.type==='dq-inspect-start'){document.body.style.cursor='crosshair';document.addEventListener('mouseover',u,true);document.addEventListener('click',s,true)}})})()</script>`

// serveProjectFile serves a project file directly at /projects/:id/files/:filepath
func serveProjectFile(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		filepath := strings.TrimPrefix(c.Param("filepath"), "/")
		if filepath == "" || filepath == "content" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
			return
		}
		data, ct, err := h.Projects.ReadFileRaw(c, c.Param("id"), filepath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if i := strings.Index(ct, ";"); i != -1 {
			ct = ct[:i]
		}
		// Render .md to HTML
		ext := strings.ToLower(filepath[strings.LastIndex(filepath, "."):])
		if ext == ".md" || ext == ".markdown" {
			var buf bytes.Buffer
			goldmark.New(goldmark.WithExtensions(extension.Table), goldmark.WithRendererOptions(goldmarkHtml.WithUnsafe())).Convert(data, &buf)
			html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><style>body{background:#fff;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif;font-size:16px;line-height:1.6;color:#24292f;max-width:900px;margin:0 auto;padding:32px}h1,h2{padding-bottom:.3em;border-bottom:1px solid #d0d7de}h1{font-size:2em}h2{font-size:1.5em}pre{background:#f6f8fa;padding:16px;border-radius:6px;overflow:auto;font-size:85%%}code{font-family:"SF Mono",Monaco,monospace;font-size:85%%;background:#f6f8fa;padding:.2em .4em;border-radius:3px}pre code{background:none;padding:0}table{border-collapse:collapse;width:100%%}td,th{border:1px solid #d0d7de;padding:6px 13px}th{background:#f6f8fa}tr:nth-child(2n){background:#f6f8fa}blockquote{border-left:4px solid #d0d7de;padding:0 1em;color:#57606a}img{max-width:100%%}a{color:#0969da}ul,ol{padding-left:2em}</style></head><body>%s</body></html>`, buf.String())
			c.Data(http.StatusOK, "text/html", []byte(html + dqInspectScript))
			return
		}
		// Inject inspect listener into HTML responses
		if ct == "text/html" {
			data = append(data, []byte(dqInspectScript)...)
		}
		c.Header("Content-Type", ct)
		c.Data(http.StatusOK, ct, data)
	}
}

// proxyDevServer proxies external dev servers (e.g. localhost:3000) and injects the inspect script.
func proxyDevServer(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimPrefix(c.Param("target"), "/")
		if raw == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target is required"})
			return
		}
		// Convert localhost-3000/path to http://localhost:3000/path
		// Replace first dash after the host portion with colon
		parts := strings.SplitN(raw, "/", 2)
		host := strings.Replace(parts[0], "-", ":", 1) // localhost-3000 → localhost:3000
		path := ""
		if len(parts) > 1 {
			path = "/" + parts[1]
		}
		targetURL := "http://" + host + path
		if q := c.Request.URL.RawQuery; q != "" {
			targetURL += "?" + q
		}

		resp, err := http.Get(targetURL)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		ct := resp.Header.Get("Content-Type")
		if i := strings.Index(ct, ";"); i != -1 {
			ct = ct[:i]
		}

		// Inject <base> tag and inspect script into HTML
		if ct == "text/html" {
			baseTag := fmt.Sprintf(`<base href="http://%s/">`, host)
			body = bytes.Replace(body, []byte("<head>"), []byte("<head>"+baseTag), 1)
			body = append(body, []byte(dqInspectScript)...)
		}

		for k, vs := range resp.Header {
			for _, v := range vs {
				c.Header(k, v)
			}
		}
		c.Header("Content-Type", ct)
		c.Data(resp.StatusCode, ct, body)
	}
}

func getProjectGitChanges(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		changes, err := h.Projects.GetGitChanges(c, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, changes)
	}
}

// ---- MCP Servers ----

func listMCPServers(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		servers, err := h.MCPServers.List(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, servers)
	}
}

func createMCPServer(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.UpsertMCPServerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		s, err := h.MCPServers.Create(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, s)
	}
}

func getMCPServer(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, err := h.MCPServers.Get(c, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, s)
	}
}

func updateMCPServer(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.UpsertMCPServerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		s, err := h.MCPServers.Update(c, c.Param("id"), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, s)
	}
}

func deleteMCPServer(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.MCPServers.Delete(c, c.Param("id")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func refreshMCPTools(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		tools, err := h.MCPServers.RefreshTools(c, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tools": tools})
	}
}

func toggleMCPTool(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		srv, err := h.MCPServers.ToggleTool(c, c.Param("id"), c.Param("toolName"), req.Enabled)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, srv)
	}
}

// ---- Model Configs ----

func getModelConfigs(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := h.Config.Get(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		models := cfg.LLM.Models
		if models == nil {
			models = []domain.ModelConfig{}
		}
		c.JSON(http.StatusOK, models)
	}
}

func updateModelConfigs(h *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var models []domain.ModelConfig
		if err := c.ShouldBindJSON(&models); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Validate entries
		for i, m := range models {
			if m.Model == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("model name required at index %d", i)})
				return
			}
		}
		cfg, err := h.Config.Get(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cfg.LLM.Models = models
		if _, err := h.Config.Update(c, domain.UpdateConfigFileRequest{LLM: &cfg.LLM}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, models)
	}
}

// ---- Version ----

func getVersion() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": Version})
	}
}
