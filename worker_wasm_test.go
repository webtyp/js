//go:build wasm

package js

import "testing"

func TestBytesRoundTripThroughJS(t *testing.T) {
	in := []byte{0, 1, 2, 250, 255}
	out := jsToBytes(bytesToJS(in))
	if string(out) != string(in) {
		t.Fatalf("round trip = %v, want %v", out, in)
	}
}

func TestJsToBytes_AcceptsArrayBuffer(t *testing.T) {
	u8 := bytesToJS([]byte("hola"))
	if got := string(jsToBytes(u8.Get("buffer"))); got != "hola" {
		t.Fatalf("from ArrayBuffer = %q, want \"hola\"", got)
	}
}
