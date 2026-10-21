# AGENTS.md — `webtyp/js`

<!-- webtyp:base — copied from devskills skill agents-md; do not edit here -->
## Ecosystem rules — read before any change

`docs/PLAN.md` is the spec: no design judgment; anything unclear is a question in the PR, not a guess.

- Public API: typed (no `any`/`interface{}`), explicit, minimal; never export a symbol only a test uses.
- One intent, one path: the old path is deleted in the same change. No silent fallback, `TODO`, commented code or stub.
- Fail at compile time, else a loud diagnostic — never silently. Zero values deny.
- A defect in a dependency is reported, never wrapped or patched here.
- No string literals in logic: named constants. The same logic never lives in two repos.
- `cmd/*/main.go`: parse flags, inject, print/exit only. stdout = data, stderr = logs, exit codes are the contract.
- WASM code: `webtyp/fmt`, `webtyp/time`, `webtyp/json` instead of stdlib; no `map`; `dom.Element` embedded by value.
- `css.go`, `svg.go`, `js.go`, `html.go` carry `//go:build !wasm`; no `ssr.go`, no `front.go`.
- Code and keys in English; UI text is `lang.Text`, translations live in `lang.json`.
- Tests cover behaviour and domain rules, never the look (CSS, colours, markup order, class strings). In `tests/` (external package), real stack with fakes only at the edge; run `gotest`.
- Docs (`README.md`, `docs/*`) updated with the API; permanent docs never link `docs/PLAN.md`.
- Before the PR: `git diff --stat main` lists every file of the plan's stages, and nothing the plan did not order is deleted.
<!-- /webtyp:base -->

## This repo

- `webtyp/js` is the single source of truth for `wasm_exec.js` (`assets/wasm_exec_go.js`, `assets/wasm_exec_tinygo.js`), `PageBootstrap(wasmURL)` and Web Worker shims (`WebWorker`, `ServeWorker`, `NewWorker`).
- Files compiled into WASM (`worker_wasm.go`, `worker_native.go`) must not import vetoed Go stdlib packages (`fmt`, `strings`, `strconv`, `errors`, `context`, `encoding/json`, `time`); use `webtyp.com/fmt` and `webtyp.com/context`.
- `scripts/update_wasm_exec.go` (`//go:build ignore`) is build-time tooling and legitimately uses the standard library.

