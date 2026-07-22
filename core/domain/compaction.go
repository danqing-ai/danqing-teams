package domain

type CompactionCheckpoint struct {
	SessionID     string               `json:"sessionId"`
	TurnID        string               `json:"turnId"`
	Summary       string               `json:"summary"`
	Todos         []CompactionTodoItem `json:"todos,omitempty"`
	TurnCount     int                  `json:"turnCount"`
	TokenEstimate int                  `json:"tokenEstimate"`
	// RetainFromTurnID is the first turn id to replay after compaction.
	// Turns strictly before this id are replaced by Summary in the system prompt.
	RetainFromTurnID string `json:"retainFromTurnId,omitempty"`
	// RetainSkipMessages skips this many leading reconstructed messages inside
	// RetainFromTurnID (0 = from the start of that turn). Enables mid-turn cuts
	// at tool-pair / message-block boundaries.
	RetainSkipMessages int `json:"retainSkipMessages,omitempty"`
}

// CompactionTodoItem is a structured todowrite entry preserved across compaction.
type CompactionTodoItem struct {
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

type CompactionConfig struct {
	Enabled      bool    `json:"enabled"`
	Model        string  `json:"model"`
	MaxTokens    int     `json:"maxTokens"`
	TriggerRatio float64 `json:"triggerRatio"`
	CutTokens    int     `json:"cutTokens"`
	TurnInterval int     `json:"turnInterval"`
	SubInterval  int     `json:"subInterval"`
	ToolTruncate int     `json:"toolTruncate"`
}
