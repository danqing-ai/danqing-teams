package remote

import (
	"context"
	"errors"
	"testing"

	"danqing-teams/internal/domain/model"
	"danqing-teams/pkg/errs"
)

func TestClient_NotImplemented(t *testing.T) {
	c := New()
	_, err := c.Complete(context.Background(), model.CompletionRequest{Role: model.LLMRoleWorker})
	if !errors.Is(err, errs.ErrNotImplemented) {
		t.Fatalf("want ErrNotImplemented, got %v", err)
	}
}
