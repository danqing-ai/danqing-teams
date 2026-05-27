package remote

import (
	"context"

	"danqing-teams/internal/domain/model"
	"danqing-teams/pkg/errs"
)

// Client is a stub for OpenAI-compatible remote APIs.
type Client struct{}

func New() *Client { return &Client{} }

func (c *Client) Complete(_ context.Context, _ model.CompletionRequest) (model.CompletionResponse, error) {
	return model.CompletionResponse{}, errs.ErrNotImplemented
}
