# Architecture — webtyp/js

## Flujo general

```mermaid
flowchart TD
    APP["webtyp/app (boot)\njs.SetRuntime(RuntimeGo | RuntimeTinyGo)"]

    APP --> PB["js.PageBootstrap(wasmURL)"]
    APP --> WW["js.WebWorker(name, wasmURL)"]

    RT[("activeRuntime\n(global)")]
    PB -.lee.- RT
    WW -.lee.- RT

    PB --> S1["*Script{Name: ''}\n→ bundle /script.js"]
    WW --> S3["*Script{Name: name}\n→ standalone /name"]

    S1 --> AM["sitec\nContent final — escribe a disco"]
    S3 --> AM
```

## Contextos de ejecución WASM

La página carga su binario (`client.wasm` en desarrollo, `client.<hash>.wasm` en release); cada **Web Worker carga su propio binario** (`wasmURL`), para que el trabajo pesado (p. ej. inferencia de modelos) se compile para velocidad sin engordar la página. Cada contexto es un scope JS aislado — no pueden compartir la instancia WASM entre sí.

```mermaid
flowchart LR
    subgraph WIN["Browser Window (DOM)"]
        direction TB
        S1["/script.js\nwasm_exec inline + bootstrap\ndetector: 'Window'"]
        W1["client.wasm\ninstancia A"]
        S1 --> W1
        W1 --> D1["init() DOM\njs.NewWorker(...).Post"]
    end

    subgraph WW["DedicatedWorkerGlobalScope"]
        direction TB
        S3["/parser.worker.js\nwasm_exec inline + cola de mensajes\n+ fetch(wasmURL)"]
        W3["parser.wasm\nbinario propio"]
        S3 --> W3
        W3 --> D3["js.ServeWorker(handler)\nvacía la cola y atiende onmessage"]
    end
```

El shim detecta el contexto vía `self.constructor.name` para evitar ejecutar código DOM en un Worker. El service worker del shell no está aquí: lo genera `webtyp/pwa` vía `sitec`.

## Separación de responsabilidades

| Paquete | Responsabilidad |
|---|---|
| `webtyp/js` | Composición JS (shims, embeds, constructores tipados). **Única fuente de wasm_exec.js** |
| `webtyp/app` | Orquestación: llama `js.SetRuntime`, registra `js.PageBootstrap(js.DefaultWasmURL)` con `sitec` en desarrollo |
| `webtyp/sitec` | Compilación WASM (Go/TinyGo) vía `WasmBuilder`, y bundling: recibe `[]*js.Script` con `Content` final y escribe a disco. **Sin JS propio** |

## Registro de handlers (lado WASM)

```mermaid
flowchart TD
    WW[host SSR: js.WebWorker name, wasmURL] -->|escribe| WS[script del Worker: cola + fetch wasmURL]
    WS --> WB[binario del Worker: main llama js.ServeWorker handler]
    WB -->|respuesta data + error, buffer transferido| PG[página: js.NewWorker onReply]
    PG -->|Post: bytes transferidos| WB
```

El handler de un Web Worker se registra **dentro de su propio binario** (`js.ServeWorker`), no en
el proceso SSR: ese proceso solo genera el script. Los mensajes son `[]byte`
(`js.Message.Data`); al cruzar a JS se copian una vez a un `Uint8Array` y su buffer se
**transfiere** (`postMessage(data, [buffer])`), sin segunda copia.

## Decisiones clave

- **`activeRuntime` es estado global write-once**: `app` lo escribe una vez al boot antes de que los módulos llamen `RenderJS()`. El extractor SSR de sitec corre en el mismo proceso → ve el global. Sin parámetros para el usuario.
- **`Content` es string final**: `sitec` lo escribe tal cual. Sin interfaces extra, sin resolución diferida. Idéntico al modelo de `webtyp/css`.
- **Sin `wasm_exec.js` en disco**: el archivo se inlinea en cada shim. No hay ruta `/wasm_exec.js` pública.
- **`client/assets/` eliminado**: `js/assets/` es la única fuente de verdad. Un solo lugar a actualizar cuando Go o TinyGo publican nuevas versiones del runtime.
