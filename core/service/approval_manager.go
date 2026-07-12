package service

import (
	"context"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"
)

type ApprovalManager struct {
	store port.ApprovalRepo
}

func NewApprovalManager(store port.ApprovalRepo) *ApprovalManager {
	return &ApprovalManager{store: store}
}

func (m *ApprovalManager) Create(ctx context.Context, a domain.Approval) error {
	return m.store.Create(ctx, a)
}
