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

// Create opens a turn log for writing.
//
// If the JSONL already exists (same-process resume after EndTurn, or process
// restart with the file still on disk), it reopens for append and does NOT
// write another "start" entry. Otherwise it creates a fresh file with start.
func (s *TurnLogStore) Create(turnID, sessionID, projectID, agentID, goal string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if projectID == "" {
		projectID = "_default"
	}
	s.sessionProjects[sessionID] = projectID

	if tf, ok := s.loadTurnFileLocked(turnID); ok {
		if goal != "" && tf.log.Goal == "" {
			tf.log.Goal = goal
		}
		if agentID != "" && tf.log.AgentID == "" {
			tf.log.AgentID = agentID
		}
		tf.log.SessionID = sessionID
		tf.log.Status = domain.TurnRunning
		return s.reopenAppendLocked(tf)
	}

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

	tf.seq = 1
	tf.writeEntry(TurnLogEntry{Seq: tf.seq, Type: "start", Data: map[string]any{"agent_id": agentID, "goal": goal}})
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
	if !ok || tf.f == nil {
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
	if tf.f != nil {
		tf.seq++
		mapStatus := map[string]any{"status": string(status)}
		tf.writeEntry(TurnLogEntry{Seq: tf.seq, Type: "end", Data: mapStatus})
		_ = tf.f.Close()
		tf.f = nil
	}
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
			if _, ok := s.loadTurnFileLocked(id); ok {
				if tf, ok := s.logs[id]; ok {
					out = append(out, *tf.log)
				}
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

	tf, ok := s.loadTurnFileLocked(turnID)
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
	// Walk from the end so we can drop an unpaired trailing tool_call
	// (cancel / tool error mid-flight) without discarding earlier pairs.
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
			// Drop this unpaired call; keep everything before it.
			return goal, all[:i]
		}
		// All other types (start, end, unknown) silently skipped.
	}

	return goal, all[:cut]
}

func (s *TurnLogStore) LoadRawLog(turnID string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tf, ok := s.loadTurnFileLocked(turnID)
	if !ok {
		return nil, fmt.Errorf("turn log not found: %s", turnID)
	}

	return os.ReadFile(tf.filePath)
}

// TurnLogNode represents a turn and its delegated children for zip packaging.
type TurnLogNode struct {
	TurnID   string         `json:"turnId"`
	AgentID  string         `json:"agentId,omitempty"`
	Goal     string         `json:"goal,omitempty"`
	File     string         `json:"file"`
	Children []*TurnLogNode `json:"children,omitempty"`
}

func (s *TurnLogStore) LoadTurnLogZip(turnID string, events []domain.StreamEvent) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.loadTurnFileLocked(turnID); !ok {
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
		tf, exists := s.loadTurnFileLocked(id)
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

	tf, ok := s.loadTurnFileLocked(turnID)
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

// loadTurnFileLocked returns an in-memory turnFile, rehydrating from disk when
// needed (e.g. after process restart). Caller must hold s.mu.
func (s *TurnLogStore) loadTurnFileLocked(turnID string) (*turnFile, bool) {
	if tf, ok := s.logs[turnID]; ok {
		return tf, true
	}
	filePath, sessionID, projectID := s.locateTurnFile(turnID)
	if filePath == "" {
		return nil, false
	}
	goal, agentID, status, lastSeq := readTurnMeta(filePath)
	tf := &turnFile{
		log: &domain.TurnLog{
			ID:        turnID,
			SessionID: sessionID,
			Status:    status,
			AgentID:   agentID,
			Goal:      goal,
		},
		f:        nil,
		seq:      lastSeq,
		filePath: filePath,
	}
	s.logs[turnID] = tf
	if sessionID != "" {
		s.sessionProjects[sessionID] = projectID
	}
	return tf, true
}

func (s *TurnLogStore) reopenAppendLocked(tf *turnFile) error {
	if tf.f != nil {
		return nil
	}
	f, err := os.OpenFile(tf.filePath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	tf.f = f
	return nil
}

// locateTurnFile scans project data dirs for sessions/*/{turnID}.jsonl.
func (s *TurnLogStore) locateTurnFile(turnID string) (filePath, sessionID, projectID string) {
	root := s.projector("")
	projectEntries, err := os.ReadDir(root)
	if err != nil {
		return "", "", ""
	}
	want := turnID + ".jsonl"
	for _, pe := range projectEntries {
		if !pe.IsDir() {
			continue
		}
		pid := pe.Name()
		sessionsRoot := filepath.Join(s.projector(pid), "sessions")
		sessionEntries, err := os.ReadDir(sessionsRoot)
		if err != nil {
			continue
		}
		for _, se := range sessionEntries {
			if !se.IsDir() {
				continue
			}
			candidate := filepath.Join(sessionsRoot, se.Name(), want)
			if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
				return candidate, se.Name(), pid
			}
		}
	}
	return "", "", ""
}

func readTurnMeta(filePath string) (goal, agentID string, status domain.TurnStatus, lastSeq int) {
	status = domain.TurnRunning
	f, err := os.Open(filePath)
	if err != nil {
		return "", "", status, 0
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	for dec.More() {
		var e EntryJSON
		if err := dec.Decode(&e); err != nil {
			break
		}
		if e.Seq > lastSeq {
			lastSeq = e.Seq
		}
		switch e.Type {
		case "start":
			if e.Data != nil {
				if g, ok := e.Data["goal"].(string); ok && g != "" {
					goal = g
				}
				if a, ok := e.Data["agent_id"].(string); ok && a != "" {
					agentID = a
				}
			}
		case "end":
			if e.Data != nil {
				if st, ok := e.Data["status"].(string); ok && st != "" {
					status = domain.TurnStatus(st)
				}
			}
		}
	}
	return goal, agentID, status, lastSeq
}
