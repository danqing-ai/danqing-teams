package domain

type SkillFile struct {
	ID      string `json:"id"`
	SkillID string `json:"skillId"`
	Path    string `json:"path"`
	Content []byte `json:"content"`
	Size    int64  `json:"size"`
}
