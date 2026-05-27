package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"danqing-teams/internal/api/rest/dto"
	"danqing-teams/internal/application/assembler"
	"danqing-teams/pkg/id"
)

func (h *Controller) UpdateTeam(c *gin.Context) {
	var req dto.UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	team, err := h.Teams.Update(c.Request.Context(), c.Param("teamId"), assembler.FromUpdateTeamRequest(req))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToTeam(*team))
}

func (h *Controller) DeleteTeam(c *gin.Context) {
	if err := h.Teams.Delete(c.Request.Context(), c.Param("teamId")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Controller) GetController(c *gin.Context) {
	ctrl, err := h.Teams.GetController(c.Request.Context(), c.Param("teamId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToTeamController(ctrl))
}

func (h *Controller) PutController(c *gin.Context) {
	var req dto.TeamController
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m := assembler.FromTeamController(req)
	if err := h.Teams.UpdateController(c.Request.Context(), c.Param("teamId"), m); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Controller) PostWorker(c *gin.Context) {
	var req dto.UpsertWorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	w, err := h.Teams.UpsertWorker(c.Request.Context(), c.Param("teamId"), assembler.FromUpsertWorkerRequest(req), "")
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, assembler.ToWorkerAgent(w))
}

func (h *Controller) PatchWorker(c *gin.Context) {
	var req dto.UpsertWorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	w, err := h.Teams.UpsertWorker(c.Request.Context(), c.Param("teamId"), assembler.FromUpsertWorkerRequest(req), c.Param("workerId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToWorkerAgent(w))
}

func (h *Controller) DeleteWorker(c *gin.Context) {
	if err := h.Teams.DeleteWorker(c.Request.Context(), c.Param("teamId"), c.Param("workerId")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Controller) ListHumans(c *gin.Context) {
	list, err := h.Teams.ListHumans(c.Request.Context(), c.Param("teamId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToHumanMembers(list))
}

func (h *Controller) PostHuman(c *gin.Context) {
	var req dto.HumanMember
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m := assembler.FromHumanMember(req)
	if m.ID == "" {
		m.ID = id.New()
	}
	if err := h.Teams.AddHuman(c.Request.Context(), c.Param("teamId"), m); err != nil {
		writeError(c, err)
		return
	}
	req.ID = m.ID
	c.JSON(http.StatusCreated, req)
}

func (h *Controller) GetApproval(c *gin.Context) {
	a, err := h.Approvals.Get(c.Request.Context(), c.Param("teamId"), c.Param("approvalId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToApprovalRequest(a))
}

func (h *Controller) CancelTask(c *gin.Context) {
	if err := h.Tasks.Cancel(c.Request.Context(), c.Param("teamId"), c.Param("taskId")); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

func (h *Controller) ListTodos(c *gin.Context) {
	list, err := h.Todos.List(c.Request.Context(), c.Param("teamId"), c.Query("taskId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToTodoItems(list))
}

func (h *Controller) PostTodo(c *gin.Context) {
	var req struct {
		Title  string `json:"title"`
		TaskID string `json:"taskId,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.Todos.Create(c.Request.Context(), c.Param("teamId"), req.Title, req.TaskID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, assembler.ToTodoItem(item))
}

func (h *Controller) PatchTodo(c *gin.Context) {
	var req struct {
		Done bool `json:"done"`
	}
	_ = c.ShouldBindJSON(&req)
	item, err := h.Todos.Update(c.Request.Context(), c.Param("teamId"), c.Param("todoId"), req.Done)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToTodoItem(item))
}

func (h *Controller) ListWorkspace(c *gin.Context) {
	list, err := h.Workspace.ListArtifacts(c.Request.Context(), c.Param("teamId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToWorkspaceArtifacts(list))
}

func (h *Controller) PostWorkspace(c *gin.Context) {
	var req dto.CreateArtifactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.Workspace.CreateArtifact(c.Request.Context(), c.Param("teamId"), assembler.FromCreateArtifactRequest(req))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, assembler.ToWorkspaceArtifact(a))
}

func (h *Controller) ListKnowledge(c *gin.Context) {
	docs, err := h.Workspace.ListKnowledgeDocs(c.Request.Context(), c.Param("teamId"), c.Param("workerId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToKnowledgeDocs(docs))
}

func (h *Controller) PutKnowledge(c *gin.Context) {
	var req dto.UpsertKnowledgeDocsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Workspace.SaveKnowledgeDocs(c.Request.Context(), c.Param("teamId"), c.Param("workerId"), assembler.FromUpsertKnowledgeDocsRequest(req)); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, req.Docs)
}
