package service

import (
	"context"
	"strings"
	"time"

	"danqing-teams/internal/contract"
	"danqing-teams/internal/core/orchestration"
	"danqing-teams/internal/core/policy"
	"danqing-teams/internal/core/worker"
	"danqing-teams/pkg/errs"
	"danqing-teams/pkg/id"
)

type OrchestrationService struct {
	teams       contract.TeamRepository
	tasks       contract.TaskRepository
	approvals   contract.ApprovalRepository
	jobs        contract.JobRepository
	llm         contract.LLMProvider
	events      contract.EventPublisher
	autoApprove bool
}

func NewOrchestrationService(
	teams contract.TeamRepository,
	tasks contract.TaskRepository,
	approvals contract.ApprovalRepository,
	jobs contract.JobRepository,
	llm contract.LLMProvider,
	events contract.EventPublisher,
	autoApprove bool,
) *OrchestrationService {
	return &OrchestrationService{
		teams: teams, tasks: tasks, approvals: approvals, jobs: jobs,
		llm: llm, events: events, autoApprove: autoApprove,
	}
}

func (s *OrchestrationService) SubmitTask(ctx context.Context, teamID string, req contract.SubmitTaskRequest) (*contract.TeamTask, error) {
	resp, err := s.SendTeamMessage(ctx, teamID, contract.SendTeamMessageRequest{Content: req.Content})
	if err != nil {
		return nil, err
	}
	return &resp.Task, nil
}

// SendTeamMessage records a user message in the team thread and triggers Team Controller dispatch.
func (s *OrchestrationService) SendTeamMessage(ctx context.Context, teamID string, req contract.SendTeamMessageRequest) (*contract.SendTeamMessageResponse, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, errs.BadRequest("content is required")
	}

	var task *contract.TeamTask
	var err error
	if req.TaskID != "" {
		task, err = s.tasks.GetTask(ctx, teamID, req.TaskID)
		if err != nil {
			return nil, err
		}
		if task.TeamID != teamID {
			return nil, errs.NotFound("task not found")
		}
		_ = s.tasks.UpdateTaskStatus(ctx, teamID, task.ID, contract.TaskDispatching)
		task.Status = contract.TaskDispatching
		task.UpdatedAt = time.Now().UTC()
	} else {
		now := time.Now().UTC()
		task = &contract.TeamTask{
			ID: id.New(), TeamID: teamID, Content: content,
			Status: contract.TaskDispatching, CreatedAt: now, UpdatedAt: now,
		}
		if err := s.tasks.CreateTask(ctx, task); err != nil {
			return nil, err
		}
	}

	userMsg := s.recordMessage(ctx, teamID, task.ID, contract.MessageRoleUser, content)

	dispatches, _ := s.tasks.ListDispatches(ctx, task.ID)
	round := len(dispatches)
	if err := s.enqueueRunTask(ctx, teamID, task.ID, content, round, ""); err != nil {
		return nil, err
	}

	return &contract.SendTeamMessageResponse{Message: userMsg, Task: *task}, nil
}

func (s *OrchestrationService) ListMessages(ctx context.Context, _, taskID string) ([]contract.TeamMessage, error) {
	return s.tasks.ListMessages(ctx, taskID)
}

func (s *OrchestrationService) recordMessage(ctx context.Context, teamID, taskID string, role contract.MessageRole, content string) contract.TeamMessage {
	now := time.Now().UTC()
	msg := contract.TeamMessage{
		ID: id.New(), TeamID: teamID, TaskID: taskID,
		Role: role, Content: content, CreatedAt: now,
	}
	_ = s.tasks.AppendMessage(ctx, &msg)
	_ = s.tasks.AppendTimeline(ctx, contract.TimelineEvent{
		ID: id.New(), TaskID: taskID, Type: "message",
		Payload: msg, CreatedAt: now,
	})
	s.events.Publish(ctx, teamID, taskID, contract.StreamEvent{
		Type: contract.EventMessagePosted, Payload: msg,
	})
	return msg
}

