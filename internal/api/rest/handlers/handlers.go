package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"danqing-teams/internal/contract"
	"danqing-teams/internal/service"
	"danqing-teams/pkg/errs"
)

type Handlers struct {
	Teams     *service.TeamService
	Tasks     *service.TaskService
	Approvals *service.ApprovalService
	Todos     *service.TodoService
	Workspace *service.WorkspaceService
	Agents    *service.AgentService
}

func writeError(c *gin.Context, err error) {
	var app *errs.AppError
	if errors.As(err, &app) {
		switch {
		case errors.Is(app.Err, errs.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": app.Message})
		case errors.Is(app.Err, errs.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": app.Message})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": app.Message})
		}
		return
	}
	if errors.Is(err, errs.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) ListTeams(c *gin.Context) {
	list, err := h.Teams.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handlers) CreateTeam(c *gin.Context) {
	var req contract.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	team, err := h.Teams.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, team)
}

func (h *Handlers) GetTeam(c *gin.Context) {
	controllerView := c.Query("view") == "controller"
	team, err := h.Teams.Get(c.Request.Context(), c.Param("teamId"), controllerView)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, team)
}

func (h *Handlers) ListWorkers(c *gin.Context) {
	controllerView := c.Query("view") == "controller"
	data, err := h.Teams.ListWorkers(c.Request.Context(), c.Param("teamId"), controllerView)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handlers) SubmitTask(c *gin.Context) {
	var req contract.SubmitTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task, err := h.Tasks.Submit(c.Request.Context(), c.Param("teamId"), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, task)
}

func (h *Handlers) SendTeamMessage(c *gin.Context) {
	var req contract.SendTeamMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.Tasks.SendMessage(c.Request.Context(), c.Param("teamId"), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, resp)
}

func (h *Handlers) ListTaskMessages(c *gin.Context) {
	list, err := h.Tasks.ListMessages(c.Request.Context(), c.Param("teamId"), c.Param("taskId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handlers) GetTask(c *gin.Context) {
	task, err := h.Tasks.Get(c.Request.Context(), c.Param("teamId"), c.Param("taskId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handlers) ListTasks(c *gin.Context) {
	status := contract.TaskStatus(c.Query("status"))
	list, err := h.Tasks.List(c.Request.Context(), c.Param("teamId"), status)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handlers) GetTimeline(c *gin.Context) {
	events, err := h.Tasks.Timeline(c.Request.Context(), c.Param("teamId"), c.Param("taskId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, events)
}

func (h *Handlers) GetRunPlan(c *gin.Context) {
	plan, err := h.Tasks.GetPlan(c.Request.Context(), c.Param("teamId"), c.Param("taskId"), c.Param("runId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (h *Handlers) ListReports(c *gin.Context) {
	reports, err := h.Tasks.Reports(c.Request.Context(), c.Param("teamId"), c.Param("taskId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, reports)
}

func (h *Handlers) ListApprovals(c *gin.Context) {
	list, err := h.Approvals.ListPending(c.Request.Context(), c.Param("teamId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handlers) Approve(c *gin.Context) {
	var req contract.DecideApprovalRequest
	_ = c.ShouldBindJSON(&req)
	a, err := h.Approvals.Approve(c.Request.Context(), c.Param("teamId"), c.Param("approvalId"), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handlers) Reject(c *gin.Context) {
	var req contract.DecideApprovalRequest
	_ = c.ShouldBindJSON(&req)
	a, err := h.Approvals.Reject(c.Request.Context(), c.Param("teamId"), c.Param("approvalId"), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}
