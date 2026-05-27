package model

import "context"

type EventType string

const (
	EventMessagePosted     EventType = "message.posted"
	EventDispatchCreated   EventType = "dispatch.created"
	EventRunPlanning       EventType = "run.planning"
	EventPlanReady         EventType = "run.plan_ready"
	EventApprovalRequired  EventType = "approval.required"
	EventApprovalApproved  EventType = "approval.approved"
	EventApprovalRejected  EventType = "approval.rejected"
	EventRunStarted        EventType = "run.started"
	EventReportReceived    EventType = "report.received"
	EventTaskCompleted     EventType = "task.completed"
	EventTaskFailed        EventType = "task.failed"
)

type StreamEvent struct {
	Type      EventType
	TaskID    string
	TeamID    string
	Payload   any
}

type EventPublisher interface {
	Publish(ctx context.Context, teamID, taskID string, evt StreamEvent)
	Subscribe(taskID string) (<-chan StreamEvent, func())
}
