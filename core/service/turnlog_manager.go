package service

import (
	"danqing-teams/core/domain"
	"danqing-teams/core/port"
)

type TurnLogManager struct {
	store port.TurnLogStore
}

func NewTurnLogManager(store port.TurnLogStore) *TurnLogManager {
	return &TurnLogManager{store: store}
}

func (m *TurnLogManager) Create(turnID, sessionID, projectID, agentID, goal string) error {
	return m.store.Create(turnID, sessionID, projectID, agentID, goal)
}

// Append writes a tool_call or tool_result entry to the turn log.
// Only these two types are consumed by LoadForRecovery for LLM message
// reconstruction. Do NOT write diagnostic entries (llm_error, step events,
// permission decisions) — those belong in Stream Events.
func (m *TurnLogManager) Append(turnID, typ string, data map[string]any) {
	m.store.Append(turnID, typ, data)
}

func (m *TurnLogManager) EndTurn(turnID string, status domain.TurnStatus) {
	m.store.EndTurn(turnID, status)
}

func (m *TurnLogManager) LastStatus(sessionID string) domain.TurnStatus {
	return m.store.LastStatus(sessionID)
}

func (m *TurnLogManager) ListTurns(sessionID string) []domain.TurnLog {
	return m.store.ListTurns(sessionID)
}

func (m *TurnLogManager) LoadForRecovery(turnID string) (goal string, entries []map[string]any) {
	return m.store.LoadForRecovery(turnID)
}

func (m *TurnLogManager) LoadRawLog(turnID string) ([]byte, error) {
	return m.store.LoadRawLog(turnID)
}

func (m *TurnLogManager) LoadTurnLogZip(turnID string, events []domain.StreamEvent) ([]byte, error) {
	return m.store.LoadTurnLogZip(turnID, events)
}
