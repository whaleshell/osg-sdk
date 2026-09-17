// Package osg is the Go SDK for the osg gateway control plane.
//
// Create/list/delete talk to osg-gateway over HTTP. Exec uses the gateway
// relay (sandbox must run osg-agent). Interactive TTY connect is not provided
// here — use the CLI: `osg connect <name>`.
package osg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zorneth/osg-sdk/gatewayclient"
)

// ErrConnectUnsupported means interactive sessions stay on the CLI.
var ErrConnectUnsupported = errors.New("osg-sdk: interactive connect is not supported; use: osg connect <name>")

// Client wraps the gateway HTTP API.
type Client struct {
	gw *gatewayclient.Client
}

// New builds a client for a gateway base URL (e.g. http://127.0.0.1:7443).
func New(baseURL string) *Client {
	c := gatewayclient.New(baseURL)
	c.HTTP.Timeout = 70 * time.Second // relay exec long-poll
	return &Client{gw: c}
}

// Sandbox is a registered sandbox record.
type Sandbox = gatewayclient.Sandbox

// ExecResult is relay exec output.
type ExecResult = gatewayclient.ExecResult

// Healthz checks gateway liveness.
func (c *Client) Healthz(ctx context.Context) (map[string]any, error) {
	return c.gw.Healthz(ctx)
}

// Info returns gateway metadata.
func (c *Client) Info(ctx context.Context) (map[string]any, error) {
	return c.gw.Info(ctx)
}

// Create registers (upserts) a sandbox in the gateway registry.
// Local Docker create remains a CLI/orchestrator concern in P11.
func (c *Client) Create(ctx context.Context, sb Sandbox) error {
	if sb.Name == "" {
		return fmt.Errorf("sandbox name required")
	}
	return c.gw.UpsertSandbox(ctx, sb)
}

// List returns registered sandboxes.
func (c *Client) List(ctx context.Context) ([]Sandbox, error) {
	return c.gw.ListSandboxes(ctx)
}

// Get returns one sandbox by name.
func (c *Client) Get(ctx context.Context, name string) (Sandbox, error) {
	return c.gw.GetSandbox(ctx, name)
}

// Delete removes a sandbox from the registry.
func (c *Client) Delete(ctx context.Context, name string) error {
	return c.gw.DeleteSandbox(ctx, name)
}

// Exec runs argv via gateway relay (sandbox agent must be polling).
func (c *Client) Exec(ctx context.Context, name string, argv ...string) (ExecResult, error) {
	if name == "" || len(argv) == 0 {
		return ExecResult{}, fmt.Errorf("usage: Exec(name, argv...)")
	}
	return c.gw.Exec(ctx, name, argv)
}

// Connect is intentionally unsupported in the SDK.
func (c *Client) Connect(ctx context.Context, name string) error {
	_ = ctx
	_ = name
	return ErrConnectUnsupported
}
