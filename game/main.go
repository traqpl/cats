//go:build js && wasm

package main

import (
	"syscall/js"
)

var engine *Engine

func main() {
	canvas := js.Global().Get("document").Call("getElementById", "gameCanvas")
	engine = NewEngine(canvas)
	engine.registerInput()

	// Expose state to JS (audio, etc.)
	js.Global().Set("purrcareScene", engine.audioScene())
	js.Global().Set("purrcareState", engine.stateName())

	var lastTime float64
	var loop js.Func
	loop = js.FuncOf(func(_ js.Value, args []js.Value) any {
		now := args[0].Float()
		if lastTime == 0 {
			lastTime = now
		}
		dt := (now - lastTime) / 1000.0
		if dt > 0.1 {
			dt = 0.1
		}
		lastTime = now

		engine.Update(dt)
		engine.Render()

		js.Global().Set("purrcareScene", engine.audioScene())
		js.Global().Set("purrcareState", engine.stateName())
		if engine.soundEvent != "" {
			js.Global().Set("purrcareSound", engine.soundEvent)
			engine.soundEvent = ""
		}

		js.Global().Call("requestAnimationFrame", loop)
		return nil
	})

	js.Global().Call("requestAnimationFrame", loop)
	select {}
}
