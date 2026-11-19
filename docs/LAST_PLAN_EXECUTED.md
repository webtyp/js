---
PLAN: "feat: preload external translation dictionary in PageBootstrap before starting WASM"
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to proceed or finalize — nobody answers, and a session that waits ends without your work.

# Plan: `js.PageBootstrap` precarga el diccionario externo en paralelo con WASM

Master: `SECURITY_OBSERVABILITY_MASTER_PLAN.md` (D3, O3b-4).

## Tools

```bash
export GOPRIVATE=webtyp.com/*,github.com/veltylabs/*
export GOFLAGS=-mod=mod
go install webtyp.com/devflow/cmd/gotest@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Ecosystem search

- Buscado en `~/Dev/Project/webtyp` (`js`, `sitec`, `lang`).
- `js.PageBootstrap(wasmURL string) *Script` (`js/js.go:96`) es el único punto de arranque del navegador que descarga e instancia el binario WASM antes de llamar `go.run(result.instance)`.
- **Decisión:** extender `PageBootstrap` para que, si existe `<meta name="webtyp-lang" content="...">`, descargue el JSON en paralelo con `browser.wasm` y lo deje en `globalThis.__webtypLang` antes de `go.run`.

## Design gate

No cambia ninguna firma pública de `webtyp.com/js` (`PageBootstrap(wasmURL string) *Script` se mantiene intacta). El acoplamiento con `lang` se limita al contrato de `<meta name="webtyp-lang">` y `globalThis.__webtypLang` (mismos nombres que `lang.MetaName` y `lang.GlobalDictKey`).

## APIs used

Ninguna dependencia externa nueva.

## Reglas de entrega y anti-footguns

- El JS generado debe ser compatible con `strict-dynamic` y sin `eval`/`Function`.
- Si no hay `<meta name="webtyp-lang">` (p. ej. app solo en inglés) o si el fetch falla, `go.run(result.instance)` igual debe ejecutarse sin bloquear el arranque.
- Hacer commit después de cada etapa. No usar `git reset`, `git restore`, `git checkout -- <file>` ni `git stash`.

---

## Etapa 1: Pruebas (TDD)

En `tests/wasm_exec_test.go` (y registrar en `docs/TESTS.md` bajo `| Test | Rule |`):
- `TestPageBootstrap_PreloadsLangMeta`: verificar que `js.PageBootstrap(js.DefaultWasmURL).Content` consulta `meta[name="webtyp-lang"]`, asigna `self.__webtypLang` y usa `Promise.all` antes de `go.run(result.instance)`.

## Etapa 2: Actualizar `PageBootstrap` en `js.go`

En `js.go:108-125`, reemplazar el bloque de arranque en `Window` por:

```js
if (self.constructor.name === "Window") {
const go = new Go();
const langMeta = document.querySelector('meta[name="webtyp-lang"]');
const langPromise = (langMeta && langMeta.content)
	? fetch(langMeta.content).then(r => r.ok ? r.json() : null).then(d => { if (d) self.__webtypLang = d; }).catch(() => {})
	: Promise.resolve();
const wasmPromise = WebAssembly.instantiateStreaming
	? WebAssembly.instantiateStreaming(fetch("` + wasmURL + `"), go.importObject)
	: fetch("` + wasmURL + `").then(r => r.arrayBuffer()).then(b => WebAssembly.instantiate(b, go.importObject));
Promise.all([wasmPromise, langPromise]).then(([result]) => {
	go.run(result.instance);
});
}
```

## Etapa 3: Verificación y documentación

1. Actualizar `README.md` y `docs/TESTS.md` si aplica.
2. Ejecutar `gotest`.

## Tabla de etapas

| Etapa | Archivos | Qué |
|---|---|---|
| 1 | `tests/wasm_exec_test.go`, `docs/TESTS.md` | Prueba TDD de precarga |
| 2 | `js.go` | Precarga paralela de `meta[name="webtyp-lang"]` en `PageBootstrap` |
| 3 | `README.md`, `docs/TESTS.md` | Documentación y `gotest` verde |
