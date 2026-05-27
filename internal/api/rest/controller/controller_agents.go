package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"danqing-teams/internal/api/rest/dto"
	"danqing-teams/internal/application/assembler"
	"danqing-teams/internal/domain/model"
)

func (h *Controller) ListAgents(c *gin.Context) {
	role := model.AgentRole(c.Query("role"))
	list, err := h.Agents.List(c.Request.Context(), role)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToAgents(list))
}

func (h *Controller) GetAgent(c *gin.Context) {
	a, err := h.Agents.Get(c.Request.Context(), c.Param("agentId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToAgent(a))
}

func (h *Controller) PostAgent(c *gin.Context) {
	var req dto.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.Agents.Create(c.Request.Context(), assembler.FromCreateAgentRequest(req))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, assembler.ToAgent(a))
}

func (h *Controller) PatchAgent(c *gin.Context) {
	var req dto.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.Agents.Update(c.Request.Context(), c.Param("agentId"), assembler.FromUpdateAgentRequest(req))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToAgent(a))
}

func (h *Controller) DeleteAgent(c *gin.Context) {
	if err := h.Agents.Delete(c.Request.Context(), c.Param("agentId")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Controller) ListTeamAgentMembers(c *gin.Context) {
	list, err := h.Agents.ListTeamMembers(c.Request.Context(), c.Param("teamId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, assembler.ToAgents(list))
}

func (h *Controller) PostTeamAgentMember(c *gin.Context) {
	if err := h.Agents.AddToTeam(c.Request.Context(), c.Param("teamId"), c.Param("agentId")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Controller) DeleteTeamAgentMember(c *gin.Context) {
	if err := h.Agents.RemoveFromTeam(c.Request.Context(), c.Param("teamId"), c.Param("agentId")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
