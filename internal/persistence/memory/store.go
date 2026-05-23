package memory

import (
	"context"
	"sync"
	"time"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
	"danqing-teams/pkg/id"
)

// Store is an in-memory implementation of all repositories.
type Store struct {
	mu sync.RWMutex

	teams      map[string]*teamRecord
	tasks      map[string]*contract.TeamTask
	dispatches map[string][]contract.Dispatch
	runs       map[string]*contract.WorkerRun
	plans      map[string]*contract.ExecutionPlan
	reports    map[string][]contract.Report
	timeline   map[string][]contract.TimelineEvent
	messages   map[string][]contract.TeamMessage
	approvals  map[string]*contract.ApprovalRequest
	todos      map[string][]contract.TodoItem
	artifacts  map[string][]contract.WorkspaceArtifact
	kbDocs     map[string][]contract.KnowledgeDoc // key teamId:workerId
	jobs       map[string]*contract.OrchestrationJob
	agents     map[string]contract.Agent
	teamAgents map[string]struct{} // key teamId:agentId
}

type teamRecord struct {
	team       contract.Team
	controller contract.TeamController
	workers    map[string]contract.WorkerAgent
	humans     []contract.HumanMember
}

func NewStore() *Store {
	return &Store{
		teams:      make(map[string]*teamRecord),
		tasks:      make(map[string]*contract.TeamTask),
		dispatches: make(map[string][]contract.Dispatch),
		runs:       make(map[string]*contract.WorkerRun),
		plans:      make(map[string]*contract.ExecutionPlan),
		reports:    make(map[string][]contract.Report),
		timeline:   make(map[string][]contract.TimelineEvent),
		messages:   make(map[string][]contract.TeamMessage),
		approvals:  make(map[string]*contract.ApprovalRequest),
		todos:      make(map[string][]contract.TodoItem),
		artifacts:  make(map[string][]contract.WorkspaceArtifact),
		kbDocs:     make(map[string][]contract.KnowledgeDoc),
		jobs:       make(map[string]*contract.OrchestrationJob),
		agents:     make(map[string]contract.Agent),
		teamAgents: make(map[string]struct{}),
	}
}

func kbKey(teamID, workerID string) string { return teamID + ":" + workerID }

func (s *Store) ListTeams(_ context.Context) ([]contract.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]contract.Team, 0, len(s.teams))
	for _, tr := range s.teams {
		out = append(out, tr.team)
	}
	return out, nil
}

func (s *Store) GetTeam(_ context.Context, teamID string) (*contract.TeamDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tr, ok := s.teams[teamID]
	if !ok {
		return nil, errs.NotFound("team not found")
	}
	workers := s.listWorkersFromAgents(teamID)
	if len(workers) == 0 {
		workers = make([]contract.WorkerAgent, 0, len(tr.workers))
		for _, w := range tr.workers {
			workers = append(workers, w)
		}
	}
	return &contract.TeamDetail{
		Team:       tr.team,
		Controller: tr.controller,
		Workers:    workers,
		Humans:     append([]contract.HumanMember(nil), tr.humans...),
	}, nil
}

func (s *Store) CreateTeam(_ context.Context, req contract.CreateTeamRequest) (*contract.TeamDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tid := id.New()
	now := time.Now().UTC()
	tr := &teamRecord{
		team: contract.Team{
			ID: tid, Name: req.Name, Description: req.Description,
			CreatedAt: now, UpdatedAt: now,
		},
		controller: contract.TeamController{
			Persona:      "负责理解用户意图，按 Worker 人设分派任务，汇总报告并规划 follow-up。",
			SystemPrompt: "你是 Team Controller，仅依据 Worker 人设匹配，不知道 Worker 的技能与 MCP Tool。",
		},
		workers: make(map[string]contract.WorkerAgent),
	}
	s.teams[tid] = tr
	return &contract.TeamDetail{Team: tr.team, Controller: tr.controller}, nil
}

