package mcp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, tools *Tools) {
	r.POST("/tools/call", func(c *gin.Context) {
		var req CallRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := tools.Call(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	})
	r.GET("/tools", func(c *gin.Context) {
		c.JSON(http.StatusOK, []gin.H{
			{"name": "teams_list"},
			{"name": "teams_get"},
			{"name": "workers_upsert"},
			{"name": "task_submit"},
			{"name": "task_timeline"},
			{"name": "task_cancel"},
			{"name": "approval_list"},
			{"name": "approval_decide"},
		})
	})
}
