// Package whaleshell is the Go SDK for the whaleshell gateway control plane.
//
// Create/list/delete talk to whaleshell-gateway over HTTP. Exec uses the gateway
// relay (sandbox must run whaleshell-agent). Interactive TTY connect is not provided
// here — use the CLI: `whaleshell connect <name>`.
package whaleshell

import (
	"context"
	"errors"
	"fmt"
	"time"

	gc "github.com/whaleshell/whaleshell-sdk/internal/gatewayclient"
)

// ErrConnectUnsupported means interactive sessions stay on the CLI.
var ErrConnectUnsupported = errors.New("whaleshell-sdk: interactive connect is not supported; use: whaleshell connect <name>")

// Client wraps the gateway HTTP API.
type Client struct {
	*gc.Client
}

// New builds a client for a gateway base URL (e.g. http://127.0.0.1:7443).
func New(baseURL string) *Client {
	c := gc.New(baseURL)
	c.HTTP.Timeout = 70 * time.Second // relay exec long-poll
	return &Client{Client: c}
}

// NewWithToken returns a client with bearer auth.
func NewWithToken(base, token string) *Client {
	c := gc.NewWithToken(base, token)
	c.HTTP.Timeout = 70 * time.Second
	return &Client{Client: c}
}

// Stable type aliases (gateway HTTP payloads).
type (
	Sandbox        = gc.Sandbox
	ExecResult     = gc.ExecResult
	LogLine        = gc.LogLine
	Proposal       = gc.Proposal
	ProviderRecord = gc.ProviderRecord
	InferenceRoute = gc.InferenceRoute
	ServiceRecord  = gc.ServiceRecord
)

// Create registers (upserts) a sandbox in the gateway registry.
func (c *Client) Create(ctx context.Context, sb Sandbox) error {
	if sb.Name == "" {
		return fmt.Errorf("sandbox name required")
	}
	return c.UpsertSandbox(ctx, sb)
}

// List returns registered sandboxes.
func (c *Client) List(ctx context.Context) ([]Sandbox, error) {
	return c.ListSandboxes(ctx)
}

// Get returns one sandbox by name.
func (c *Client) Get(ctx context.Context, name string) (Sandbox, error) {
	return c.GetSandbox(ctx, name)
}

// Delete removes a sandbox from the registry.
func (c *Client) Delete(ctx context.Context, name string) error {
	return c.DeleteSandbox(ctx, name)
}

// Exec runs argv via gateway relay (sandbox agent must be polling).
func (c *Client) Exec(ctx context.Context, name string, argv ...string) (ExecResult, error) {
	if name == "" || len(argv) == 0 {
		return ExecResult{}, fmt.Errorf("usage: Exec(name, argv...)")
	}
	return c.Client.Exec(ctx, name, argv)
}

// Connect is intentionally unsupported in the SDK.
func (c *Client) Connect(ctx context.Context, name string) error {
	_ = ctx
	_ = name
	return ErrConnectUnsupported
}
