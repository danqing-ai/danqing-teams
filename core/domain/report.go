package domain

type ReportStatus string

const (
	ReportDone    ReportStatus = "done"
	ReportBlocked ReportStatus = "blocked"
	ReportFailed  ReportStatus = "failed"
)

type Report struct {
	Status     ReportStatus `json:"status"`
	Summary    string       `json:"summary"`
	Confidence float64      `json:"confidence"`
	StepsUsed  int          `json:"stepsUsed"`
	TraceID    string       `json:"traceId,omitempty"`
	WorkerID   string       `json:"workerId,omitempty"`
	WorkerName string       `json:"workerName,omitempty"`
}
