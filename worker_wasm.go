//go:build wasm

package js

import (
	"sync"
	"syscall/js"

	"webtyp.com/context"
)

// Reply envelope fields: the Worker answers with {data: Uint8Array | null, error: string}.
const (
	replyDataField  = "data"
	replyErrorField = "error"
)

// ServeWorker makes the current binary answer the messages of the Web Worker it runs in, with
// h, one message at a time and in arrival order. Call it from the Worker binary's main; it
// never returns. Replies are posted back with their buffer transferred, not copied.
func ServeWorker(h WebWorkerHandler) {
	self := js.Global()

	var mu sync.Mutex
	var inbox [][]byte
	wake := make(chan struct{}, 1)
	enqueue := func(v js.Value) {
		mu.Lock()
		inbox = append(inbox, jsToBytes(v))
		mu.Unlock()
		select {
		case wake <- struct{}{}:
		default:
		}
	}

	// Messages queued by the Worker script before this binary started, then the live ones.
	if q := self.Get(workerQueueName); q.Truthy() {
		for i := 0; i < q.Length(); i++ {
			enqueue(q.Index(i))
		}
	}
	self.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) any {
		enqueue(args[0].Get("data"))
		return nil
	}))

	for {
		<-wake
		for {
			mu.Lock()
			if len(inbox) == 0 {
				mu.Unlock()
				break
			}
			data := inbox[0]
			inbox = inbox[1:]
			mu.Unlock()

			reply, err := h.OnMessage(context.Background(), &Message{Data: data})
			postReply(self, reply, err)
		}
	}
}

// PostToPage sends msg from inside the Worker to the page outside any reply: progress while a long
// message is being handled, or news the page did not ask for. The page receives it through the
// same onReply as a reply (NewWorker), so the application's own protocol tells them apart.
// Call it only from the Worker binary (after ServeWorker started).
func PostToPage(msg *Message) {
	if msg == nil {
		return
	}
	postReply(js.Global(), msg, nil)
}

func postReply(target js.Value, reply *Message, err error) {
	env := js.Global().Get("Object").New()
	errText := ""
	if err != nil {
		errText = err.Error()
	}
	env.Set(replyErrorField, errText)
	if reply == nil {
		if err == nil {
			return // nothing to say
		}
		env.Set(replyDataField, js.Null())
		target.Call("postMessage", env)
		return
	}
	u8 := bytesToJS(reply.Data)
	env.Set(replyDataField, u8)
	target.Call("postMessage", env, []any{u8.Get("buffer")})
}

// Worker is the page's handle on a Web Worker started from a script made by WebWorker.
type Worker struct {
	v       js.Value
	onReply js.Func
}

// NewWorker starts the Worker whose script is at scriptURL (the Name given to WebWorker) and
// calls onReply for every reply it sends: its message, or the error its handler returned.
func NewWorker(scriptURL string, onReply func(msg *Message, err error)) *Worker {
	w := &Worker{v: js.Global().Get("Worker").New(scriptURL)}
	w.onReply = js.FuncOf(func(this js.Value, args []js.Value) any {
		env := args[0].Get("data")
		var msg *Message
		if d := env.Get(replyDataField); d.Truthy() {
			msg = &Message{Data: jsToBytes(d)}
		}
		var err error
		if e := env.Get(replyErrorField); e.Truthy() && e.String() != "" {
			err = workerError(e.String())
		}
		onReply(msg, err)
		return nil
	})
	w.v.Set("onmessage", w.onReply)
	return w
}

// Post sends msg to the Worker. Its bytes are copied once into JavaScript and then transferred.
func (w *Worker) Post(msg *Message) {
	u8 := bytesToJS(msg.Data)
	w.v.Call("postMessage", u8, []any{u8.Get("buffer")})
}

// Terminate stops the Worker immediately and releases the reply callback.
func (w *Worker) Terminate() {
	w.v.Call("terminate")
	w.onReply.Release()
}

type workerError string

func (e workerError) Error() string { return string(e) }

// bytesToJS copies b into a new Uint8Array.
func bytesToJS(b []byte) js.Value {
	u8 := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(u8, b)
	return u8
}

// jsToBytes copies a Uint8Array or an ArrayBuffer into a new []byte.
func jsToBytes(v js.Value) []byte {
	if v.InstanceOf(js.Global().Get("ArrayBuffer")) {
		v = js.Global().Get("Uint8Array").New(v)
	}
	b := make([]byte, v.Get("length").Int())
	js.CopyBytesToGo(b, v)
	return b
}
