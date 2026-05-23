package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"danqing-teams/internal/contract"
)

func (h *Handlers) ListAgents(c *gin.Context) {
	role := contract.AgentRole(c.Query("role"))
	list, err := h.Agents.List(c.Request.Context(), role)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handlers) GetAgent(c *gin.Context) {
	a, err := h.Agents.Get(c.Request.Context(), c.Param("agentId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handlers) PostAgent(c *gin.Context) {
	var req contract.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.Agents.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *Handlers) PatchAgent(c *gin.Context) {
	var req contract.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.Agents.Update(c.Request.Context(), c.Param("agentId"), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handlers) DeleteAgent(c *gin.Context) {
	if err := h.Agents.Delete(c.Request.Context(), c.Param("agentId")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) ListTeamAgentMembers(c *gin.Context) {
	list, err := h.Agents.ListTeamMembers(c.Request.Context(), c.Param("teamId"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handlers) PostTeamAgentMember(c *gin.Context) {
	if err := h.Agents.AddToTeam(c.Request.Context(), c.Param("teamId"), c.Param("agentId")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) DeleteTeamAgentMember(c *gin.Context) {
	if err := h.Agents.RemoveFromTeam(c.Request.Context(), c.Param("teamId"), c.Param("agentId")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