func (s *OrchestrationService) runTask(ctx context.Context, teamID string, task *contract.TeamTask, intent string, round int, contextSummary string) {
	// Idempotent: if this round was already dispatched, resume from saved run.
	dispatches, _ := s.tasks.ListDispatches(ctx, task.ID)
	for _, d := range dispatches {
		if d.Round != round {
			continue
		}
		runs, _ := s.tasks.ListRuns(ctx, task.ID)
		for _, r := range runs {
			if r.DispatchID != d.ID {
				continue
			}
			s.resumeRunTask(ctx, teamID, task, d, &r, intent, round, contextSummary)
			return
		}
	}

	personas, err := s.teams.ListPersonaCatalog(ctx, teamID)
	if err != nil {
		s.failTask(ctx, teamID, task.ID, err)
		return
	}
	ctrl, err := s.teams.GetController(ctx, teamID)
	if err != nil {
		s.failTask(ctx, teamID, task.ID, err)
		return
	}
	persona, ok := orchestration.DispatchWorker(ctx, s.llm, ctrl, intent, personas)
	if !ok {
		s.completeTask(ctx, teamID, task.ID, contract.CloseReasonNoIntent)
		return
	}
	ctrlMsg := "分派 @" + persona.Name
	if round > 0 {
		ctrlMsg = "跟进分派 @" + persona.Name
	} else if persona.Persona != "" {
		ctrlMsg += "：" + truncatePersona(persona.Persona, 80)
	}
	s.recordMessage(ctx, teamID, task.ID, contract.MessageRoleController, ctrlMsg)
	if contextSummary == "" {
		contextSummary = orchestration.BuildContextSummary(intent, round)
	}

	dispatch := contract.Dispatch{
		ID: id.New(), TaskID: task.ID, WorkerID: persona.ID, WorkerName: persona.Name,
		Intent: intent, ContextSummary: contextSummary, Round: round, CreatedAt: time.Now().UTC(),
	}
	_ = s.tasks.SaveDispatch(ctx, &dispatch)
	_ = s.tasks.AppendTimeline(ctx, contract.TimelineEvent{
		ID: id.New(), TaskID: task.ID, Type: "dispatch", Payload: dispatch, CreatedAt: time.Now().UTC(),
	})
	s.events.Publish(ctx, teamID, task.ID, contract.StreamEvent{Type: contract.EventDispatchCreated, Payload: dispatch})

	run := &contract.WorkerRun{
		ID: id.New(), TaskID: task.ID, DispatchID: dispatch.ID, WorkerID: persona.ID,
		Status: contract.RunPlanning, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	_ = s.tasks.SaveRun(ctx, run)
	_ = s.tasks.UpdateTaskStatus(ctx, teamID, task.ID, contract.TaskRunning)
	s.events.Publish(ctx, teamID, task.ID, contract.StreamEvent{Type: contract.EventRunPlanning, Payload: run})

	profile, err := s.teams.GetWorkerPrivateProfile(ctx, teamID, persona.ID)
	if err != nil {
		s.failTask(ctx, teamID, task.ID, err)
		return
	}

	plan := worker.PlanExecution(intent, *profile)
	maxRisk, highItems := policy.EvaluatePlan(*profile, plan)
	plan.EvaluatedRisk = maxRisk
	plan.HighRiskItems = highItems
	plan.RunID = run.ID
	run.Plan = &plan
	_ = s.tasks.SavePlan(ctx, &plan)
	_ = s.tasks.UpdateRun(ctx, run)
	s.events.Publish(ctx, teamID, task.ID, contract.StreamEvent{Type: contract.EventPlanReady, Payload: plan})

	if policy.RequiresApproval(maxRisk, highItems) {
		if err := s.requestApproval(ctx, teamID, task.ID, run, plan); err != nil {
			s.failTask(ctx, teamID, task.ID, err)
			return
		}
		if !s.autoApprove {
			run.Status = contract.RunAwaitingApproval
			_ = s.tasks.UpdateRun(ctx, run)
			_ = s.tasks.UpdateTaskStatus(ctx, teamID, task.ID, contract.TaskAwaitingApproval)
			return // resume on Approve
		}
	}

	s.executeRun(ctx, teamID, task, run, dispatch, persona.Name, profile, plan)
}

func (s *OrchestrationService) resumeRunTask(
	ctx context.Context,
	teamID string,
	task *contract.TeamTask,
	dispatch contract.Dispatch,
	run *contract.WorkerRun,
	intent string,
	round int,
	contextSummary string,
) {
	if run.Status == contract.RunAwaitingApproval {
		return
	}
	if run.Status == contract.RunCompleted || run.Status == contract.RunFailed || run.Status == contract.RunRejected {
		return
	}
	profile, err := s.teams.GetWorkerPrivateProfile(ctx, teamID, run.WorkerID)
	if err != nil {
		s.failTask(ctx, teamID, task.ID, err)
		return
	}
	w, _ := s.teams.GetWorker(ctx, teamID, run.WorkerID)
	name := dispatch.WorkerName
	if w != nil && name == "" {
		name = w.Name
	}
	plan, err := s.tasks.GetPlan(ctx, run.ID)
	if err != nil {
		s.runTask(ctx, teamID, task, intent, round, contextSummary)
		return
	}
	if run.Status == contract.RunRunning {
		s.executeRun(ctx, teamID, task, run, dispatch, name, profile, *plan)
		return
	}
	if policy.RequiresApproval(plan.EvaluatedRisk, plan.HighRiskItems) && !s.autoApprove {
		run.Status = contract.RunAwaitingApproval
		_ = s.tasks.UpdateRun(ctx, run)
		_ = s.tasks.UpdateTaskStatus(ctx, teamID, task.ID, contract.TaskAwaitingApproval)
		return
	}
	s.executeRun(ctx, teamID, task, run, dispatch, name, profile, *plan)
}

func (s *OrchestrationService) requestApproval(ctx context.Context, teamID, taskID string, run *contract.WorkerRun, plan contract.ExecutionPlan) error {
	summary := "拟执行高危操作：" + joinRiskNames(plan.HighRiskItems)
	approval := &contract.ApprovalRequest{
		ID: id.New(), TeamID: teamID, TaskID: taskID, RunID: run.ID,
		Summary: summary, HighRiskItems: plan.HighRiskItems,
		Status: contract.ApprovalPending, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := s.approvals.Create(ctx, approval); err != nil {
		return err
	}
	_ = s.tasks.AppendTimeline(ctx, contract.TimelineEvent{
		ID: id.New(), TaskID: taskID, Type: "approval", Payload: approval, CreatedAt: time.Now().UTC(),
	})
	s.events.Publish(ctx, teamID, taskID, contract.StreamEvent{Type: contract.EventApprovalRequired, Payload: approval})
	return nil
}

func (s *OrchestrationService) ResumeRunAfterApproval(ctx context.Context, teamID, runID string) error {
	run, err := s.tasks.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	if run.Status != contract.RunAwaitingApproval {
		return nil
	}
	task, err := s.tasks.GetTask(ctx, teamID, run.TaskID)
	if err != nil {
		return err
	}
	dispatches, _ := s.tasks.ListDispatches(ctx, run.TaskID)
	var dispatch contract.Dispatch
	for _, d := range dispatches {
		if d.ID == run.DispatchID {
			dispatch = d
			break
		}
	}
	profile, err := s.teams.GetWorkerPrivateProfile(ctx, teamID, run.WorkerID)
	if err != nil {
		return err
	}
	w, _ := s.teams.GetWorker(ctx, teamID, run.WorkerID)
	name := ""
	if w != nil {
		name = w.Name
	}
	plan, err := s.tasks.GetPlan(ctx, run.ID)
	if err != nil {
		return err
	}
	s.executeRun(ctx, teamID, task, run, dispatch, name, profile, *plan)
	return nil
}

func (s *OrchestrationService) executeRun(
	ctx context.Context,
	teamID string,
	task *contract.TeamTask,
	run *contract.WorkerRun,
	dispatch contract.Dispatch,
	workerName string,
	profile *contract.WorkerPrivateProfile,
	plan contract.ExecutionPlan,
) {
	reports, _ := s.tasks.ListReports(ctx, task.ID)
	for _, r := range reports {
		if r.RunID == run.ID {
			s.finishAfterReport(ctx, teamID, task, dispatch, r)
			return
		}
	}

	run.Status = contract.RunRunning
	_ = s.tasks.UpdateRun(ctx, run)
	s.events.Publish(ctx, teamID, task.ID, contract.StreamEvent{Type: contract.EventRunStarted, Payload: run})

	resp, err := s.llm.Complete(ctx, contract.CompletionRequest{
		Role: contract.LLMRoleWorker,
		Context: map[string]string{
			"worker_name": workerName,
			"intent":      dispatch.Intent,
			"plan_skills": strings.Join(plan.SkillIDs, ", "),
			"plan_tools":  strings.Join(plan.ToolIDs, ", "),
			"needs_follow_up": "false",
		},
	})
	if err != nil {
		s.failRun(ctx, teamID, task.ID, run, err)
		return
	}

	intent, actions := orchestration.AnalyzeReportIntent(resp.Content)
	report := contract.Report{
		ID: id.New(), RunID: run.ID, TaskID: task.ID,
		WorkerID: run.WorkerID, WorkerName: workerName,
		ContentMarkdown: resp.Content, Intent: intent, SuggestedActions: actions,
		CreatedAt: time.Now().UTC(),
	}
	_ = s.tasks.SaveReport(ctx, &report)
	_ = s.tasks.AppendTimeline(ctx, contract.TimelineEvent{
		ID: id.New(), TaskID: task.ID, Type: "report", Payload: report, CreatedAt: time.Now().UTC(),
	})
	s.events.Publish(ctx, teamID, task.ID, contract.StreamEvent{Type: contract.EventReportReceived, Payload: report})

	run.Status = contract.RunCompleted
	_ = s.tasks.UpdateRun(ctx, run)
	s.finishAfterReport(ctx, teamID, task, dispatch, report)
}

func (s *OrchestrationService) finishAfterReport(
	ctx context.Context,
	teamID string,
	task *contract.TeamTask,
	dispatch contract.Dispatch,
	report contract.Report,
) {
	const maxFollowUpRounds = 2
	if report.Intent == contract.ReportNeedsFollowUp && len(report.SuggestedActions) > 0 && dispatch.Round < maxFollowUpRounds {
		hint := report.SuggestedActions[0].TargetPersonaHint
		_ = s.enqueueRunTask(ctx, teamID, task.ID,
			"对 prod 执行扩容："+hint, dispatch.Round+1,
			orchestration.BuildContextSummary(report.ContentMarkdown, dispatch.Round+1))
		return
	}

	s.completeTask(ctx, teamID, task.ID, contract.CloseReasonDone)
}

func (s *OrchestrationService) completeTask(ctx context.Context, teamID, taskID string, reason contract.TaskCloseReason) {
	_ = s.tasks.UpdateTaskClosure(ctx, teamID, taskID, contract.TaskCompleted, reason)
	s.events.Publish(ctx, teamID, taskID, contract.StreamEvent{Type: contract.EventTaskCompleted, Payload: reason})
}

func (s *OrchestrationService) failTask(ctx context.Context, teamID, taskID string, err error) {
	reason := s.resolveFailReason(ctx, taskID, err)
	_ = s.tasks.UpdateTaskClosure(ctx, teamID, taskID, contract.TaskFailed, reason)
	var payload any = reason
	if err != nil {
		payload = err.Error()
	}
	s.events.Publish(ctx, teamID, taskID, contract.StreamEvent{Type: contract.EventTaskFailed, Payload: payload})
}

func (s *OrchestrationService) resolveFailReason(ctx context.Context, taskID string, err error) contract.TaskCloseReason {
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "cancel") {
			return contract.CloseReasonCancelled
		}
	}
	dispatches, _ := s.tasks.ListDispatches(ctx, taskID)
	if len(dispatches) >= 2 {
		return contract.CloseReasonExhausted
	}
	return contract.CloseReasonError
}