func (s *Store) UpdateTeam(_ context.Context, teamID string, req contract.UpdateTeamRequest) (*contract.Team, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tr, ok := s.teams[teamID]
	if !ok {
		return nil, errs.NotFound("team not found")
	}
	if req.Name != nil {
		tr.team.Name = *req.Name
	}
	if req.Description != nil {
		tr.team.Description = *req.Description
	}
	tr.team.UpdatedAt = time.Now().UTC()
	return &tr.team, nil
}

func (s *Store) DeleteTeam(_ context.Context, teamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.teams[teamID]; !ok {
		return errs.NotFound("team not found")
	}
	delete(s.teams, teamID)
	return nil
}

func (s *Store) ListPersonaCatalog(ctx context.Context, teamID string) ([]contract.WorkerPersonaCatalog, error) {
	workers, err := s.ListWorkers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	out := make([]contract.WorkerPersonaCatalog, len(workers))
	for i, w := range workers {
		out[i] = contract.WorkerPersonaCatalog{ID: w.ID, Name: w.Name, Persona: w.Persona}
	}
	return out, nil
}

func (s *Store) GetController(_ context.Context, teamID string) (*contract.TeamController, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tr, ok := s.teams[teamID]
	if !ok {
		return nil, errs.NotFound("team not found")
	}
	c := tr.controller
	return &c, nil
}

func (s *Store) UpdateController(_ context.Context, teamID string, c contract.TeamController) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tr, ok := s.teams[teamID]
	if !ok {
		return errs.NotFound("team not found")
	}
	tr.controller = c
	return nil
}

func (s *Store) ListWorkers(_ context.Context, teamID string) ([]contract.WorkerAgent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if workers := s.listWorkersFromAgents(teamID); len(workers) > 0 {
		return workers, nil
	}
	tr, ok := s.teams[teamID]
	if !ok {
		return nil, errs.NotFound("team not found")
	}
	out := make([]contract.WorkerAgent, 0, len(tr.workers))
	for _, w := range tr.workers {
		out = append(out, w)
	}
	return out, nil
}

func (s *Store) GetWorker(_ context.Context, teamID, workerID string) (*contract.WorkerAgent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.teamAgents[teamAgentKey(teamID, workerID)]; ok {
		if a, exists := s.agents[workerID]; exists {
			w := agentToWorker(a)
			return &w, nil
		}
	}
	tr, ok := s.teams[teamID]
	if !ok {
		return nil, errs.NotFound("team not found")
	}
	w, ok := tr.workers[workerID]
	if !ok {
		return nil, errs.NotFound("worker not found")
	}
	return &w, nil
}

func (s *Store) UpsertWorker(_ context.Context, teamID string, worker *contract.WorkerAgent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tr, ok := s.teams[teamID]
	if !ok {
		return errs.NotFound("team not found")
	}
	if worker.ID == "" {
		worker.ID = id.New()
	}
	tr.workers[worker.ID] = *worker
	return nil
}

func (s *Store) DeleteWorker(_ context.Context, teamID, workerID string) error {
	if err := s.RemoveTeamAgent(context.Background(), teamID, workerID); err == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tr, ok := s.teams[teamID]
	if !ok {
		return errs.NotFound("team not found")
	}
	if _, ok := tr.workers[workerID]; !ok {
		return errs.NotFound("worker not found")
	}
	delete(tr.workers, workerID)
	return nil
}

func (s *Store) GetWorkerPrivateProfile(_ context.Context, teamID, workerID string) (*contract.WorkerPrivateProfile, error) {
	w, err := s.GetWorker(context.Background(), teamID, workerID)
	if err != nil {
		return nil, err
	}
	return &contract.WorkerPrivateProfile{
		WorkerID: w.ID, Skills: w.Skills, Tools: w.Tools, KnowledgeBase: w.KnowledgeBase,
	}, nil
}

