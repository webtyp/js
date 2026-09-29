//go:build wasm

package js

import (
	"syscall/js"

	"webtyp.com/context"
)

func init() {
	js.Global().Set("__webtyp_sw_install", js.FuncOf(swInstall))
	js.Global().Set("__webtyp_sw_activate", js.FuncOf(swActivate))
	js.Global().Set("__webtyp_sw_fetch", js.FuncOf(swFetch))
}

func swInstall(this js.Value, args []js.Value) any {
	if swHandler == nil {
		return nil
	}
	// Simplified: in a real implementation we would handle the promise
	err := swHandler.OnInstall(context.Background())
	if err != nil {
		return err.Error()
	}
	return nil
}

func swActivate(this js.Value, args []js.Value) any {
	if swHandler == nil {
		return nil
	}
	err := swHandler.OnActivate(context.Background())
	if err != nil {
		return err.Error()
	}
	return nil
}

func swFetch(this js.Value, args []js.Value) any {
	if swHandler == nil {
		// Default behavior: fetch from network
		return js.Global().Call("fetch", args[0])
	}

	reqJS := args[0]
	req := &Request{
		URL:    reqJS.Get("url").String(),
		Method: reqJS.Get("method").String(),
	}
	// TODO: headers and body

	resp, err := swHandler.OnFetch(context.Background(), req)
	if err != nil {
		return js.Global().Call("fetch", args[0])
	}
	if resp == nil {
		return js.Global().Call("fetch", args[0])
	}

	// Convert fetch.Response to JS Response
	// This is a complex part that would involve NewResponse in JS
	return nil // Placeholder
}
