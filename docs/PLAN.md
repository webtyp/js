---
PLAN: "refactor: rename page binary URL from /client.wasm to /browser.wasm"
EXECUTOR: jules
REVIEWER: none
---

> Read `AGENTS.md` at the repo root before any change.

# `webtyp/js` — Default page binary URL `/browser.wasm`

Part of [BUILD_DESTINATIONS_MASTER_PLAN.md](https://github.com/webtyp/docs/blob/main/BUILD_DESTINATIONS_MASTER_PLAN.md) (D1, D3, D12, D15): every build destination in `web/` is named after its target (`web/browser.go` → `browser.wasm`, `web/edge.go` → `edge.wasm`, `web/server.go` → `server`). The page WASM binary served in development changes from `/client.wasm` to `/browser.wasm` (`browser.<hash>.wasm` in release).

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
```

## Design gate

1. **Prior art:** Node/bundler `package.json` conditional exports (`browser`, `workerd`, `node`) and WebTyp's `BUILD_DESTINATIONS` D1 (`web/browser.go` → `browser.wasm`, `web/edge.go` → `edge.wasm`).
2. **Novice-name test:** `js.DefaultWasmURL` stays the same identifier; its value `"/browser.wasm"` matches `web/browser.go` 1:1 without the ambiguous `"client"` word.
3. **Complexity ledger:** +0 symbols / −1 mismatch between source filename (`web/browser.go`) and emitted binary (`browser.wasm`).
4. **Where it belongs:** `webtyp/js` owns `DefaultWasmURL` and `PageBootstrap`.
5. **What it deletes:** every `client.wasm` / `client.<hash>.wasm` reference in `js.go`, `tests/wasm_exec_test.go`, `README.md`, and `docs/ARCHITECTURE.md` (no backwards-compatibility alias).

---

## Stage 1 — Update `DefaultWasmURL` and comments in `js.go`

File: `js.go`

1. Change lines 88–90 from:
   ```go
   // DefaultWasmURL is where the development build serves the page binary. Release builds pass the
   // content-hashed name sitec emits (client.3f9a1c2b.wasm).
   const DefaultWasmURL = "/client.wasm"
   ```
   to:
   ```go
   // DefaultWasmURL is where the development build serves the page binary. Release builds pass the
   // content-hashed name sitec emits (browser.3f9a1c2b.wasm).
   const DefaultWasmURL = "/browser.wasm"
   ```
2. Update the doc comment on `WebWorker` (line 130) from `not the page's client.wasm` to `not the page's browser.wasm`.

---

## Stage 2 — Update tests in `tests/wasm_exec_test.go`

File: `tests/wasm_exec_test.go`

1. Rename `TestPageBootstrap_ReferencesClientWasm` to `TestPageBootstrap_ReferencesBrowserWasm` and assert `Contains(c, "/browser.wasm")` (error message: `"PageBootstrap() must fetch /browser.wasm"`).
2. In `TestPageBootstrap_UsesGivenURL`, pass `"/browser.3f9a1c2b.wasm"` and assert `Count(c, `fetch("/browser.3f9a1c2b.wasm")`) == 2` and `!Contains(c, js.DefaultWasmURL)`.
3. In `TestWebWorker_LoadsItsOwnBinaryAndQueuesEarlyMessages`, update the comment and assertion from `"/client.wasm"` / `client.wasm` to `"/browser.wasm"` / `browser.wasm`.

Run `gotest ./...` and verify all tests pass.

---

## Stage 3 — Update `README.md` and `docs/ARCHITECTURE.md`

Files: `README.md`, `docs/ARCHITECTURE.md`

1. In `README.md`, replace `client.wasm` with `browser.wasm` (lines 42 and 93).
2. In `docs/ARCHITECTURE.md`, replace `client.wasm` with `browser.wasm` and `client.<hash>.wasm` with `browser.<hash>.wasm` (lines 25 and 32).

---

## Deletions and acceptance criteria

- `grep -rn --exclude="LAST_PLAN_EXECUTED.md" "client\.wasm\|client\.<hash>\.wasm\|client\.3f9a1c2b\.wasm" .` returns empty.
- `gotest ./...` passes with zero failures.

## Stages

| # | Files | Goal |
|---|---|---|
| 1 | `js.go` | Set `DefaultWasmURL = "/browser.wasm"` and update comments |
| 2 | `tests/wasm_exec_test.go` | Update unit tests to assert `/browser.wasm` and `/browser.3f9a1c2b.wasm` |
| 3 | `README.md`, `docs/ARCHITECTURE.md` | Replace `client.wasm` with `browser.wasm` in documentation |

