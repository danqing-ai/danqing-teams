package domain

type CompactionCheckpoint struct {
	SessionID       string               `json:"sessionId"`
	TurnID          string               `json:"turnId"`
	Summary         string               `json:"summary"`
	WorkState       CompactionWorkState  `json:"workState"`
	Decisions       []string             `json:"decisions"`
	NextMove        string               `json:"nextMove"`
	CriticalCtx     []string             `json:"criticalContext"`
	AgentsInvolved  []string             `json:"agentsInvolved"`
	FilesTouched    []string             `json:"filesTouched"`
	Todos           []CompactionTodoItem `json:"todos,omitempty"`
	TurnCount       int                  `json:"turnCount"`
	TokenEstimate   int                  `json:"tokenEstimate"`
}

// CompactionTodoItem is a structured todowrite entry preserved across compaction.
type CompactionTodoItem struct {
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

type CompactionWorkState struct {
	Completed []string `json:"completed"`
	Active    []string `json:"active"`
	Blocked   []string `json:"blocked"`
}

type CompactionConfig struct {
	Enabled       bool    `json:"enabled"`
	Model         string  `json:"model"`
	MaxTokens     int     `json:"maxTokens"`
	TriggerRatio  float64 `json:"triggerRatio"`
	CutTokens     int     `json:"cutTokens"`
	TurnInterval  int     `json:"turnInterval"`
	SubInterval   int     `json:"subInterval"`
	ToolTruncate  int     `json:"toolTruncate"`
}
