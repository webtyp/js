package js_test

import (
	"testing"

	. "webtyp.com/fmt"
	"webtyp.com/js"
)

// Canonical spec for the runtime selection + JS composition exposed by
// webtyp/js. These tests define what the implementation must satisfy
// once js/docs/PLAN.md stages 5-6 land.
//
// The wasm_exec.js content getters are intentionally NOT public — behavior
// is verified through the public composers (PageBootstrap / WebWorker),
// which inline the runtime selected via SetRuntime.

// Signatures that distinguish Go's wasm_exec.js (declared inline; the public
// API does not export signature lists).
var goRuntimeSignatures = []string{
	"runtime.scheduleTimeoutEvent",
	"runtime.clearTimeoutEvent",
	"runtime.wasmExit",
}

// Signatures that distinguish TinyGo's wasm_exec.js.
var tinyGoRuntimeSignatures = []string{
	"runtime.sleepTicks",
	"runtime.ticks",
	"gojs",
}

func TestPageBootstrap_IsBundleScript(t *testing.T) {
	s := js.PageBootstrap(js.DefaultWasmURL)
	if s.Name != "" {
		t.Errorf("PageBootstrap().Name = %q, want \"\" (goes into /script.js bundle)", s.Name)
	}
	if s.Content == "" {
		t.Fatal("PageBootstrap().Content is empty")
	}
}

func TestPageBootstrap_ReferencesClientWasm(t *testing.T) {
	c := js.PageBootstrap(js.DefaultWasmURL).Content
	if !Contains(c, "WebAssembly.instantiateStreaming") {
		t.Error("PageBootstrap() missing WebAssembly.instantiateStreaming")
	}
	if !Contains(c, "/client.wasm") {
		t.Error("PageBootstrap() must fetch /client.wasm")
	}
}

func TestPageBootstrap_InlinesRuntimePerSetRuntime(t *testing.T) {
	t.Cleanup(func() { js.SetRuntime(js.RuntimeGo) })

	js.SetRuntime(js.RuntimeTinyGo)
	tiny := js.PageBootstrap(js.DefaultWasmURL).Content
	for _, sig := range tinyGoRuntimeSignatures {
		if !Contains(tiny, sig) {
			t.Errorf("after SetRuntime(TinyGo), bootstrap missing TinyGo signature %q", sig)
		}
	}
	for _, sig := range goRuntimeSignatures {
		if Contains(tiny, sig) {
			t.Errorf("after SetRuntime(TinyGo), bootstrap unexpectedly contains Go signature %q", sig)
		}
	}

	js.SetRuntime(js.RuntimeGo)
	go_ := js.PageBootstrap(js.DefaultWasmURL).Content
	for _, sig := range goRuntimeSignatures {
		if !Contains(go_, sig) {
			t.Errorf("after SetRuntime(Go), bootstrap missing Go signature %q", sig)
		}
	}
}

// A release build passes the content-hashed name; the bootstrap must load exactly that binary.
func TestPageBootstrap_UsesGivenURL(t *testing.T) {
	c := js.PageBootstrap("/client.3f9a1c2b.wasm").Content
	if Count(c, `fetch("/client.3f9a1c2b.wasm")`) != 2 {
		t.Error("both fetch paths (streaming and fallback) must load the given URL")
	}
	if Contains(c, js.DefaultWasmURL) {
		t.Error("bootstrap still references the default URL")
	}
}

func TestWebWorker_UsesGivenName(t *testing.T) {
	s := js.WebWorker("parser.worker.js", "/parser.wasm")
	if s.Name != "parser.worker.js" {
		t.Errorf("WebWorker().Name = %q, want \"parser.worker.js\"", s.Name)
	}
}

// The Worker runs its own binary (compiled for speed), never the page's client.wasm, and
// queues messages that arrive before that binary calls ServeWorker.
func TestWebWorker_LoadsItsOwnBinaryAndQueuesEarlyMessages(t *testing.T) {
	c := js.WebWorker("model.worker.js", "/worker.wasm").Content
	if !Contains(c, `fetch("/worker.wasm")`) {
		t.Error("worker script must fetch its own wasm URL")
	}
	if Contains(c, "/client.wasm") {
		t.Error("worker script must not load the page binary client.wasm")
	}
	if !Contains(c, "__webtyp_worker_queue") || !Contains(c, "self.onmessage") {
		t.Error("worker script must queue messages until ServeWorker takes over")
	}
}
