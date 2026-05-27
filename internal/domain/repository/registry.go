package repository

// Registry groups all persistence repositories for a single store backend.
type Registry struct {
	Teams     TeamRepository
	Tasks     TaskRepository
	Workspace WorkspaceRepository
	Approvals ApprovalRepository
	Agents    AgentRepository
	Jobs      JobRepository
	Recover   RecoverableTaskStore
}
