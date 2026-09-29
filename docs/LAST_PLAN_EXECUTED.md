# Plan (executed locally) — typed Web Worker messaging with the Worker's own binary

## Context

`js.WebWorker(name, handler)` generated a Worker script that loaded the page's `client.wasm` and
forwarded messages to a global `__webtyp_worker_message`. The message payload was never converted
(`// TODO: convert args[1] (js.Value) to []byte`), replies were never posted, and the handler was
registered in the SSR process, which is not the process that runs in the Worker. The agent's
model runs in a Worker built for speed (SIMD, `-opt=2`; see `webtyp/nn/docs/SIMD.md`), so the
Worker needs its own binary and working byte messages.

## Design gate (api-design)

1. **Prior art.** The Web Workers API (`new Worker(url)`, `postMessage(data, transfer)`,
   `onmessage`); **Comlink** (RPC over postMessage with transferables); **wasm-bindgen-rayon** and
   **wllama**, which run a separate wasm module per Worker. We keep the platform's shape (post
   bytes, receive bytes), with transfer, and one binary per Worker.
2. **Novice-name test.** `js.WebWorker(name, wasmURL)`, `js.ServeWorker(handler)`,
   `js.NewWorker(url, onReply)`, `(*Worker).Post`, `(*Worker).Terminate`. They are the platform's
   own verbs.
3. **Complexity ledger.** Concepts +3 (`ServeWorker`, `NewWorker`/`Worker`, `Post`) / −2 (the
   global dispatcher and `workerHandlers` map). Ways to do the same thing +0 / −1.
4. **Where it belongs.** `webtyp/js` is the only JS API of the framework.
5. **What it deletes.** `workerHandlers`, `__webtyp_worker_message`, `workerMessage`, and the
   handler parameter of `WebWorker`.

## Result

`worker_wasm.go` (`ServeWorker`, `NewWorker`, `Post`, `Terminate`, byte conversion);
`WebWorker(name, wasmURL)`; tests for the script (own binary, early-message queue) and for the
byte round trip in wasm; `tests/wasm_exec_sync_test.go` tagged `!wasm` (it downloads TinyGo, so it
is host-only).
