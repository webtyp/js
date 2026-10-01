//go:build !tinygo

// TinyGo's wasm target cannot recover from a panic, so this check runs under the Go compiler
// only (host and GOOS=js); the behavior it proves is the same for both compilers.
package js_test

import (
	"testing"

	. "webtyp.com/fmt"
	"webtyp.com/js"
)

// An empty URL is a programming error and says what to pass.
func TestPageBootstrap_EmptyURLPanics(t *testing.T) {
	defer func() {
		r := recover()
		msg, _ := r.(string)
		if !Contains(msg, "js.DefaultWasmURL") {
			t.Errorf("panic = %v, want a message naming js.DefaultWasmURL", r)
		}
	}()
	js.PageBootstrap("")
}
