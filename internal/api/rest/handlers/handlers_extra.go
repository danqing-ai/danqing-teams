package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/id"
)

func (h *Handlers) UpdateTeam(c *gin.Context) {
	var req contract.UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	team, err := h.Teams.Update(c.Request.Context(), c.Param("teamId"), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, team)
}

func (h *Handlers) DeleteTeam(c *gin.Context) {
	if err := h.Teams.Delete(c.Request.Context(), c.Param("teamId")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) GetController(c *gin.Context) {
	ctrl, err := h.Teams.GetController(c.Request.Context(), c.Param("teamId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, ctrl)
}

func (h *Handlers) PutController(c *gin.Context) {
	var req contract.TeamController
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Teams.UpdateController(c.Request.Context(), c.Param("teamId"), req); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Handlers) PostWorker(c *gin.Context) {
	var req contract.UpsertWorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	w, err := h.Teams.UpsertWorker(c.Request.Context(), c.Param("teamId"), req, "")
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, w)
}

func (h *Handlers) PatchWorker(c *gin.Context) {
	var req contract.UpsertWorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	w, err := h.Teams.UpsertWorker(c.Request.Context(), c.Param("teamId"), req, c.Param("workerId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, w)
}

func (h *Handlers) DeleteWorker(c *gin.Context) {
	if err := h.Teams.DeleteWorker(c.Request.Context(), c.Param("teamId"), c.Param("workerId")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) ListHumans(c *gin.Context) {
	list, err := h.Teams.ListHumans(c.Request.Context(), c.Param("teamId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handlers) PostHuman(c *gin.Context) {
	var req contract.HumanMember
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ID == "" {
		req.ID = id.New()
	}
	if err := h.Teams.AddHuman(c.Request.Context(), c.Param("teamId"), req); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, req)
}

func (h *Handlers) GetApproval(c *gin.Context) {
	a, err := h.Approvals.Get(c.Request.Context(), c.Param("teamId"), c.Param("approvalId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handlers) CancelTask(c *gin.Context) {
	if err := h.Tasks.Cancel(c.Request.Context(), c.Param("teamId"), c.Param("taskId")); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

func (h *Handlers) ListTodos(c *gin.Context) {
	list, err := h.Todos.List(c.Request.Context(), c.Param("teamId"), c.Query("taskId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handlers) PostTodo(c *gin.Context) {
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
	c.JSON(http.StatusCreated, item)
}

func (h *Handlers) PatchTodo(c *gin.Context) {
	var req struct {
		Done bool `json:"done"`
	}
	_ = c.ShouldBindJSON(&req)
	item, err := h.Todos.Update(c.Request.Context(), c.Param("teamId"), c.Param("todoId"), req.Done)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handlers) ListWorkspace(c *gin.Context) {
	list, err := h.Workspace.ListArtifacts(c.Request.Context(), c.Param("teamId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handlers) PostWorkspace(c *gin.Context) {
	var req contract.CreateArtifactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.Workspace.CreateArtifact(c.Request.Context(), c.Param("teamId"), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *Handlers) ListKnowledge(c *gin.Context) {
	docs, err := h.Workspace.ListKnowledgeDocs(c.Request.Context(), c.Param("teamId"), c.Param("workerId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, docs)
}

func (h *Handlers) PutKnowledge(c *gin.Context) {
	var req contract.UpsertKnowledgeDocsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Workspace.SaveKnowledgeDocs(c.Request.Context(), c.Param("teamId"), c.Param("workerId"), req); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, req.Docs)
}
