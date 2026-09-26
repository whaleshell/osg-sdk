package gatewayclient

import (
	"context"
	"net/http"
	"net/url"
)

// ExecResult is the relay exec response.
type ExecResult struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}

// GetSandbox GET /v1/sandboxes/{name}.
func (c *Client) GetSandbox(ctx context.Context, name string) (Sandbox, error) {
	var sb Sandbox
	if err := c.get(ctx, "/v1/sandboxes/"+name, &sb); err != nil {
		return Sandbox{}, err
	}
	return sb, nil
}

// Exec runs argv in the sandbox over the supervisor SSH relay (OpenShell
// ExecSandbox). Returns ErrSandboxNotReady when the relay is not connected.
func (c *Client) Exec(ctx context.Context, name string, argv []string) (ExecResult, error) {
	var out ExecResult
	err := c.do(ctx, http.MethodPost, "/v1/sandboxes/"+url.PathEscape(name)+"/exec", map[string]any{"argv": argv}, http.StatusOK, &out)
	return out, err
}
