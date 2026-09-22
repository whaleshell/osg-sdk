<h1 align="center">whaleshell-sdk</h1>

<p align="center">
  <strong>Go client for whaleshell-gateway</strong><br>
  Thin HTTP client (`go/whaleshell`) — no fat runtime dependency.
</p>
<p align="center">
  <a href="https://github.com/whaleshell/whaleshell-sdk/actions/workflows/ci.yml"><img src="https://github.com/whaleshell/whaleshell-sdk/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://pkg.go.dev/github.com/whaleshell/whaleshell-sdk"><img src="https://pkg.go.dev/badge/github.com/whaleshell/whaleshell-sdk.svg" alt="Go Reference"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License"></a>
  <a href="https://github.com/whaleshell/whaleshell-sdk"><img src="https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go" alt="Go Version"></a>
</p>
<p align="center">
  <sub>Part of the <a href="https://github.com/whaleshell">whaleshell / whaleshell</a> ecosystem</sub>
</p>

---

## Overview

**whaleshell-sdk** talks to [whaleshell-gateway](https://github.com/whaleshell/whaleshell-gateway) over HTTP: create/list/delete sandboxes, relay exec, logs, and policy proposals. Interactive TTY stays on the CLI (`whaleshell connect`).

### Key Features

| Category | Capabilities |
|----------|--------------|
| **Client** | `go/whaleshell` — public SDK (wraps internal gateway HTTP client) |
| **Relay** | Long-poll exec against `whaleshell-agent` |
| **Proposals** | List / get / approve / reject |

---

## Installation

```bash
go get github.com/whaleshell/whaleshell-sdk@latest
```

**Requirements:** Go 1.27+

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    "github.com/whaleshell/whaleshell-sdk/go/whaleshell"
)

func main() {
    c := whaleshell.New("http://127.0.0.1:7443")
    list, err := c.List(context.Background())
    if err != nil {
        panic(err)
    }
    fmt.Println(list)
}
```

---

## Package Structure

| Path | Purpose |
|------|---------|
| `go/whaleshell` | Public Go SDK |
| `internal/gatewayclient/` | HTTP API implementation (not importable outside the module) |


---

## Related

| Resource | Link |
|----------|------|
| Roadmap | [ROADMAP.md](./ROADMAP.md) |
| Organization | [https://github.com/whaleshell](https://github.com/whaleshell) |
| Organization overview | [github.com/whaleshell](https://github.com/whaleshell) |
| pkg.go.dev | [`github.com/whaleshell/whaleshell-sdk`](https://pkg.go.dev/github.com/whaleshell/whaleshell-sdk) |

## License

[MIT](./LICENSE) © whaleshell