func (s *Store) ListHumans(_ context.Context, teamID string) ([]contract.HumanMember, error) {
	detail, err := s.GetTeam(context.Background(), teamID)
	if err != nil {
		return nil, err
	}
	return detail.Humans, nil
}

func (s *Store) AddHuman(_ context.Context, teamID string, h contract.HumanMember) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tr, ok := s.teams[teamID]
	if !ok {
		return errs.NotFound("team not found")
	}
	if h.ID == "" {
		h.ID = id.New()
	}
	tr.humans = append(tr.humans, h)
	return nil
}

func (s *Store) ListTodos(_ context.Context, teamID, taskID string) ([]contract.TodoItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := s.todos[teamID]
	if taskID == "" {
		return append([]contract.TodoItem(nil), items...), nil
	}
	out := make([]contract.TodoItem, 0)
	for _, it := range items {
		if it.TaskID == taskID {
			out = append(out, it)
		}
	}
	return out, nil
}

func (s *Store) CreateTodo(_ context.Context, teamID string, item contract.TodoItem) (*contract.TodoItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.ID == "" {
		item.ID = id.New()
	}
	item.TeamID = teamID
	item.CreatedAt = time.Now().UTC()
	s.todos[teamID] = append(s.todos[teamID], item)
	return &item, nil
}

func (s *Store) UpdateTodo(_ context.Context, teamID, todoID string, done bool) (*contract.TodoItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, it := range s.todos[teamID] {
		if it.ID == todoID {
			it.Done = done
			s.todos[teamID][i] = it
			return &it, nil
		}
	}
	return nil, errs.NotFound("todo not found")
}

func (s *Store) ListTasks(_ context.Context, teamID string, status contract.TaskStatus) ([]contract.TeamTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]contract.TeamTask, 0)
	for _, t := range s.tasks {
		if t.TeamID == teamID && (status == "" || t.Status == status) {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (s *Store) GetTask(_ context.Context, teamID, taskID string) (*contract.TeamTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return nil, errs.NotFound("task not found")
	}
	if teamID != "" && t.TeamID != teamID {
		return nil, errs.NotFound("task not found")
	}
	return t, nil
}

func (s *Store) CreateTask(_ context.Context, task *contract.TeamTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
	return nil
}

func (s *Store) UpdateTaskStatus(_ context.Context, _, taskID string, status contract.TaskStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return errs.NotFound("task not found")
	}
	t.Status = status
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) UpdateTaskClosure(_ context.Context, _, taskID string, status contract.TaskStatus, reason contract.TaskCloseReason) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return errs.NotFound("task not found")
	}
	t.Status = status
	t.CloseReason = reason
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) SaveDispatch(_ context.Context, d *contract.Dispatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dispatches[d.TaskID] = append(s.dispatches[d.TaskID], *d)
	return nil
}

func (s *Store) ListDispatches(_ context.Context, taskID string) ([]contract.Dispatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]contract.Dispatch(nil), s.dispatches[taskID]...), nil
}

func (s *Store) SaveRun(_ context.Context, run *contract.WorkerRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.ID] = run
	return nil
}

func (s *Store) GetRun(_ context.Context, runID string) (*contract.WorkerRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[runID]
	if !ok {
		return nil, errs.NotFound("run not found")
	}
	return r, nil
}

func (s *Store) UpdateRun(_ context.Context, run *contract.WorkerRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[run.ID]; !ok {
		return errs.NotFound("run not found")
	}
	run.UpdatedAt = time.Now().UTC()
	s.runs[run.ID] = run
	return nil
}

func (s *Store) ListRuns(_ context.Context, taskID string) ([]contract.WorkerRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]contract.WorkerRun, 0)
	for _, r := range s.runs {
		if r.TaskID == taskID {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (s *Store) SavePlan(_ context.Context, plan *contract.ExecutionPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[plan.RunID] = plan
	return nil
}

func (s *Store) GetPlan(_ context.Context, runID string) (*contract.ExecutionPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.plans[runID]
	if !ok {
		return nil, errs.NotFound("plan not found")
	}
	return p, nil
}

func (s *Store) SaveReport(_ context.Context, r *contract.Report) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reports[r.TaskID] = append(s.reports[r.TaskID], *r)
	return nil
}

func (s *Store) ListReports(_ context.Context, taskID string) ([]contract.Report, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]contract.Report(nil), s.reports[taskID]...), nil
}

