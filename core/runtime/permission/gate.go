package permission

import (
	"path/filepath"
	"strings"
	"danqing-teams/core/domain"
)

type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionAsk   Decision = "ask"
	DecisionDeny  Decision = "deny"
)

type Gate struct {
	Rules []domain.PermissionRule
}

func NewGate(rules []domain.PermissionRule) *Gate {
	return &Gate{Rules: rules}
}

func (g *Gate) Check(toolName string, risk domain.RiskLevel) Decision {
	if risk == domain.RiskHigh {
		return DecisionAsk
	}
	for _, r := range g.Rules {
		if matchPattern(r.Pattern, toolName) {
			if r.Action == domain.PermDeny {
				return DecisionDeny
			}
			if r.Action == domain.PermAsk {
				return DecisionAsk
			}
		}
	}
	return DecisionAllow
}

func matchPattern(pattern, name string) bool {
	if pattern == "*" {
		return true
	}
	ok, _ := filepath.Match(pattern, name)
	return ok || strings.Contains(name, strings.Trim(pattern, "*"))
}
