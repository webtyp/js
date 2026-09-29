//go:generate go run scripts/update_wasm_exec.go

package js

import (
	_ "embed"
	"sync"

	"webtyp.com/context"
	"webtyp.com/fetch"
	. "webtyp.com/fmt"
)

// Script represents a JS fragment produced by an SSR module.
// - Empty Name: Content is bundled into the global script.js.
// - Non-empty Name: Content is written as /public/<Name> (standalone file).
type Script struct {
	Name    string // Simple filename (e.g., "sw.js"); no separators or path traversals.
	Content string
}

// String returns the raw content (for parity with webtyp/css Stylesheet.String()).
func (s *Script) String() string {
	return s.Content
}

// validate ensures Name is a simple filename.
func (s *Script) validate() {
	if s.Name == "" {
		return
	}
	if Contains(s.Name, "/") || Contains(s.Name, "\\") || Contains(s.Name, "..") {
		panic("js: Script.Name must be a simple filename, got " + s.Name)
	}
}

// Request is the FetchEvent incoming intercepted by the SW (inbound).
type Request struct {
	URL     string
	Method  string
	Headers []fetch.Header
	Body    []byte
}

// Message is the payload of postMessage of a Web Worker.
type Message struct {
	Data []byte
}

// ServiceWorkerHandler is the logic for SW events.
type ServiceWorkerHandler interface {
	OnInstall(ctx *context.Context) error
	OnActivate(ctx *context.Context) error
	OnFetch(ctx *context.Context, req *Request) (*fetch.Response, error)
}

// WebWorkerHandler answers the messages a Web Worker receives. It runs inside the Worker's own
// binary, registered with ServeWorker. A nil reply sends nothing back; an error reaches the
// page's onReply as its error.
type WebWorkerHandler interface {
	OnMessage(ctx *context.Context, msg *Message) (*Message, error)
}

// Runtime represents the Go compiler used.
type Runtime int

const (
	RuntimeGo Runtime = iota
	RuntimeTinyGo
)

var (
	runtimeMu     sync.RWMutex
	activeRuntime = RuntimeGo
)

// SetRuntime selects which wasm_exec.js glue PageBootstrap, ServiceWorker and
// WebWorker embed. Safe to call at any time, including while a request for
// one of those is in flight on another goroutine — callers that switch
// compilers at runtime (e.g. the dev-mode TUI) rely on this.
func SetRuntime(r Runtime) {
	runtimeMu.Lock()
	activeRuntime = r
	runtimeMu.Unlock()
}

func currentRuntime() Runtime {
	runtimeMu.RLock()
	defer runtimeMu.RUnlock()
	return activeRuntime
}

var swHandler ServiceWorkerHandler

// --- Embeds ---

//go:embed assets/wasm_exec_go.js
var wasmExecGoSource string

//go:embed assets/wasm_exec_tinygo.js
var wasmExecTinyGoSource string

func wasmExecGo() string     { return wasmExecGoSource }
func wasmExecTinyGo() string { return wasmExecTinyGoSource }

const defaultWasmURL = "/client.wasm"

// PageBootstrap returns the entrypoint Script for the main page.
func PageBootstrap() *Script {
	s := &Script{Name: ""}
	s.validate()

	runtimeJS := wasmExecGo()
	if currentRuntime() == RuntimeTinyGo {
		runtimeJS = wasmExecTinyGo()
	}

	content := runtimeJS + `
if (self.constructor.name === "Window") {
const go = new Go();
if (WebAssembly.instantiateStreaming) {
	WebAssembly.instantiateStreaming(fetch("` + defaultWasmURL + `"), go.importObject).then((result) => {
		go.run(result.instance);
	});
} else {
	fetch("` + defaultWasmURL + `").then(response =>
		response.arrayBuffer()
	).then(bytes =>
		WebAssembly.instantiate(bytes, go.importObject)
	).then(result => {
		go.run(result.instance);
	});
}
}
`
	return &Script{Name: "", Content: content}
}

// ServiceWorker returns the standalone Script for the service worker.
func ServiceWorker(handler ServiceWorkerHandler) *Script {
	s := &Script{Name: "sw.js"}
	s.validate()

	swHandler = handler

	runtimeJS := wasmExecGo()
	if currentRuntime() == RuntimeTinyGo {
		runtimeJS = wasmExecTinyGo()
	}

	content := runtimeJS + `
const go = new Go();
WebAssembly.instantiateStreaming(fetch("` + defaultWasmURL + `"), go.importObject).then((result) => {
	go.run(result.instance);
});

self.addEventListener('install',  e => {
    if (self.__webtyp_sw_install) {
        e.waitUntil(self.__webtyp_sw_install());
    }
});
self.addEventListener('activate', e => {
    if (self.__webtyp_sw_activate) {
        e.waitUntil(self.__webtyp_sw_activate());
    }
});
self.addEventListener('fetch',    e => {
    if (self.__webtyp_sw_fetch) {
        e.respondWith(self.__webtyp_sw_fetch(e.request));
    }
});
`
	return &Script{Name: "sw.js", Content: content}
}

// WebWorker returns the standalone script that starts a Web Worker running its own binary,
// wasmURL (e.g. "/worker.wasm"), not the page's client.wasm: heavy work such as model inference
// is compiled separately, for speed. Inside that binary, js.ServeWorker(handler) answers the
// messages; the page talks to it with js.NewWorker(name, onReply). Messages that arrive while the
// binary is still starting are queued, not lost.
func WebWorker(name, wasmURL string) *Script {
	s := &Script{Name: name}
	s.validate()

	runtimeJS := wasmExecGo()
	if currentRuntime() == RuntimeTinyGo {
		runtimeJS = wasmExecTinyGo()
	}

	content := runtimeJS + `
self.` + workerQueueName + ` = [];
self.onmessage = e => self.` + workerQueueName + `.push(e.data);
const go = new Go();
WebAssembly.instantiateStreaming(fetch("` + wasmURL + `"), go.importObject).then((result) => {
	go.run(result.instance);
});
`
	return &Script{Name: name, Content: content}
}

// workerQueueName is the global where the Worker script queues messages until ServeWorker runs.
const workerQueueName = "__webtyp_worker_queue"
