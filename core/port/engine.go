package port

import (
	"context"
	"danqing-teams/core/domain"
)

type Engine interface {
	StartSession(ctx context.Context, s domain.Session)
	StartTurn(ctx context.Context, sessionID, userInput, agentID, modelID string) (string, error)
	ResumeTurn(ctx context.Context, sessionID, turnID string)
	CancelTurn(ctx context.Context, turnID string)
	ListTurns(sessionID string) []domain.TurnLog

	StreamEvents(sessionID string, since int64) []domain.StreamEvent
	Subscribe(sessionID string) chan domain.StreamEvent
	Unsubscribe(sessionID string, ch chan domain.StreamEvent)
	ResolveApproval(id string, approved bool)
	ResolveAskUser(askID, answer string)
}
