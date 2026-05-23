package contract

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

func (r RiskLevel) Rank() int {
	switch r {
	case RiskHigh:
		return 3
	case RiskMedium:
		return 2
	default:
		return 1
	}
}

func MaxRisk(a, b RiskLevel) RiskLevel {
	if a.Rank() >= b.Rank() {
		return a
	}
	return b
}