func (s *Store) AppendTimeline(_ context.Context, evt contract.TimelineEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timeline[evt.TaskID] = append(s.timeline[evt.TaskID], evt)
	return nil
}

func (s *Store) GetTimeline(_ context.Context, taskID string) ([]contract.TimelineEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]contract.TimelineEvent(nil), s.timeline[taskID]...), nil
}

func (s *Store) AppendMessage(_ context.Context, msg *contract.TeamMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[msg.TaskID] = append(s.messages[msg.TaskID], *msg)
	return nil
}

func (s *Store) ListMessages(_ context.Context, taskID string) ([]contract.TeamMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]contract.TeamMessage(nil), s.messages[taskID]...), nil
}

func (s *Store) Create(_ context.Context, req *contract.ApprovalRequest) error {
	return s.createApproval(req)
}

func (s *Store) Get(_ context.Context, teamID, approvalID string) (*contract.ApprovalRequest, error) {
	return s.GetApproval(context.Background(), teamID, approvalID)
}

func (s *Store) Update(_ context.Context, req *contract.ApprovalRequest) error {
	return s.UpdateApproval(context.Background(), req)
}

func (s *Store) List(_ context.Context, teamID string, status contract.ApprovalStatus) ([]contract.ApprovalRequest, error) {
	return s.ListApprovals(context.Background(), teamID, status)
}

func (s *Store) GetByRunID(ctx context.Context, runID string) (*contract.ApprovalRequest, error) {
	return s.GetApprovalByRunID(ctx, runID)
}

func (s *Store) createApproval(req *contract.ApprovalRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approvals[req.ID] = req
	return nil
}

func (s *Store) GetApproval(_ context.Context, _ string, approvalID string) (*contract.ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.approvals[approvalID]
	if !ok {
		return nil, errs.NotFound("approval not found")
	}
	return a, nil
}

func (s *Store) UpdateApproval(_ context.Context, req *contract.ApprovalRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approvals[req.ID] = req
	return nil
}

func (s *Store) ListApprovals(_ context.Context, teamID string, status contract.ApprovalStatus) ([]contract.ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]contract.ApprovalRequest, 0)
	for _, a := range s.approvals {
		if a.TeamID == teamID && (status == "" || a.Status == status) {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (s *Store) GetApprovalByRunID(_ context.Context, runID string) (*contract.ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.approvals {
		if a.RunID == runID {
			return a, nil
		}
	}
	return nil, errs.NotFound("approval not found")
}

func (s *Store) ListArtifacts(_ context.Context, teamID string) ([]contract.WorkspaceArtifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]contract.WorkspaceArtifact(nil), s.artifacts[teamID]...), nil
}

func (s *Store) CreateArtifact(_ context.Context, teamID string, a contract.WorkspaceArtifact) (*contract.WorkspaceArtifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.ID == "" {
		a.ID = id.New()
	}
	a.TeamID = teamID
	a.CreatedAt = time.Now().UTC()
	s.artifacts[teamID] = append(s.artifacts[teamID], a)
	return &a, nil
}

func (s *Store) ListKnowledgeDocs(_ context.Context, teamID, workerID string) ([]contract.KnowledgeDoc, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]contract.KnowledgeDoc(nil), s.kbDocs[kbKey(teamID, workerID)]...), nil
}

func (s *Store) SaveKnowledgeDocs(_ context.Context, teamID, workerID string, docs []contract.KnowledgeDoc) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kbDocs[kbKey(teamID, workerID)] = append([]contract.KnowledgeDoc(nil), docs...)
	return nil
}