func (s *OrchestrationService) failRun(ctx context.Context, teamID, taskID string, run *contract.WorkerRun, err error) {
	run.Status = contract.RunFailed
	_ = s.tasks.UpdateRun(ctx, run)
	s.failTask(ctx, teamID, taskID, err)
}

func (s *OrchestrationService) GetTimeline(ctx context.Context, _, taskID string) ([]contract.TimelineEvent, error) {
	return s.tasks.GetTimeline(ctx, taskID)
}

func (s *OrchestrationService) GetPlan(ctx context.Context, _, _, runID string) (*contract.ExecutionPlan, error) {
	return s.tasks.GetPlan(ctx, runID)
}

func (s *OrchestrationService) CancelTask(ctx context.Context, teamID, taskID string) error {
	_ = s.tasks.UpdateTaskClosure(ctx, teamID, taskID, contract.TaskFailed, contract.CloseReasonCancelled)
	runs, _ := s.tasks.ListRuns(ctx, taskID)
	for _, r := range runs {
		if r.Status == contract.RunQueued || r.Status == contract.RunPlanning ||
			r.Status == contract.RunAwaitingApproval || r.Status == contract.RunRunning {
			r.Status = contract.RunRejected
			_ = s.tasks.UpdateRun(ctx, &r)
		}
	}
	s.events.Publish(ctx, teamID, taskID, contract.StreamEvent{Type: contract.EventTaskFailed, Payload: "cancelled"})
	return nil
}

func truncatePersona(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func joinRiskNames(items []contract.RiskItem) string {
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.DisplayName
	}
	return strings.Join(names, ", ")
}
