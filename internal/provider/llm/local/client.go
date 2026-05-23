package local

import (
	"context"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
)

// Client is a stub for ONNX Runtime + local models.
type Client struct{}

func New() *Client { return &Client{} }

func (c *Client) Complete(_ context.Context, _ contract.CompletionRequest) (contract.CompletionResponse, error) {
	return contract.CompletionResponse{}, errs.ErrNotImplemented
}
