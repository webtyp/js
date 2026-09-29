# webtyp/js
<img src="docs/img/badges.svg">

Typed layer for Service Workers and Web Workers in WebTyp.

> **Write Go. The framework generates the JS shim.**

## Overview

`webtyp/js` is the **only JS API** in the WebTyp framework — mirroring what `webtyp/css` does for stylesheets. SSR modules call typed constructors returning `*Script` values with final JS content; `sitec` writes them to disk without additional coordination.

## API

### Script (v1 escape hatch)

The core `Script` type is available for arbitrary JS snippets:

```go
type Script struct {
    Name    string // Empty → bundled into /script.js. Non-empty → standalone /public/<Name>.
    Content string // Raw JavaScript.
}
```

### Runtime configuration (call once at boot)

```go
js.SetRuntime(js.RuntimeGo)    // Standard Go compiler
js.SetRuntime(js.RuntimeTinyGo) // TinyGo compiler (smaller binaries)
```

`webtyp/app` calls this automatically when it detects the compiler mode — **user modules never call it directly**.

### Typed constructors (recommended)

```go
// Bundles wasm_exec.js + WASM bootstrap into /script.js.
js.PageBootstrap() *Script

// Generates /sw.js with wasm_exec.js + SW event listeners inlined.
js.ServiceWorker(handler ServiceWorkerHandler) *Script

// Generates /<name>: wasm_exec.js + a queue for early messages, loading the Worker's OWN
// binary wasmURL (e.g. "/worker.wasm"), never the page's client.wasm.
js.WebWorker(name, wasmURL string) *Script
```

### Handler interfaces

Implement in Go — the shim bridges browser events to your methods:

```go
import (
    "webtyp.com/context" // stdlib context is vetoed in WASM
    "webtyp.com/fetch"
)

type ServiceWorkerHandler interface {
    OnInstall(ctx context.Context) error
    OnActivate(ctx context.Context) error
    OnFetch(ctx context.Context, req *js.Request) (*fetch.Response, error)
}

type WebWorkerHandler interface {
    OnMessage(ctx context.Context, msg *js.Message) (*js.Message, error)
}
```

### Event types

```go
// Request is the inbound FetchEvent intercepted by the SW.
type Request struct {
    URL     string
    Method  string
    Headers []fetch.Header // {Key, Value string} — no maps
    Body    []byte
}

// Message is the postMessage payload for Web Workers.
type Message struct {
    Data []byte
}
```

## Service Worker example (PWA)

```go
package mymodule

import (
    "webtyp.com/context"
    "webtyp.com/fetch"
    "webtyp.com/js"
)

type CachingSW struct{}

func (sw *CachingSW) OnInstall(ctx context.Context) error  { return nil }
func (sw *CachingSW) OnActivate(ctx context.Context) error { return nil }

func (sw *CachingSW) OnFetch(ctx context.Context, req *js.Request) (*fetch.Response, error) {
    // Only intercept API routes; fall through for statics.
    body := []byte(`{"cached": true}`)
    return fetch.NewResponse(200,
        []fetch.Header{{Key: "Content-Type", Value: "application/json"}},
        body,
    ), nil
}

func (m Module) RenderJS() []*js.Script {
    return []*js.Script{
        js.PageBootstrap(),          // wasm_exec + bootstrap → bundled in /script.js
        js.ServiceWorker(&CachingSW{}), // full shim → /sw.js
    }
}
```

## Web Worker example

A Web Worker runs its **own binary**, so heavy work (model inference, parsing) never blocks the
page and can be compiled for speed while the page stays compiled for size. Three pieces, one per
place they run:

```go
// 1. SSR module (host): publish the Worker script.
func (m Module) RenderJS() []*js.Script {
    return []*js.Script{js.WebWorker("parser.worker.js", "/parser.wasm")}
}

// 2. The Worker binary (parser.wasm, its own main): answer messages, in order.
type ParserWorker struct{}

func (ParserWorker) OnMessage(ctx *context.Context, msg *js.Message) (*js.Message, error) {
    return &js.Message{Data: process(msg.Data)}, nil // nil reply = nothing sent back
}

func main() { js.ServeWorker(ParserWorker{}) } // never returns

// 3. The page binary (client.wasm): talk to it.
w := js.NewWorker("parser.worker.js", func(reply *js.Message, err error) { /* … */ })
w.Post(&js.Message{Data: input}) // bytes are transferred, not copied again
```

Messages that reach the Worker before its binary has started are queued and delivered in order.
A handler error arrives at the page as `err`.

## Updating wasm_exec.js

When TinyGo or Go releases a new version:

1. Update `DefaultVersion` in `webtyp/tinygo/tinygo.go` (for TinyGo).
   For Go, update the `go` directive in `js/go.mod`.
2. Run: `go generate ./...`
3. Run: `go test ./...`   (first run updates assets if generate was skipped; second run must pass)
4. Publish: `gopush` (webtyp/tinygo first, then webtyp/js)

## Escape hatch

For raw JS snippets (analytics, polyfills, init scripts):

```go
// Bundled into /script.js
&js.Script{Content: "console.log('init')"}

// Written as standalone file
&js.Script{Name: "analytics.js", Content: analyticsJS}
```

## Stdlib constraints

`webtyp/js` compiles to WASM — the Go stdlib is **vetoed** to keep binary size minimal.

| Stdlib (prohibited) | Replacement |
|---|---|
| `context` | `webtyp.com/context` |
| `fmt`, `errors`, `strings`, `strconv`, `path` | `webtyp.com/fmt` |
| `encoding/json` | `webtyp.com/json` |
| `time` | `webtyp.com/time` |
| `map[string]string` (headers) | `[]fetch.Header` |
