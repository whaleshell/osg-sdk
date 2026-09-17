<h1 align="center">osg-sdk</h1>

<p align="center">
  <strong>Go client for osg-gateway</strong><br>
  Thin HTTP client + gatewayclient — no fat runtime dependency.
</p>
<p align="center">
  <a href="https://github.com/zorneth/osg-sdk/actions/workflows/ci.yml"><img src="https://github.com/zorneth/osg-sdk/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://pkg.go.dev/github.com/zorneth/osg-sdk"><img src="https://pkg.go.dev/badge/github.com/zorneth/osg-sdk.svg" alt="Go Reference"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License"></a>
  <a href="https://github.com/zorneth/osg-sdk"><img src="https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go" alt="Go Version"></a>
</p>
<p align="center">
  <sub>Part of the <a href="https://github.com/zorneth">zorneth / osg</a> ecosystem</sub>
</p>

---

## Overview

**osg-sdk** talks to [osg-gateway](https://github.com/zorneth/osg-gateway) over HTTP: create/list/delete sandboxes, relay exec, logs, and policy proposals. Interactive TTY stays on the CLI (`osg connect`).

### Key Features

| Category | Capabilities |
|----------|--------------|
| **Client** | `go/osg` ergonomic wrapper |
| **Low-level** | `gatewayclient` for raw `/v1/*` |
| **Relay** | Long-poll exec against `osg-agent` |
| **Proposals** | List / get / approve / reject |

---

## Installation

```bash
go get github.com/zorneth/osg-sdk@latest
```

**Requirements:** Go 1.27+

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    "github.com/zorneth/osg-sdk/go/osg"
)

func main() {
    c := osg.New("http://127.0.0.1:7443")
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
| `go/osg` | High-level client |
| `gatewayclient/` | HTTP API types + calls |


---

## Related

| Resource | Link |
|----------|------|
| Roadmap | [ROADMAP.md](./ROADMAP.md) |
| Organization | [https://github.com/zorneth](https://github.com/zorneth) |
| Organization overview | [github.com/zorneth](https://github.com/zorneth) |
| pkg.go.dev | [`github.com/zorneth/osg-sdk`](https://pkg.go.dev/github.com/zorneth/osg-sdk) |

## License

[MIT](./LICENSE) © zorneth
