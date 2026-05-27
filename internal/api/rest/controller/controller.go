package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"danqing-teams/internal/api/rest/dto"
	"danqing-teams/internal/application/assembler"
	"danqing-teams/internal/application/port"
	"danqing-teams/internal/domain/model"
	"danqing-teams/pkg/errs"
)

type Controller struct {
	Teams     port.TeamService
	Tasks     port.TaskService
	Approvals port.ApprovalService
	Todos     port.TodoService
	Workspace port.WorkspaceService
	Agents    port.AgentService
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

func (h *Controller) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Controller) ListTeams(c *gin.Context) {
	list, err := h.Teams.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToTeams(list))
}

func (h *Controller) CreateTeam(c *gin.Context) {
	var req dto.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	team, err := h.Teams.Create(c.Request.Context(), assembler.FromCreateTeamRequest(req))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, assembler.ToTeamDetail(team))
}

func (h *Controller) GetTeam(c *gin.Context) {
	controllerView := c.Query("view") == "controller"
	team, err := h.Teams.Get(c.Request.Context(), c.Param("teamId"), controllerView)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToTeamDetail(team))
}

func (h *Controller) ListWorkers(c *gin.Context) {
	controllerView := c.Query("view") == "controller"
	data, err := h.Teams.ListWorkers(c.Request.Context(), c.Param("teamId"), controllerView)
	if err != nil {
		writeError(c, err)
		return
	}
	switch v := data.(type) {
	case []model.WorkerPersonaCatalog:
		c.JSON(http.StatusOK, assembler.ToPersonaCatalog(v))
	case []model.WorkerAgent:
		c.JSON(http.StatusOK, assembler.ToWorkerAgents(v))
	default:
		c.JSON(http.StatusOK, data)
	}
}

func (h *Controller) SubmitTask(c *gin.Context) {
	var req dto.SubmitTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task, err := h.Tasks.Submit(c.Request.Context(), c.Param("teamId"), assembler.FromSubmitTaskRequest(req))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, assembler.ToTeamTask(task))
}

func (h *Controller) SendTeamMessage(c *gin.Context) {
	var req dto.SendTeamMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.Tasks.SendMessage(c.Request.Context(), c.Param("teamId"), assembler.FromSendTeamMessageRequest(req))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, assembler.ToSendTeamMessageResponse(resp))
}

func (h *Controller) ListTaskMessages(c *gin.Context) {
	list, err := h.Tasks.ListMessages(c.Request.Context(), c.Param("teamId"), c.Param("taskId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToTeamMessages(list))
}

func (h *Controller) GetTask(c *gin.Context) {
	task, err := h.Tasks.Get(c.Request.Context(), c.Param("teamId"), c.Param("taskId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToTeamTask(task))
}

func (h *Controller) ListTasks(c *gin.Context) {
	status := model.TaskStatus(c.Query("status"))
	list, err := h.Tasks.List(c.Request.Context(), c.Param("teamId"), status)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToTeamTasks(list))
}

func (h *Controller) GetTimeline(c *gin.Context) {
	events, err := h.Tasks.Timeline(c.Request.Context(), c.Param("teamId"), c.Param("taskId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToTimelineEvents(events))
}

func (h *Controller) GetRunPlan(c *gin.Context) {
	plan, err := h.Tasks.GetPlan(c.Request.Context(), c.Param("teamId"), c.Param("taskId"), c.Param("runId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToExecutionPlan(plan))
}

func (h *Controller) ListReports(c *gin.Context) {
	reports, err := h.Tasks.Reports(c.Request.Context(), c.Param("teamId"), c.Param("taskId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToReports(reports))
}

func (h *Controller) ListApprovals(c *gin.Context) {
	list, err := h.Approvals.ListPending(c.Request.Context(), c.Param("teamId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToApprovalRequests(list))
}

func (h *Controller) Approve(c *gin.Context) {
	var req dto.DecideApprovalRequest
	_ = c.ShouldBindJSON(&req)
	a, err := h.Approvals.Approve(c.Request.Context(), c.Param("teamId"), c.Param("approvalId"), assembler.FromDecideApprovalRequest(req))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToApprovalRequest(a))
}

func (h *Controller) Reject(c *gin.Context) {
	var req dto.DecideApprovalRequest
	_ = c.ShouldBindJSON(&req)
	a, err := h.Approvals.Reject(c.Request.Context(), c.Param("teamId"), c.Param("approvalId"), assembler.FromDecideApprovalRequest(req))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToApprovalRequest(a))
}
