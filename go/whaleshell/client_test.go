package whaleshell_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/whaleshell/whaleshell-sdk/go/whaleshell"
)

const testToken = "sdk-test-token"

// requireBearer mimics the gateway: every route needs the bearer.
func requireBearer(next http.Handler, unauth *atomic.Int32) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testToken {
			unauth.Add(1)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func newServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/v1/sandboxes", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sandboxes": []map[string]string{{"name": "demo", "status": "running"}},
		})
	})
	mux.HandleFunc("/v1/sandboxes/demo", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut, http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]string{"name": "demo", "status": "running"})
		}
	})
	mux.HandleFunc("/v1/sandboxes/demo/exec", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"exit_code": 0, "output": "hi\n"})
	})
	mux.HandleFunc("/v1/sandboxes/cold/exec", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sandbox is not ready", http.StatusPreconditionFailed)
	})
	mux.HandleFunc("/v1/sandboxes/cold/ssh-session", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sandbox is not ready", http.StatusPreconditionFailed)
	})
	mux.HandleFunc("/v1/sandboxes/demo/ssh-session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": "ssh-1", "sandbox_id": "demo", "token": "sess-tok",
			"gateway_scheme": "http", "gateway_host": "127.0.0.1", "gateway_port": 7443, "expires_at_ms": 42,
		})
	})
	mux.HandleFunc("/v1/sandboxes/demo/supervisor-token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"sandbox": "demo", "sandbox_token": "sb-tok"})
	})
	mux.HandleFunc("/v1/ssh-sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("sandbox") != "demo" {
			http.Error(w, "want sandbox=demo", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"sessions": []map[string]any{{"id": "ssh-1", "sandbox": "demo"}}})
	})
	mux.HandleFunc("/v1/ssh-sessions/ssh-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	unauth := &atomic.Int32{}
	srv := httptest.NewServer(requireBearer(mux, unauth))
	t.Cleanup(srv.Close)
	return srv, unauth
}

func TestClientCRUDAndExec(t *testing.T) {
	srv, unauth := newServer(t)
	c := whaleshell.NewWithToken(srv.URL, testToken)
	ctx := context.Background()
	hz, err := c.Healthz(ctx)
	if err != nil || hz["ok"] != true {
		t.Fatalf("healthz=%v err=%v", hz, err)
	}
	if err := c.Create(ctx, whaleshell.Sandbox{Name: "demo", Status: "running"}); err != nil {
		t.Fatal(err)
	}
	list, err := c.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
	got, err := c.Get(ctx, "demo")
	if err != nil || got.Name != "demo" {
		t.Fatalf("get=%v err=%v", got, err)
	}
	res, err := c.Exec(ctx, "demo", "echo", "hi")
	if err != nil || res.ExitCode != 0 || res.Output != "hi\n" {
		t.Fatalf("exec=%v err=%v", res, err)
	}
	if _, err := c.Exec(ctx, "cold", "true"); !errors.Is(err, whaleshell.ErrSandboxNotReady) {
		t.Fatalf("exec on cold sandbox err=%v, want ErrSandboxNotReady", err)
	}
	if err := c.Connect(ctx, "demo"); err != whaleshell.ErrConnectUnsupported {
		t.Fatalf("connect err=%v", err)
	}
	if err := c.Delete(ctx, "demo"); err != nil {
		t.Fatal(err)
	}
	if n := unauth.Load(); n != 0 {
		t.Fatalf("%d requests were sent without the bearer", n)
	}
}

func TestSSHSessionAPI(t *testing.T) {
	srv, unauth := newServer(t)
	c := whaleshell.NewWithToken(srv.URL, testToken)
	ctx := context.Background()
	s, err := c.CreateSSHSession(ctx, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if s.SessionID != "ssh-1" || s.Token != "sess-tok" || s.GatewayPort != 7443 || s.ExpiresAtMS != 42 {
		t.Fatalf("session=%+v", s)
	}
	if _, err := c.CreateSSHSession(ctx, "cold"); !errors.Is(err, whaleshell.ErrSandboxNotReady) {
		t.Fatalf("cold session err=%v", err)
	}
	list, err := c.ListSSHSessions(ctx, "demo")
	if err != nil || len(list) != 1 || list[0].ID != "ssh-1" {
		t.Fatalf("list=%v err=%v", list, err)
	}
	if err := c.RevokeSSHSession(ctx, "ssh-1"); err != nil {
		t.Fatal(err)
	}
	tok, err := c.IssueSandboxToken(ctx, "demo")
	if err != nil || tok != "sb-tok" {
		t.Fatalf("sandbox token=%q err=%v", tok, err)
	}
	if n := unauth.Load(); n != 0 {
		t.Fatalf("%d requests were sent without the bearer", n)
	}
}

func TestTokenFromEnvAndMissingToken(t *testing.T) {
	srv, _ := newServer(t)
	t.Setenv(whaleshell.EnvToken, testToken)
	if _, err := whaleshell.New(srv.URL).List(context.Background()); err != nil {
		t.Fatalf("env token: %v", err)
	}
	t.Setenv(whaleshell.EnvToken, "")
	if _, err := whaleshell.New(srv.URL).List(context.Background()); err == nil {
		t.Fatal("expected 401 without token")
	}
}
