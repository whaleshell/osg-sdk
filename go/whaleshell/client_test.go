package whaleshell_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/whaleshell/whaleshell-sdk/go/whaleshell"
)

func TestClientCRUDAndExec(t *testing.T) {
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
		case http.MethodPut:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]string{"name": "demo", "status": "running"})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	})
	mux.HandleFunc("/v1/relay/demo/exec", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"exit_code": 0, "output": "hi\n"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := whaleshell.New(srv.URL)
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
	if err := c.Connect(ctx, "demo"); err != whaleshell.ErrConnectUnsupported {
		t.Fatalf("connect err=%v", err)
	}
	if err := c.Delete(ctx, "demo"); err != nil {
		t.Fatal(err)
	}
}
