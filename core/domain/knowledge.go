package domain

type KnowledgeDoc struct {
	ID      string `json:"id"`
	KBID    string `json:"kbId"`
	Title   string `json:"title"`
	Content string `json:"content"`
}
