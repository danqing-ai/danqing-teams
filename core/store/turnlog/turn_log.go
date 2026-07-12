package turnlog

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"danqing-teams/core/domain"
)

type TurnLogEntry struct {
	Seq  int            `json:"seq"`
	Type string         `json:"type"`
	Data map[string]any `json:"data,omitempty"`
}

type TurnLogStore struct {
	mu              sync.Mutex
	projector       func(projectID string) string
	logs            map[string]*turnFile
	sessionProjects map[string]string
}

type turnFile struct {
	log      *domain.TurnLog
	f        *os.File
	seq      int
	filePath string
}

func NewTurnLogStore(projector func(projectID string) string) *TurnLogStore {
	return &TurnLogStore{projector: projector, logs: make(map[string]*turnFile), sessionProjects: make(map[string]string)}
}

func (s *TurnLogStore) sessionsDir(projectID string) string {
	return filepath.Join(s.projector(projectID), "sessions")
}

func (s *TurnLogStore) Create(turnID, sessionID, projectID, agentID, goal string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if projectID == "" {
		projectID = "_default"
	}
	s.sessionProjects[sessionID] = projectID

	dir := filepath.Join(s.sessionsDir(projectID), sessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%s.jsonl", turnID))
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	tf := &turnFile{
		log:      &domain.TurnLog{ID: turnID, SessionID: sessionID, Status: domain.TurnRunning, AgentID: agentID, Goal: goal},
		f:        f,
		seq:      0,
		filePath: filePath,
	}
	s.logs[turnID] = tf

	entry := TurnLogEntry{Seq: 1, Type: "start", Data: map[string]any{"agent_id": agentID, "goal": goal}}
	tf.writeEntry(entry)
	return nil
}

// Append writes a single entry to the turn's JSONL log.
//
// IMPORTANT: Only "tool_call" and "tool_result" types should be written here.
// These are the only types consumed by LoadForRecovery for LLM message
// reconstruction. All other entry types (start, end) are written by
// Create / EndTurn respectively.
//
// Do NOT call Append with diagnostic or audit types (e.g. "llm_error",
// "step", "permission_*"). Use Stream Events (port.EventStream) for those.
func (s *TurnLogStore) Append(turnID, typ string, data map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tf, ok := s.logs[turnID]
	if !ok {
		return
	}
	tf.seq++
	tf.writeEntry(TurnLogEntry{Seq: tf.seq, Type: typ, Data: data})
}

func (s *TurnLogStore) EndTurn(turnID string, status domain.TurnStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tf, ok := s.logs[turnID]
	if !ok {
		return
	}
	tf.log.Status = status
	tf.seq++
	mapStatus := map[string]any{"status": string(status)}
	tf.writeEntry(TurnLogEntry{Seq: tf.seq, Type: "end", Data: mapStatus})
	tf.f.Close()
}

func (s *TurnLogStore) LastTurn(sessionID string) (turnID string, status domain.TurnStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := s.sessionDir(sessionID)
	entries, _ := os.ReadDir(dir)
	for i := len(entries) - 1; i >= 0; i-- {
		name := entries[i].Name()
		if len(name) > 6 && name[len(name)-6:] == ".jsonl" {
			id := name[:len(name)-6]
			if tf, ok := s.logs[id]; ok {
				return tf.log.ID, tf.log.Status
			}
		}
	}
	return "", ""
}

func (s *TurnLogStore) ListTurns(sessionID string) []domain.TurnLog {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.sessionDir(sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	out := make([]domain.TurnLog, 0)
	for _, entry := range entries {
		name := entry.Name()
		if len(name) > 6 && name[len(name)-6:] == ".jsonl" {
			id := name[:len(name)-6]
			if tf, ok := s.logs[id]; ok {
				out = append(out, *tf.log)
			}
		}
	}
	return out
}

func (s *TurnLogStore) LastStatus(sessionID string) domain.TurnStatus {
	_, status := s.LastTurn(sessionID)
	return status
}

func (s *TurnLogStore) sessionDir(sessionID string) string {
	if pid, ok := s.sessionProjects[sessionID]; ok {
		return filepath.Join(s.sessionsDir(pid), sessionID)
	}
	projectIDs, _ := os.ReadDir(s.projector(""))
	for _, e := range projectIDs {
		if !e.IsDir() {
			continue
		}
		sessionsRoot := filepath.Join(s.projector(e.Name()), "sessions")
		entries, err := os.ReadDir(sessionsRoot)
		if err != nil {
			continue
		}
		for _, se := range entries {
			if se.IsDir() && se.Name() == sessionID {
				s.sessionProjects[sessionID] = e.Name()
				return filepath.Join(sessionsRoot, sessionID)
			}
		}
	}
	dir := filepath.Join(s.sessionsDir("_default"), sessionID)
	os.MkdirAll(dir, 0755)
	return dir
}

type EntryJSON struct {
	Seq  int            `json:"seq"`
	Type string         `json:"type"`
	Data map[string]any `json:"data,omitempty"`
}

func (s *TurnLogStore) LoadForRecovery(turnID string) (goal string, entries []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tf, ok := s.logs[turnID]
	if !ok {
		return "", nil
	}
	goal = tf.log.Goal

	f, err := os.Open(tf.filePath)
	if err != nil {
		return goal, nil
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var all []map[string]any
	for dec.More() {
		var e map[string]any
		if err := dec.Decode(&e); err == nil {
			all = append(all, e)
		}
	}

	// Whitelist: only tool_call / tool_result participate in recovery.
	// All other entry types are transparently skipped.
	cut := 0
	for i := len(all) - 1; i >= 0; i-- {
		typ, _ := all[i]["type"].(string)
		switch typ {
		case "tool_result":
			for j := i - 1; j >= 0; j-- {
				if t, _ := all[j]["type"].(string); t == "tool_call" {
					cut = i + 1
					break
				}
			}
			return goal, all[:cut]
		case "tool_call":
			return goal, all[:cut]
		}
		// All other types (start, end, unknown) silently skipped.
	}

	return goal, all[:cut]
}

func (s *TurnLogStore) LoadRawLog(turnID string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tf, ok := s.logs[turnID]
	if !ok {
		return nil, fmt.Errorf("turn log not found: %s", turnID)
	}

	return os.ReadFile(tf.filePath)
}

// TurnLogNode represents a turn and its delegated children for zip packaging.
type TurnLogNode struct {
	TurnID  string         `json:"turnId"`
	AgentID string         `json:"agentId,omitempty"`
	Goal    string         `json:"goal,omitempty"`
	File    string         `json:"file"`
	Children []*TurnLogNode `json:"children,omitempty"`
}

func (s *TurnLogStore) LoadTurnLogZip(turnID string, events []domain.StreamEvent) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.logs[turnID]; !ok {
		return nil, fmt.Errorf("turn log not found: %s", turnID)
	}

	// Build parent->children map from stream events (delegate.started)
	childrenOf := make(map[string][]string)
	for _, ev := range events {
		if ev.Type != domain.EventDelegateStarted {
			continue
		}
		var p domain.DelegateStartedPayload
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			continue
		}
		if p.ChildTurnID != "" {
			childrenOf[ev.TurnID] = append(childrenOf[ev.TurnID], p.ChildTurnID)
		}
	}

	// Collect all related turns recursively
	files := make(map[string]string)
	var collect func(id string) *TurnLogNode
	collect = func(id string) *TurnLogNode {
		tf, exists := s.logs[id]
		if !exists {
			return nil
		}
		files[id] = tf.filePath
		node := &TurnLogNode{
			TurnID:  id,
			AgentID: tf.log.AgentID,
			Goal:    tf.log.Goal,
			File:    id + ".jsonl",
		}
		for _, childID := range childrenOf[id] {
			if child := collect(childID); child != nil {
				node.Children = append(node.Children, child)
			}
		}
		return node
	}

	rootNode := collect(turnID)
	var nodes []*TurnLogNode
	if rootNode != nil {
		nodes = append(nodes, rootNode)
	}

	// Build zip
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	manifest, _ := json.MarshalIndent(nodes, "", "  ")
	w, _ := zw.Create("manifest.json")
	w.Write(manifest)

	for id, fp := range files {
		data, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		w, _ := zw.Create(id + ".jsonl")
		w.Write(data)
	}

	zw.Close()
	return buf.Bytes(), nil
}

func (s *TurnLogStore) ListEntries(turnID string) []EntryJSON {
	s.mu.Lock()
	defer s.mu.Unlock()

	tf, ok := s.logs[turnID]
	if !ok {
		return nil
	}

	f, err := os.Open(tf.filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	entries := make([]EntryJSON, 0)
	for dec.More() {
		var e EntryJSON
		if err := dec.Decode(&e); err == nil {
			entries = append(entries, e)
		}
	}
	return entries
}

func LoadTurnLog(projector func(projectID string) string, projectID, sessionID string) ([]EntryJSON, error) {
	dir := filepath.Join(projector(projectID), "sessions", sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var all []EntryJSON
	for _, entry := range entries {
		name := entry.Name()
		if len(name) > 6 && name[len(name)-6:] == ".jsonl" {
			f, err := os.Open(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			dec := json.NewDecoder(f)
			for dec.More() {
				var e EntryJSON
				if err := dec.Decode(&e); err == nil {
					all = append(all, e)
				}
			}
			f.Close()
		}
	}
	return all, nil
}

func (tf *turnFile) writeEntry(e TurnLogEntry) {
	b, _ := json.Marshal(e)
	tf.f.Write(append(b, '\n'))
}
