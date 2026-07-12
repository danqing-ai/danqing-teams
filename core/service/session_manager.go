package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"
)

type SessionManager struct {
	store  port.Repository
	engine port.Engine
	llm    port.LLMProvider
	mu     sync.Mutex
}

func NewSessionManager(store port.Repository, engine port.Engine, llm port.LLMProvider) *SessionManager {
	return &SessionManager{store: store, engine: engine, llm: llm}
}

func (m *SessionManager) SetEngine(engine port.Engine) {
	m.engine = engine
}

func (m *SessionManager) Create(ctx context.Context, req domain.CreateSessionRequest) (domain.Session, error) {
	if req.Content == "" {
		return domain.Session{}, fmt.Errorf("content required")
	}
	if req.AgentID == "" {
		return domain.Session{}, fmt.Errorf("agentId required")
	}
	now := time.Now().UTC()
	s := domain.Session{
		ID:        fmt.Sprintf("session-%d", time.Now().UnixNano()),
		ProjectID: req.ProjectID,
		AgentID:   req.AgentID,
		ModelID:   req.ModelID,
		Content:   req.Content,
		Status:    domain.SessionStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := m.store.Sessions().Create(ctx, s); err != nil {
		return domain.Session{}, err
	}
	m.engine.StartSession(ctx, s)
	go m.generateTitle(s.ID, s.Content, s.ModelID)
	return s, nil
}

func (m *SessionManager) generateTitle(sessionID, content, modelID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := m.llm.Chat(ctx, port.LLMChatRequest{
		Model: modelID,
		Messages: []port.ChatMessage{
			{Role: "system", Content: "You are a session title generator. Generate a concise title (max 8 words) for the given session description. Reply with ONLY the title, no extra text."},
			{Role: "user", Content: content},
		},
	})
	if err != nil {
		log.Printf("generate title for session %s: %v", sessionID, err)
		return
	}
	title := strings.TrimSpace(resp.Content)
	if title == "" {
		return
	}
	s, err := m.store.Sessions().Get(ctx, sessionID)
	if err != nil {
		log.Printf("generate title: get session %s: %v", sessionID, err)
		return
	}
	s.Title = title
	if err := m.store.Sessions().Update(ctx, s); err != nil {
		log.Printf("generate title: update session %s: %v", sessionID, err)
	}
}

func (m *SessionManager) StartTurn(ctx context.Context, sessionID string, req domain.SendMessageRequest) (string, error) {
	return m.engine.StartTurn(ctx, sessionID, req.UserInput, req.AgentID, req.ModelID)
}

func (m *SessionManager) CancelTurn(ctx context.Context, turnID string) {
	m.engine.CancelTurn(ctx, turnID)
}

func (m *SessionManager) ResumeTurn(ctx context.Context, sessionID, turnID string) {
	m.engine.ResumeTurn(ctx, sessionID, turnID)
}

func (m *SessionManager) ListTurns(sessionID string) []domain.TurnLog {
	return m.engine.ListTurns(sessionID)
}

func (m *SessionManager) Get(ctx context.Context, id string) (domain.Session, error) {
	return m.store.Sessions().Get(ctx, id)
}

func (m *SessionManager) List(ctx context.Context) ([]domain.Session, error) {
	return m.store.Sessions().List(ctx)
}

func (m *SessionManager) Update(ctx context.Context, id string, req domain.UpdateSessionRequest) (domain.Session, error) {
	s, err := m.store.Sessions().Get(ctx, id)
	if err != nil {
		return domain.Session{}, err
	}
	if req.Title != nil {
		s.Title = *req.Title
	}
	if req.ProjectID != nil {
		s.ProjectID = *req.ProjectID
	}
	if req.Status != nil {
		s.Status = *req.Status
	}
	if req.ModelID != nil {
		s.ModelID = *req.ModelID
	}
	if req.AgentID != nil {
		s.AgentID = *req.AgentID
	}
	s.UpdatedAt = time.Now().UTC()
	if err := m.store.Sessions().Update(ctx, s); err != nil {
		return domain.Session{}, err
	}
	return s, nil
}

func (m *SessionManager) UpdateSession(ctx context.Context, s domain.Session) error {
	return m.store.Sessions().Update(ctx, s)
}

func (m *SessionManager) Delete(ctx context.Context, id string) error {
	return m.store.Sessions().Delete(ctx, id)
}

func (m *SessionManager) StreamEvents(sessionID string, since int64) []domain.StreamEvent {
	return m.engine.StreamEvents(sessionID, since)
}

func (m *SessionManager) DecideApproval(ctx context.Context, id string, approved bool) error {
	a, err := m.store.Approvals().Get(ctx, id)
	if err != nil {
		return err
	}
	if approved {
		a.Status = "approved"
	} else {
		a.Status = "rejected"
	}
	if err := m.store.Approvals().Update(ctx, a); err != nil {
		return err
	}
	m.engine.ResolveApproval(id, approved)
	return nil
}

func (m *SessionManager) Subscribe(sessionID string) chan domain.StreamEvent {
	return m.engine.Subscribe(sessionID)
}

func (m *SessionManager) Unsubscribe(sessionID string, ch chan domain.StreamEvent) {
	m.engine.Unsubscribe(sessionID, ch)
}

func (m *SessionManager) ResolveAskUser(askID, answer string) {
	m.engine.ResolveAskUser(askID, answer)
}
