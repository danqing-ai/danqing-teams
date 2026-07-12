package v1

import (
	"net/http"
	"strings"

	"danqing-teams/core/domain"
	"danqing-teams/core/service"

	"github.com/gin-gonic/gin"
)

type SkillHandler struct {
	Skills    *service.SkillManager
	Importer  *service.SkillImporter
}

func listSkills(h *SkillHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		skills, err := h.Skills.List(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, skills)
	}
}

func getSkill(h *SkillHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		skill, err := h.Skills.Get(c, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if skill == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "skill not found"})
			return
		}
		c.JSON(http.StatusOK, skill)
	}
}

func createSkill(h *SkillHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var sk domain.Skill
		if err := c.ShouldBindJSON(&sk); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if sk.ID == "" {
			if sk.Name == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "id or name required"})
				return
			}
			sk.ID = sk.Name
		}
		if err := h.Skills.Upsert(c, sk); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, sk)
	}
}

func updateSkill(h *SkillHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var sk domain.Skill
		if err := c.ShouldBindJSON(&sk); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sk.ID = id
		if err := h.Skills.Upsert(c, sk); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, sk)
	}
}

func deleteSkill(h *SkillHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := h.Skills.Delete(c, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := h.Skills.DeleteFiles(c, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func importSkillDir(h *SkillHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Path string `json:"path"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
			return
		}
		skill, files, err := h.Importer.Import(req.Path)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := h.Skills.Upsert(c, *skill); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, f := range files {
			if err := h.Skills.UpsertFile(c, f); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusCreated, gin.H{"skill": skill, "fileCount": len(files)})
	}
}

func exportSkillMD(h *SkillHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		skill, err := h.Skills.Get(c, c.Param("id"))
		if err != nil || skill == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "skill not found"})
			return
		}
		md := h.Importer.ToSkillMD(*skill)
		c.Header("Content-Type", "text/markdown; charset=utf-8")
		c.String(http.StatusOK, md)
	}
}

func resetSkill(h *SkillHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		skill, err := h.Skills.ResetFromTemplate(c, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, skill)
	}
}

func listSkillFiles(h *SkillHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		files, err := h.Skills.Files(c, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, files)
	}
}

func getSkillFile(h *SkillHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		fpath := strings.TrimPrefix(c.Param("path"), "/")
		f, err := h.Skills.File(c, c.Param("id"), fpath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if isTextFile(f.Path) {
			c.String(http.StatusOK, string(f.Content))
		} else {
			c.Data(http.StatusOK, "application/octet-stream", f.Content)
		}
	}
}

func isTextFile(path string) bool {
	textExt := []string{".md", ".txt", ".yaml", ".yml", ".json", ".toml", ".py", ".sh", ".js", ".ts", ".go", ".rs", ".html", ".css", ".xml", ".csv", ".ini", ".cfg"}
	lower := strings.ToLower(path)
	for _, ext := range textExt {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}
