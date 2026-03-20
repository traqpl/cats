//go:build js && wasm

package main

import "syscall/js"

func (e *Engine) registerInput() {
	doc := js.Global().Get("document")

	// Keyboard
	doc.Call("addEventListener", "keydown",
		js.FuncOf(func(_ js.Value, args []js.Value) any {
			code := args[0].Get("code").String()
			switch code {
			case "KeyP", "Escape":
				switch e.state {
				case StatePlaying:
					e.state = StatePaused
				case StatePaused:
					e.state = StatePlaying
				}
			case "KeyQ":
				if e.state == StatePaused {
					e.enterMainMenu()
				}
			case "Space", "Enter":
				switch e.state {
				case StateMainMenu:
					e.newGame()
				case StateNight:
					if e.nightDone {
						e.nextDay()
					}
				case StateAlert:
					e.dismissAlert()
				case StateGameOver, StateVictory:
					e.enterMainMenu()
				}
			case "KeyS":
				if e.state == StateMainMenu {
					// Cycle speed (x5=3min/day, x8=1.75min, x12=1.2min)
					switch e.speed {
					case 5:
						e.speed = 8
					case 8:
						e.speed = 12
					default:
						e.speed = 5
					}
				}
			}
			switch code {
			case "Space", "Enter", "KeyP", "KeyQ", "KeyS", "Escape":
				args[0].Call("preventDefault")
			}
			return nil
		}),
	)

	// Mouse move — hover detection
	e.canvas.Call("addEventListener", "mousemove",
		js.FuncOf(func(_ js.Value, args []js.Value) any {
			evt := args[0]
			mx, my := e.canvasXY(evt)
			for i := range e.objects {
				e.objects[i].Hovered = e.objHit(i, mx, my)
			}
			return nil
		}),
	)

	// Click
	e.canvas.Call("addEventListener", "click",
		js.FuncOf(func(_ js.Value, args []js.Value) any {
			evt := args[0]
			mx, my := e.canvasXY(evt)
			e.handleClick(mx, my)
			return nil
		}),
	)

	// Touch support
	e.canvas.Call("addEventListener", "touchend",
		js.FuncOf(func(_ js.Value, args []js.Value) any {
			args[0].Call("preventDefault")
			evt := args[0]
			touches := evt.Get("changedTouches")
			if touches.Length() > 0 {
				mx, my := e.canvasXY(touches.Index(0))
				e.handleClick(mx, my)
			}
			return nil
		}),
	)
}

func (e *Engine) handleClick(mx, my float64) {
	switch e.state {
	case StateMainMenu:
		e.newGame()
	case StateAlert:
		e.dismissAlert()
	case StateNight:
		if e.nightDone {
			e.nextDay()
		}
	case StateGameOver, StateVictory:
		e.enterMainMenu()
	case StatePlaying:
		e.handlePlayClick(mx, my)
	}
}

func (e *Engine) handlePlayClick(mx, my float64) {
	// Click on cat → pet it
	if e.catHit(mx, my) {
		e.needs.Social = clamp01(e.needs.Social + 0.25)
		e.needs.Fun = clamp01(e.needs.Fun + 0.05)
		e.score += ScorePet
		e.spawnHearts()
		e.cat.State = CatPetting
		e.cat.StateTime = 1.5
		e.cat.TargetObj = -1
		return
	}

	// Click on object
	for i := range e.objects {
		if !e.objHit(i, mx, my) {
			continue
		}
		obj := &e.objects[i]
		switch obj.Name {
		case "food":
			if obj.Filled <= 0 {
				e.refillObject(i)
			} else {
				// Direct player action: send cat immediately
				e.sendCatTo(i)
				e.setFlash("Going to eat! 🍽")
			}
		case "water":
			if obj.Filled <= 0 {
				e.refillObject(i)
			} else {
				e.sendCatTo(i)
				e.setFlash("Going to drink! 💧")
			}
		case "litter":
			if obj.Visits >= obj.Cap {
				e.refillObject(i) // "clean"
				e.setFlash("Litter box cleaned! +" + itoa(ScoreCleanLitter))
			} else {
				e.sendCatTo(i)
				e.setFlash("Using litter box…")
			}
		case "toy":
			e.sendCatTo(i)
			e.setFlash("Playtime! 🧸")
		case "bed":
			e.sendCatTo(i)
			e.setFlash("Nap time… 😴")
		case "scratcher":
			e.sendCatTo(i)
			e.setFlash("Scratch scratch! 🌿")
		case "brush":
			e.sendCatTo(i)
			e.setFlash("Brushing time! 🪮")
		}
		return
	}
}

func (e *Engine) dismissAlert() {
	e.activeAlert = nil
	e.state = StatePlaying
}

// canvasXY converts a browser event clientX/Y to canvas coordinates.
func (e *Engine) canvasXY(evt js.Value) (float64, float64) {
	rect := e.canvas.Call("getBoundingClientRect")
	scaleX := e.canvas.Get("width").Float() / rect.Get("width").Float()
	scaleY := e.canvas.Get("height").Float() / rect.Get("height").Float()
	x := (evt.Get("clientX").Float() - rect.Get("left").Float()) * scaleX
	y := (evt.Get("clientY").Float() - rect.Get("top").Float()) * scaleY
	return x, y
}

// objHit returns true if (mx,my) is within the object's bounding box,
// or within the refill/clean button drawn above it when empty.
func (e *Engine) objHit(i int, mx, my float64) bool {
	obj := &e.objects[i]
	x0 := obj.X - obj.W/2
	x1 := obj.X + obj.W/2
	y0 := obj.FootY - obj.H
	y1 := obj.FootY
	if mx >= x0 && mx <= x1 && my >= y0 && my <= y1 {
		return true
	}
	// Also hit the refill button (drawn above empty bowls/litter)
	if obj.Filled <= 0 {
		var btnTop float64
		if obj.Name == "litter" {
			btnTop = obj.FootY - obj.H - 14
		} else {
			btnTop = obj.FootY - 42
		}
		if mx >= obj.X-31 && mx <= obj.X+31 && my >= btnTop-44 && my <= btnTop {
			return true
		}
	}
	return false
}

// catHit returns true if (mx,my) is within the cat's bounding box.
func (e *Engine) catHit(mx, my float64) bool {
	cx := e.cat.X
	cy := e.cat.Y
	return mx >= cx-30 && mx <= cx+30 && my >= cy-70 && my <= cy
}
