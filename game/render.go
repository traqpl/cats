//go:build js && wasm

package main

import (
	"fmt"
	"math"
	"syscall/js"
)

// ── main dispatch ─────────────────────────────────────────────────────────────

func (e *Engine) Render() {
	w, h := canvasW, canvasH
	e.ctx.Call("clearRect", 0, 0, w, h)

	switch e.state {
	case StateMainMenu:
		e.renderMainMenu()
		return
	}

	// Base game view (all non-menu states)
	e.renderBackground()
	e.renderObjects()
	e.renderCat()
	e.renderNeedBars()
	e.renderHUD()
	e.renderFlash()

	// Overlays
	switch e.state {
	case StatePaused:
		e.renderPaused()
	case StateNight:
		e.renderNight()
	case StateAlert:
		e.renderBackground() // redraw so overlay is on top
		e.renderObjects()
		e.renderCat()
		e.renderNeedBars()
		e.renderHUD()
		e.renderAlert()
	case StateGameOver:
		e.renderGameOver()
	case StateVictory:
		e.renderVictory()
	}
}

// ── background / room ─────────────────────────────────────────────────────────

func (e *Engine) renderBackground() {
	ctx := e.ctx

	// Wall
	ctx.Set("fillStyle", colWall)
	ctx.Call("fillRect", roomX, roomY, roomW, roomH)

	// Window (sky changes with time)
	e.renderWindow()

	// Wallpaper dots (subtle decoration)
	ctx.Set("fillStyle", "rgba(180,150,100,0.15)")
	for xi := 0.0; xi < roomW; xi += 60 {
		for yi := roomY; yi < floorY-20; yi += 60 {
			ctx.Call("beginPath")
			ctx.Call("arc", xi+30, yi+30, 3, 0, math.Pi*2)
			ctx.Call("fill")
		}
	}

	// Floor
	grad := ctx.Call("createLinearGradient", 0, floorY-8, 0, floorY+20)
	grad.Call("addColorStop", 0, "#c9a96e")
	grad.Call("addColorStop", 1, "#a07840")
	ctx.Set("fillStyle", grad)
	ctx.Call("fillRect", 0, floorY-8, canvasW, 28)

	// Baseboard
	ctx.Set("fillStyle", "#8b5e30")
	ctx.Call("fillRect", 0, floorY+18, canvasW, 4)
}

func (e *Engine) renderWindow() {
	ctx := e.ctx
	wx, wy := 330.0, roomY+20.0
	ww, wh := 150.0, 130.0

	// Sky gradient based on game time
	skyTop, skyBot := e.skyColors()
	skyGrad := ctx.Call("createLinearGradient", wx, wy, wx, wy+wh)
	skyGrad.Call("addColorStop", 0, skyTop)
	skyGrad.Call("addColorStop", 1, skyBot)
	ctx.Set("fillStyle", skyGrad)
	roundRect(ctx, wx, wy, ww, wh, 4)
	ctx.Call("fill")

	// Sun or moon
	if e.gameTime < 720 { // before 20:00 → sun
		t := e.gameTime / 840.0
		sunX := wx + 20 + t*110
		sunY := wy + 25 - math.Sin(t*math.Pi)*20
		ctx.Set("fillStyle", "#ffe066")
		ctx.Call("beginPath")
		ctx.Call("arc", sunX, sunY, 12, 0, math.Pi*2)
		ctx.Call("fill")
	} else { // moon
		ctx.Set("fillStyle", "#e8e8d0")
		ctx.Call("beginPath")
		ctx.Call("arc", wx+110, wy+25, 10, 0, math.Pi*2)
		ctx.Call("fill")
		// crater
		ctx.Set("fillStyle", "rgba(0,0,0,0.1)")
		ctx.Call("beginPath")
		ctx.Call("arc", wx+114, wy+22, 4, 0, math.Pi*2)
		ctx.Call("fill")
	}

	// Simple landscape line
	ctx.Set("fillStyle", "#6a994e")
	ctx.Call("fillRect", wx, wy+wh-18, ww, 18)

	// Window frame
	ctx.Set("strokeStyle", "#8b6040")
	ctx.Set("lineWidth", 4)
	roundRect(ctx, wx, wy, ww, wh, 4)
	ctx.Call("stroke")
	// cross bars
	ctx.Set("lineWidth", 2)
	ctx.Call("beginPath")
	ctx.Call("moveTo", wx+ww/2, wy)
	ctx.Call("lineTo", wx+ww/2, wy+wh)
	ctx.Call("stroke")
	ctx.Call("beginPath")
	ctx.Call("moveTo", wx, wy+wh/2)
	ctx.Call("lineTo", wx+ww, wy+wh/2)
	ctx.Call("stroke")

	// Window sill
	ctx.Set("fillStyle", "#8b6040")
	ctx.Call("fillRect", wx-8, wy+wh, ww+16, 8)
}

func (e *Engine) skyColors() (string, string) {
	t := e.gameTime
	switch {
	case t < 60: // 08:00–09:00 early morning
		return "#f4a460", "#87ceeb"
	case t < 480: // 09:00–16:00 full day
		return "#5ba8e0", "#87ceeb"
	case t < 600: // 16:00–18:00 late afternoon
		return "#87ceeb", "#b0d8f0"
	case t < 720: // 18:00–20:00 sunset
		return "#f4814a", "#ffb347"
	case t < 780: // 20:00–21:00 dusk
		return "#4a3060", "#8060a0"
	default: // 21:00–22:00 night
		return "#1a1a4e", "#2a2a6e"
	}
}

// ── objects ───────────────────────────────────────────────────────────────────

func (e *Engine) renderObjects() {
	for i := range e.objects {
		e.renderObject(i)
	}
}

func (e *Engine) renderObject(i int) {
	ctx := e.ctx
	obj := &e.objects[i]
	cx := obj.X
	fy := obj.FootY

	// Hover glow
	if obj.Hovered {
		ctx.Set("shadowBlur", 14)
		ctx.Set("shadowColor", "rgba(255,220,80,0.8)")
	}

	switch obj.Name {
	case "scratcher":
		e.drawScratcher(cx, fy)
	case "food":
		e.drawBowl(cx, fy, "#cd853f", obj.Filled)
	case "water":
		e.drawBowl(cx, fy, "#4fc3f7", obj.Filled)
	case "toy":
		e.drawToy(cx, fy)
	case "bed":
		e.drawBed(cx, fy)
	case "litter":
		e.drawLitter(cx, fy, obj.Filled)
	case "brush":
		e.drawBrush(cx, fy)
	}

	if obj.Hovered {
		ctx.Set("shadowBlur", 0)
		ctx.Set("shadowColor", "transparent")
		// Tooltip
		e.drawTooltip(obj.Label, cx, fy-obj.H-10)
	}
}

func (e *Engine) drawScratcher(cx, fy float64) {
	ctx := e.ctx
	// Pole
	ctx.Set("fillStyle", "#8b6040")
	ctx.Call("fillRect", cx-8, fy-90, 16, 90)
	// Rope texture lines
	ctx.Set("strokeStyle", "#a07850")
	ctx.Set("lineWidth", 1.5)
	for y := fy - 85.0; y < fy-5; y += 8 {
		ctx.Call("beginPath")
		ctx.Call("moveTo", cx-8, y)
		ctx.Call("lineTo", cx+8, y)
		ctx.Call("stroke")
	}
	// Base
	ctx.Set("fillStyle", "#6a4020")
	ctx.Call("fillRect", cx-22, fy-12, 44, 12)
	// Top platform
	ctx.Set("fillStyle", "#6a4020")
	ctx.Call("fillRect", cx-18, fy-90, 36, 10)
}

func (e *Engine) drawBowl(cx, fy float64, color string, filled float64) {
	ctx := e.ctx
	bw := 56.0
	// Bowl shadow
	ctx.Set("fillStyle", "rgba(0,0,0,0.15)")
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx, fy-2, bw/2+4, 8, 0, 0, math.Pi*2)
	ctx.Call("fill")
	// Bowl body
	ctx.Set("fillStyle", "#d2a679")
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx, fy-10, bw/2, 18, 0, 0, math.Pi*2)
	ctx.Call("fill")
	// Content
	if filled > 0.05 {
		ctx.Set("fillStyle", color)
		r := (bw/2 - 6) * filled
		ctx.Call("beginPath")
		ctx.Call("ellipse", cx, fy-14, r, r*0.45, 0, 0, math.Pi*2)
		ctx.Call("fill")
	}
	// Rim highlight
	ctx.Set("strokeStyle", "#b8896a")
	ctx.Set("lineWidth", 2)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx, fy-10, bw/2, 18, 0, 0, math.Pi*2)
	ctx.Call("stroke")
	// Refill button when empty
	if filled <= 0 {
		e.drawRefillButton(cx, fy-42, "🥣", "Refill")
	}
}

func (e *Engine) drawToy(cx, fy float64) {
	ctx := e.ctx
	t := e.time
	// Stick
	ctx.Set("strokeStyle", "#8b6040")
	ctx.Set("lineWidth", 3)
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx, fy)
	ctx.Call("lineTo", cx, fy-75)
	ctx.Call("stroke")
	// String with swing
	swing := math.Sin(t*3) * 12
	ctx.Set("strokeStyle", "#c0a0a0")
	ctx.Set("lineWidth", 1.5)
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx, fy-75)
	ctx.Call("lineTo", cx+swing, fy-55)
	ctx.Call("stroke")
	// Feather
	for _, angle := range []float64{-0.3, 0, 0.3} {
		ctx.Set("strokeStyle", "#e87040")
		ctx.Set("lineWidth", 3)
		ctx.Call("beginPath")
		fx := cx + swing + math.Sin(angle)*14
		fy2 := fy - 55 + math.Cos(angle)*10
		ctx.Call("moveTo", cx+swing, fy-55)
		ctx.Call("lineTo", fx, fy2)
		ctx.Call("stroke")
	}
}

func (e *Engine) toyFeatherPose(cx float64) (anchorX, anchorY, featherX, featherY float64) {
	swing := math.Sin(e.time*3) * 12
	anchorX = cx
	anchorY = objFootY
	featherX = cx + swing
	featherY = objFootY - 55
	return
}

func (e *Engine) drawBed(cx, fy float64) {
	ctx := e.ctx
	bw := 100.0
	// Cushion
	ctx.Set("fillStyle", "#e8b4b8")
	roundRect(ctx, cx-bw/2, fy-42, bw, 38, 8)
	ctx.Call("fill")
	// Pillow
	ctx.Set("fillStyle", "#f8d8da")
	roundRect(ctx, cx-bw/2+6, fy-42, bw/3, 20, 6)
	ctx.Call("fill")
	// Rim
	ctx.Set("strokeStyle", "#c08080")
	ctx.Set("lineWidth", 2)
	roundRect(ctx, cx-bw/2, fy-42, bw, 38, 8)
	ctx.Call("stroke")
	// Zzz if cat is sleeping here
	if e.cat.State == CatSleeping && absF(e.cat.X-cx) < 60 {
		alpha := 0.5 + 0.5*math.Sin(e.time*2)
		ctx.Set("fillStyle", fmt.Sprintf("rgba(100,100,180,%.2f)", alpha))
		ctx.Set("font", "bold 14px sans-serif")
		ctx.Set("textAlign", "center")
		ctx.Call("fillText", "zzz", cx+30, fy-50)
	}
}

func (e *Engine) drawLitter(cx, fy float64, filled float64) {
	ctx := e.ctx
	bw := 70.0
	bh := 38.0
	// Box
	ctx.Set("fillStyle", "#e0d8b0")
	roundRect(ctx, cx-bw/2, fy-bh, bw, bh, 4)
	ctx.Call("fill")
	// Litter inside
	ctx.Set("fillStyle", "#c8c0a0")
	roundRect(ctx, cx-bw/2+4, fy-bh+6, bw-8, bh-14, 3)
	ctx.Call("fill")
	// Fill indicator
	if filled > 0 {
		usedW := (bw - 12) * (1 - filled)
		ctx.Set("fillStyle", "rgba(100,80,50,0.5)")
		ctx.Call("fillRect", cx-bw/2+6, fy-bh+8, usedW, bh-18)
	}
	// Rim
	ctx.Set("strokeStyle", "#a89860")
	ctx.Set("lineWidth", 2)
	roundRect(ctx, cx-bw/2, fy-bh, bw, bh, 4)
	ctx.Call("stroke")
	// Clean button when full
	if filled <= 0 {
		e.drawRefillButton(cx, fy-bh-14, "🧹", "Clean")
	}
}

func (e *Engine) drawBrush(cx, fy float64) {
	ctx := e.ctx
	// Short handle — angled, thin
	ctx.Set("strokeStyle", "#8b5a2b")
	ctx.Set("lineWidth", 4)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx+4, fy-4)
	ctx.Call("lineTo", cx-4, fy-32)
	ctx.Call("stroke")
	// Brush head — compact oval body
	hx := cx - 6.0
	hy := fy - 38.0
	ctx.Set("fillStyle", "#c07840")
	ctx.Call("beginPath")
	ctx.Call("ellipse", hx, hy, 12, 7, -0.3, 0, math.Pi*2)
	ctx.Call("fill")
	// Bristles — dense short lines on underside
	ctx.Set("strokeStyle", "#f0e0b0")
	ctx.Set("lineWidth", 1.2)
	ctx.Set("lineCap", "round")
	for i := 0; i < 8; i++ {
		bx2 := hx - 9 + float64(i)*2.6
		by2 := hy + 4 - float64(i%2)*1.5
		ctx.Call("beginPath")
		ctx.Call("moveTo", bx2, by2)
		ctx.Call("lineTo", bx2-0.5, by2+6)
		ctx.Call("stroke")
	}
	// Metal ferrule ring
	ctx.Set("strokeStyle", "#a0a0a0")
	ctx.Set("lineWidth", 2)
	ctx.Call("beginPath")
	ctx.Call("ellipse", hx, hy, 12, 7, -0.3, 0, math.Pi*2)
	ctx.Call("stroke")
}

func (e *Engine) drawCatGrooming(cx, cy float64) {
	ctx := e.ctx

	// Side-profile cat standing on 4 paws, being brushed.
	// Body sways gently as brush strokes along the spine.
	stroke := math.Sin(e.time * 2.8) // -1..1
	bx := cx - 2 + stroke*10         // brush moves back and forth along spine

	// Tail — raised and curved behind the cat (away from viewer)
	tipSway := stroke * 5
	ctx.Set("strokeStyle", colCatPointL)
	ctx.Set("lineWidth", 8)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx-12, cy-16)
	ctx.Call("bezierCurveTo", cx-36, cy-14+tipSway, cx-44, cy-50, cx-22, cy-62)
	ctx.Call("stroke")
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 6)
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx-30, cy-46)
	ctx.Call("bezierCurveTo", cx-42+tipSway, cy-54, cx-26, cy-64, cx-22, cy-62)
	ctx.Call("stroke")

	// Back legs
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-14, cy-10, 8, 13, 0.2, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-13, cy-2, 8, 5, 0.1, 0, math.Pi*2)
	ctx.Call("fill")

	// Body — horizontal standing ellipse
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx, cy-22, 30, 17, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatChest)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+6, cy-20, 18, 10, 0, 0, math.Pi*2)
	ctx.Call("fill")

	// Front legs
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+14, cy-10, 8, 13, -0.15, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+14, cy-2, 8, 5, 0, 0, math.Pi*2)
	ctx.Call("fill")

	// Head — side profile facing right, content half-closed eyes
	hx := cx + 26.0
	hy := cy - 34.0
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, hy, 16, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("arc", hx+2, hy+3, 10, 0, math.Pi)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPointL)
	ctx.Call("beginPath")
	ctx.Call("ellipse", hx+6, hy-1, 5.5, 4, 0.2, 0, math.Pi*2)
	ctx.Call("fill")

	e.drawEar(hx+4, hy-12, 1, true)

	// Half-closed pleasure eye
	ctx.Set("fillStyle", colCatEye)
	ctx.Call("beginPath")
	ctx.Call("arc", hx+6, hy-1, 3.5, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPupil)
	ctx.Call("beginPath")
	ctx.Call("arc", hx+7, hy-1, 1.8, 0, math.Pi*2)
	ctx.Call("fill")
	// Eyelid squint line
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 2)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", hx+2, hy-2)
	ctx.Call("quadraticCurveTo", hx+6, hy-5, hx+10, hy-2)
	ctx.Call("stroke")

	// Whiskers pointing forward
	ctx.Set("strokeStyle", "rgba(255,255,255,0.75)")
	ctx.Set("lineWidth", 1)
	for _, wy := range []float64{hy + 2, hy + 5, hy + 8} {
		ctx.Call("beginPath")
		ctx.Call("moveTo", hx+8, wy)
		ctx.Call("lineTo", hx+26, wy-1)
		ctx.Call("stroke")
	}

	// Brush — moving along the cat's back, bristles pointing down toward fur
	ctx.Call("save")
	ctx.Call("translate", bx, cy-46)
	ctx.Call("rotate", -0.3+stroke*0.15)
	// Handle goes UP from brush head (held by hand above)
	ctx.Set("strokeStyle", "#8b5a2b")
	ctx.Set("lineWidth", 4)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", 0, 0)
	ctx.Call("lineTo", 0, -26)
	ctx.Call("stroke")
	// Brush head at origin
	ctx.Set("fillStyle", "#c07840")
	ctx.Call("beginPath")
	ctx.Call("ellipse", 0, 0, 11, 6, 0, 0, math.Pi*2)
	ctx.Call("fill")
	// Bristles pointing DOWN into the cat's fur
	ctx.Set("strokeStyle", "#f0e0b0")
	ctx.Set("lineWidth", 1.2)
	for j := 0; j < 7; j++ {
		bx2 := -8 + float64(j)*2.7
		ctx.Call("beginPath")
		ctx.Call("moveTo", bx2, 4)
		ctx.Call("lineTo", bx2, 10)
		ctx.Call("stroke")
	}
	ctx.Set("strokeStyle", "#a0a0a0")
	ctx.Set("lineWidth", 2)
	ctx.Call("beginPath")
	ctx.Call("ellipse", 0, 0, 11, 6, 0, 0, math.Pi*2)
	ctx.Call("stroke")
	ctx.Call("restore")

	// Fur wisps floating off during brushing
	for i := 0; i < 3; i++ {
		phase := math.Mod(e.time*1.5+float64(i)*0.9, 2.5) / 2.5
		wx := bx - phase*12 + float64(i)*6
		wy := cy - 44 - phase*22
		alpha := 1.0 - phase
		ctx.Set("strokeStyle", fmt.Sprintf("rgba(245,238,221,%.2f)", alpha*0.8))
		ctx.Set("lineWidth", 1.2)
		ctx.Set("lineCap", "round")
		ctx.Call("beginPath")
		ctx.Call("moveTo", wx, wy)
		ctx.Call("quadraticCurveTo", wx-3, wy-4, wx-1, wy-9)
		ctx.Call("stroke")
	}
}

// drawRefillButton draws a pulsing icon+label button above an empty object.
func (e *Engine) drawRefillButton(cx, top float64, icon, label string) {
	ctx := e.ctx
	pulse := 0.75 + 0.25*math.Sin(e.time*4.0) // fast pulse to attract attention

	bw := 62.0
	bh := 44.0
	bx := cx - bw/2
	by := top - bh

	// Glow halo
	ctx.Set("shadowBlur", 10+pulse*8)
	ctx.Set("shadowColor", "rgba(255,180,40,0.7)")

	// Button background
	ctx.Set("globalAlpha", 0.85+pulse*0.15)
	ctx.Set("fillStyle", "#4a2a08")
	roundRect(ctx, bx, by, bw, bh, 10)
	ctx.Call("fill")

	// Border
	ctx.Set("strokeStyle", fmt.Sprintf("rgba(255,200,60,%.2f)", 0.7+pulse*0.3))
	ctx.Set("lineWidth", 2)
	roundRect(ctx, bx, by, bw, bh, 10)
	ctx.Call("stroke")

	// Icon
	ctx.Set("font", "18px sans-serif")
	ctx.Set("textAlign", "center")
	ctx.Set("fillStyle", "#ffffff")
	ctx.Call("fillText", icon, cx, by+22)

	// Label
	ctx.Set("font", "bold 10px 'Segoe UI', Arial, sans-serif")
	ctx.Set("fillStyle", "#ffd060")
	ctx.Call("fillText", label, cx, by+38)

	// Small downward arrow below button
	ctx.Set("fillStyle", fmt.Sprintf("rgba(255,200,60,%.2f)", 0.6+pulse*0.4))
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx-6, top-4)
	ctx.Call("lineTo", cx+6, top-4)
	ctx.Call("lineTo", cx, top+2)
	ctx.Call("closePath")
	ctx.Call("fill")

	ctx.Set("shadowBlur", 0)
	ctx.Set("shadowColor", "transparent")
	ctx.Set("globalAlpha", 1)
}

func (e *Engine) drawTooltip(label string, cx, y float64) {
	ctx := e.ctx
	ctx.Set("font", "500 13px 'Segoe UI', Arial, sans-serif")
	ctx.Set("textAlign", "center")
	metrics := ctx.Call("measureText", label)
	tw := metrics.Get("width").Float()
	pw, ph := tw+16, 22.0
	// Background
	ctx.Set("fillStyle", "rgba(40,30,20,0.85)")
	roundRect(ctx, cx-pw/2, y-ph, pw, ph, 6)
	ctx.Call("fill")
	// Text
	ctx.Set("fillStyle", "#fff8e8")
	ctx.Call("fillText", label, cx, y-6)
}

// ── cat drawing ───────────────────────────────────────────────────────────────

func (e *Engine) renderCat() {
	ctx := e.ctx
	cat := &e.cat

	ctx.Call("save")

	// Mirror if facing left
	if cat.Direction < 0 {
		ctx.Call("translate", cat.X*2, 0)
		ctx.Call("scale", -1, 1)
	}

	cx := cat.X
	cy := cat.Y

	switch cat.State {
	case CatWalking:
		e.drawCatWalking(cx, cy)
	case CatSleeping:
		e.drawCatSleeping(cx, cy)
	case CatEating:
		e.drawCatEating(cx, cy)
	case CatDrinking:
		e.drawCatDrinking(cx, cy)
	case CatLitter:
		e.drawCatLitter(cx, cy)
	case CatPlaying:
		e.drawCatPlaying(cx, cy)
	case CatScratch:
		e.drawCatScratching(cx, cy)
	case CatGrooming:
		e.drawCatGrooming(cx, cy)
	default:
		e.drawCatSitting(cx, cy)
	}

	// Ruffled fur overlay when Coat need is low (not during grooming/sleeping)
	if cat.State != CatGrooming && cat.State != CatSleeping {
		coat := e.needs.Coat
		if coat < 0.55 {
			intensity := (0.55 - coat) / 0.55 // 0 at coat=0.55, 1 at coat=0
			e.drawRuffledFur(cx, cy, intensity)
		}
	}

	ctx.Call("restore")

	// Hearts FX
	for _, h := range e.hearts {
		alpha := h.T / 1.5
		ctx.Set("fillStyle", fmt.Sprintf("rgba(255,80,120,%.2f)", alpha))
		ctx.Set("font", "16px sans-serif")
		ctx.Set("textAlign", "center")
		ctx.Call("fillText", "♥", h.X, h.Y)
	}

	// Thought bubble (need below 0.30)
	lowest := e.lowestNeed()
	if e.needs.get(lowest) < 0.30 && e.state == StatePlaying {
		e.drawThoughtBubble(cat.X, cat.Y, needIcons[lowest])
	}
}

func (e *Engine) drawCatSitting(cx, cy float64) {
	ctx := e.ctx

	// Tail — dark seal tip
	ctx.Set("lineWidth", 9)
	ctx.Set("lineCap", "round")
	ctx.Set("strokeStyle", colCatPointL)
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx+14, cy-8)
	ctx.Call("bezierCurveTo", cx+42, cy-8, cx+44, cy-52, cx+22, cy-58)
	ctx.Call("stroke")
	// Tail tip (darker)
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 7)
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx+34, cy-46)
	ctx.Call("bezierCurveTo", cx+42, cy-52, cx+30, cy-60, cx+22, cy-58)
	ctx.Call("stroke")

	// Body — fluffy cream oval, slightly wider
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx, cy-30, 24, 30, 0, 0, math.Pi*2)
	ctx.Call("fill")
	// Chest lighter patch
	ctx.Set("fillStyle", colCatChest)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx, cy-26, 13, 18, 0, 0, math.Pi*2)
	ctx.Call("fill")

	// Head
	e.drawCatHead(cx, cy-63, false)

	// Paws — darker seal tips
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-13, cy-5, 11, 7, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+13, cy-5, 11, 7, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-13, cy-3, 8, 5, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+13, cy-3, 8, 5, 0, 0, math.Pi*2)
	ctx.Call("fill")
}

func (e *Engine) drawCatWalking(cx, cy float64) {
	ctx := e.ctx
	ctx.Call("save")
	ctx.Call("translate", cx, cy)
	ctx.Call("scale", 1.14, 1.14)
	ctx.Call("translate", -cx, -cy)
	defer ctx.Call("restore")

	cat := &e.cat
	phase := (float64(cat.AnimFrame) + cat.AnimTime/0.25) * (math.Pi / 2)
	backLead := math.Sin(phase)
	frontLead := math.Sin(phase + math.Pi*0.52)
	spineWave := math.Sin(phase*2 + 0.35)
	bodyDip := 1.2 + 0.8*math.Cos(phase*2)
	headGlide := math.Sin(phase+0.25) * 0.9
	shoulderRoll := math.Sin(phase+math.Pi*0.52) * 1.4
	hipRoll := math.Sin(phase) * 1.1
	tailSway := math.Sin(phase*0.55-0.3) * 4.5

	// Tail stays fairly level and follows the spine with a delayed sway.
	ctx.Set("strokeStyle", colCatPointL)
	ctx.Set("lineWidth", 7)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx-24, cy-30-bodyDip*0.2)
	ctx.Call("bezierCurveTo", cx-42, cy-34+tailSway*0.4, cx-50, cy-44+tailSway, cx-34, cy-51+tailSway*0.3)
	ctx.Call("stroke")
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 5)
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx-42, cy-44+tailSway*0.2)
	ctx.Call("bezierCurveTo", cx-48, cy-50+tailSway*0.5, cx-38, cy-55, cx-34, cy-51+tailSway*0.3)
	ctx.Call("stroke")

	// Rear legs: compact, stepping under the pelvis.
	e.drawWalkingLeg(cx-14, cy-24+hipRoll, cy-3, backLead, false, true)
	e.drawWalkingLeg(cx-4, cy-23-hipRoll*0.5, cy-3.5, -backLead, false, false)

	// Long flexible torso with a subtle feline spine wave.
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+4, cy-29-bodyDip, 29, 14.5, -0.05+spineWave*0.02, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatChest)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+14, cy-27-bodyDip*0.85, 16, 8.8, -0.12, 0, math.Pi*2)
	ctx.Call("fill")

	// Front legs glide rather than stomp; one reaches while the other bears weight.
	e.drawWalkingLeg(cx+13, cy-22+shoulderRoll, cy-2.8, frontLead, true, true)
	e.drawWalkingLeg(cx+24, cy-21-shoulderRoll*0.45, cy-2.2, -frontLead, true, false)

	// Neck and head float forward with minimal bob, typical of a cat walk.
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+25, cy-38-bodyDip*0.7+headGlide*0.3, 9, 12, -0.22, 0, math.Pi*2)
	ctx.Call("fill")

	hx := cx + 37.0
	hy := cy - 47.0 - bodyDip*0.55 + headGlide
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, hy, 15.5, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("arc", hx+1.5, hy+3, 10.5, 0, math.Pi)
	ctx.Call("fill")

	e.drawEar(hx-7, hy-12, -1, false)
	e.drawEar(hx+7, hy-12, 1, false)

	// Focused forward gaze, but with the same eye scale as the idle pose.
	for _, ex := range []float64{hx - 4.5, hx + 5} {
		ctx.Set("fillStyle", colCatEye)
		ctx.Call("beginPath")
		ctx.Call("arc", ex, hy-1, 5, 0, math.Pi*2)
		ctx.Call("fill")
		ctx.Set("fillStyle", colCatPupil)
		ctx.Call("beginPath")
		ctx.Call("ellipse", ex, hy-1, 1.5, 4, 0, 0, math.Pi*2)
		ctx.Call("fill")
		ctx.Set("fillStyle", "rgba(255,255,255,0.75)")
		ctx.Call("beginPath")
		ctx.Call("arc", ex+2, hy-3, 1.8, 0, math.Pi*2)
		ctx.Call("fill")
	}

	// Nose and whiskers.
	ctx.Set("fillStyle", "#d98fa0")
	ctx.Call("beginPath")
	ctx.Call("arc", hx+11, hy+5, 2.1, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("strokeStyle", "rgba(255,255,255,0.75)")
	ctx.Set("lineWidth", 1)
	for _, wy := range []float64{hy + 3, hy + 6, hy + 9} {
		ctx.Call("beginPath")
		ctx.Call("moveTo", hx+9, wy)
		ctx.Call("lineTo", hx+25, wy-1)
		ctx.Call("stroke")
	}
}

func (e *Engine) drawWalkingLeg(hipX, hipY, groundY, stride float64, front, near bool) {
	ctx := e.ctx

	reach := 5.5
	legWidth := 8.2
	footW := 8.8
	footH := 5.8
	padW := 6.8
	padH := 4.1
	if front {
		reach = 7.0
		legWidth = 7.1
		footW = 7.6
		footH = 5.0
		padW = 5.8
		padH = 3.5
	}
	footX := hipX + stride*reach
	kneeX := hipX + stride*2.6
	if front {
		kneeX = hipX + stride*3.3
	}
	kneeY := hipY + 10 + math.Abs(stride)*3
	footLift := math.Max(0, -stride) * 3.2
	footY := groundY - footLift

	ctx.Set("strokeStyle", colCatBody)
	ctx.Set("lineWidth", legWidth)
	if !near {
		ctx.Set("globalAlpha", 0.82)
	}
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", hipX, hipY)
	ctx.Call("quadraticCurveTo", kneeX, kneeY, footX, footY)
	ctx.Call("stroke")

	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", footX, footY, footW, footH, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("ellipse", footX+0.5, footY+1, padW, padH, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("globalAlpha", 1)
}

func (e *Engine) drawCatSleeping(cx, cy float64) {
	ctx := e.ctx

	// Ragdoll hallmark: sleeping on back, all four paws dangling in the air
	breath := math.Sin(e.time * math.Pi * 0.4) // slow ~0.2 Hz breathing
	belly := breath * 2.2                      // belly expands/contracts

	// Tail hanging off right side, tip sways with breath
	tipSway := breath * 5
	ctx.Set("strokeStyle", colCatPointL)
	ctx.Set("lineWidth", 7)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx+34, cy-10)
	ctx.Call("bezierCurveTo", cx+52, cy-6+tipSway, cx+60, cy-28, cx+46, cy-38)
	ctx.Call("stroke")
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 5)
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx+48, cy-28)
	ctx.Call("bezierCurveTo", cx+58+tipSway, cy-34, cx+48, cy-41, cx+46, cy-38)
	ctx.Call("stroke")

	// Body — wide flat oval (lying on back), belly expands with breath
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx, cy-14, 38, 16+belly, 0, 0, math.Pi*2)
	ctx.Call("fill")
	// Belly/chest lighter (visible since on back)
	ctx.Set("fillStyle", colCatChest)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+4, cy-15, 24, 10+belly*0.5, 0, 0, math.Pi*2)
	ctx.Call("fill")

	// Four paws pointing upward — limp, relaxed ragdoll flop
	type pawPos struct{ x, y float64 }
	paws := []pawPos{
		{cx - 22, cy - 34},
		{cx - 8, cy - 36},
		{cx + 8, cy - 35},
		{cx + 22, cy - 32},
	}
	for i, p := range paws {
		dangle := breath * (1.2 - float64(i)*0.15) // each paw sways slightly differently
		px := p.x + dangle
		// Leg
		ctx.Set("strokeStyle", colCatBody)
		ctx.Set("lineWidth", 7)
		ctx.Set("lineCap", "round")
		ctx.Call("beginPath")
		ctx.Call("moveTo", px, cy-14)
		ctx.Call("lineTo", px, p.y)
		ctx.Call("stroke")
		// Paw — cream base, seal tip
		ctx.Set("fillStyle", colCatBody)
		ctx.Call("beginPath")
		ctx.Call("ellipse", px, p.y, 7, 5, 0, 0, math.Pi*2)
		ctx.Call("fill")
		ctx.Set("fillStyle", colCatPoint)
		ctx.Call("beginPath")
		ctx.Call("ellipse", px, p.y-1, 5, 3.5, 0, 0, math.Pi*2)
		ctx.Call("fill")
	}

	// Head at left end, face upward
	hx := cx - 38.0
	hy := cy - 14.0
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, hy, 16, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, hy+2, 11, 0, math.Pi)
	ctx.Call("fill")
	e.drawEar(hx-8, hy-12, -1, false)
	e.drawEar(hx+8, hy-12, 1, false)

	// Closed eyes — relaxed upward curves (content)
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 2)
	ctx.Set("lineCap", "round")
	for _, ex := range []float64{hx - 5, hx + 5} {
		ctx.Call("beginPath")
		ctx.Call("moveTo", ex-4, hy-2)
		ctx.Call("quadraticCurveTo", ex, hy-6, ex+4, hy-2)
		ctx.Call("stroke")
	}

	// ZZz bubbles floating up from head
	for i := 0; i < 3; i++ {
		t := math.Mod(e.time*0.55+float64(i)*1.1, 3.0) / 3.0
		bx2 := hx - 6 + float64(i)*7
		by2 := hy - 22 - t*38
		alpha := 1.0 - t
		size := 9.0 + float64(i)*3 + t*4
		ctx.Set("fillStyle", fmt.Sprintf("rgba(140,140,200,%.2f)", alpha*0.85))
		ctx.Set("font", fmt.Sprintf("bold %.0fpx sans-serif", size))
		ctx.Set("textAlign", "center")
		ctx.Call("fillText", "z", bx2, by2)
	}
}

func (e *Engine) drawCatEating(cx, cy float64) {
	ctx := e.ctx
	// Head bobs toward bowl at ~2 Hz — deeper dip = taking a bite
	bob := math.Sin(e.time * 2.2 * math.Pi) // -1..1
	dip := 10.0 + bob*7.0                   // 3..17 px down

	hx := cx + 14
	hy := cy - 34 + dip

	// Tail sways gently
	ctx.Set("strokeStyle", colCatPointL)
	ctx.Set("lineWidth", 8)
	ctx.Set("lineCap", "round")
	sway := math.Sin(e.time*1.8) * 10
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx-14, cy-8)
	ctx.Call("bezierCurveTo", cx-40, cy-10+sway, cx-44, cy-46+sway, cx-22, cy-52)
	ctx.Call("stroke")

	// Body leaned forward
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx, cy-22, 26, 20, -0.3, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatChest)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+8, cy-20, 14, 12, -0.3, 0, math.Pi*2)
	ctx.Call("fill")

	// Neck
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+12, cy-33, 9, 12, -0.15, 0, math.Pi*2)
	ctx.Call("fill")

	// Head angled down toward bowl
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, hy, 17, 0, math.Pi*2)
	ctx.Call("fill")
	// Seal mask
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, hy+3, 12, 0, math.Pi)
	ctx.Call("fill")

	// Ears pressed back when eating
	e.drawEar(hx-9, hy-12, -1, false)
	e.drawEar(hx+9, hy-12, 1, false)

	// Eyes — half-closed with pleasure
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 2.5)
	ctx.Set("lineCap", "round")
	for _, ex := range []float64{hx - 6, hx + 6} {
		ctx.Call("beginPath")
		ctx.Call("moveTo", ex-4, hy-2)
		ctx.Call("quadraticCurveTo", ex, hy-5, ex+4, hy-2)
		ctx.Call("stroke")
	}

	// Tongue lapping when head is lowest (bob > 0.4)
	if bob > 0.4 {
		tongueLen := (bob - 0.4) * 14
		ctx.Set("fillStyle", "#e06080")
		ctx.Call("beginPath")
		ctx.Call("ellipse", hx, hy+12, 5, tongueLen, 0, 0, math.Pi*2)
		ctx.Call("fill")
	}

	// Paws flat on floor
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-2, cy-4, 10, 7, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+18, cy-4, 10, 7, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-2, cy-2, 7, 5, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+18, cy-2, 7, 5, 0, 0, math.Pi*2)
	ctx.Call("fill")
}

func (e *Engine) drawCatDrinking(cx, cy float64) {
	ctx := e.ctx
	// Drinking: slower, deeper head dip; tongue extended and held longer
	bob := math.Sin(e.time * 1.4 * math.Pi)
	dip := 8.0 + bob*9.0

	hx := cx + 14
	hy := cy - 32 + dip

	// Tail
	ctx.Set("strokeStyle", colCatPointL)
	ctx.Set("lineWidth", 8)
	ctx.Set("lineCap", "round")
	sway := math.Sin(e.time*1.2) * 8
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx-14, cy-8)
	ctx.Call("bezierCurveTo", cx-42, cy-8+sway, cx-42, cy-44+sway, cx-20, cy-50)
	ctx.Call("stroke")

	// Body
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx, cy-21, 26, 20, -0.3, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatChest)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+8, cy-19, 14, 12, -0.3, 0, math.Pi*2)
	ctx.Call("fill")

	// Neck
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+12, cy-32, 9, 12, -0.15, 0, math.Pi*2)
	ctx.Call("fill")

	// Head
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, hy, 17, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, hy+3, 12, 0, math.Pi)
	ctx.Call("fill")

	e.drawEar(hx-9, hy-12, -1, false)
	e.drawEar(hx+9, hy-12, 1, false)

	// Eyes nearly closed (concentrating on water surface)
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 2)
	ctx.Set("lineCap", "round")
	for _, ex := range []float64{hx - 6, hx + 6} {
		ctx.Call("beginPath")
		ctx.Call("moveTo", ex-4, hy-2)
		ctx.Call("lineTo", ex+4, hy-2)
		ctx.Call("stroke")
	}

	// Tongue — always visible when drinking, curls up at tip
	tongueLen := 8.0 + bob*6
	ctx.Set("fillStyle", "#e06080")
	ctx.Call("beginPath")
	ctx.Call("ellipse", hx, hy+12, 4.5, tongueLen, 0, 0, math.Pi*2)
	ctx.Call("fill")
	// Tongue curl at bottom
	ctx.Set("fillStyle", "#c84060")
	ctx.Call("beginPath")
	ctx.Call("ellipse", hx-1, hy+12+tongueLen-3, 5, 4, 0.3, 0, math.Pi*2)
	ctx.Call("fill")

	// Water ripple when tongue touches bowl (bob peak)
	if bob > 0.5 {
		rippleAlpha := (bob - 0.5) * 1.4
		ctx.Set("strokeStyle", fmt.Sprintf("rgba(100,180,255,%.2f)", rippleAlpha))
		ctx.Set("lineWidth", 1.5)
		ctx.Call("beginPath")
		ctx.Call("ellipse", hx, cy-3, 12+rippleAlpha*8, 4, 0, 0, math.Pi*2)
		ctx.Call("stroke")
	}

	// Paws
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-2, cy-4, 10, 7, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+18, cy-4, 10, 7, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-2, cy-2, 7, 5, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+18, cy-2, 7, 5, 0, 0, math.Pi*2)
	ctx.Call("fill")
}

func (e *Engine) drawCatCrouching(cx, cy float64) {
	ctx := e.ctx
	hx := cx + 10 // head shifted forward

	// Body leaned forward
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx, cy-20, 26, 19, -0.25, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatChest)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+6, cy-18, 14, 11, -0.25, 0, math.Pi*2)
	ctx.Call("fill")

	// Head — seal mask on lower half
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, cy-38, 18, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, cy-35, 12, 0, math.Pi)
	ctx.Call("fill")

	// Ears on crouching head
	e.drawEar(hx-10, cy-52, -1, false)
	e.drawEar(hx+10, cy-52, 1, false)

	// Blue eyes, down-cast (small — concentrating)
	ctx.Set("fillStyle", colCatEye)
	ctx.Call("beginPath")
	ctx.Call("arc", hx-6, cy-40, 3.5, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("arc", hx+6, cy-40, 3.5, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPupil)
	ctx.Call("beginPath")
	ctx.Call("arc", hx-6, cy-40, 2, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("arc", hx+6, cy-40, 2, 0, math.Pi*2)
	ctx.Call("fill")

	// Front paws — seal tips
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-4, cy-4, 10, 7, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+16, cy-4, 10, 7, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-4, cy-2, 7, 5, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+16, cy-2, 7, 5, 0, 0, math.Pi*2)
	ctx.Call("fill")
}

// drawEar draws one ragdoll-style ear: baseX/baseY is where the ear meets the head,
// side is -1 (left) or +1 (right). If tuft is true, small inner fur lines are drawn.
func (e *Engine) drawEar(baseX, baseY float64, side float64, tuft bool) {
	ctx := e.ctx
	// Outer ear — compact and seated lower so the base stays fully merged into the skull.
	tipX := baseX + side*2.6
	tipY := baseY - 17
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("moveTo", baseX-side*5.2, baseY+1.4)
	ctx.Call("quadraticCurveTo", baseX, baseY-2.5, baseX+side*5.2, baseY+1.4)
	ctx.Call("lineTo", tipX, tipY)
	ctx.Call("closePath")
	ctx.Call("fill")

	// Inner ear — inset and shorter so it stays fully inside the outer ear.
	iBaseX := baseX - side*1.1
	ctx.Set("fillStyle", colCatInner)
	ctx.Call("beginPath")
	ctx.Call("moveTo", iBaseX-side*2.8, baseY+0.4)
	ctx.Call("quadraticCurveTo", iBaseX, baseY-1.5, iBaseX+side*2.8, baseY+0.4)
	ctx.Call("lineTo", tipX+side*0.4, tipY+6.4)
	ctx.Call("closePath")
	ctx.Call("fill")

	// Fur tuft lines at tip
	if tuft {
		ctx.Set("strokeStyle", colCatBody)
		ctx.Set("lineWidth", 1.2)
		ctx.Set("lineCap", "round")
		for i := -1; i <= 1; i++ {
			ctx.Call("beginPath")
			ctx.Call("moveTo", tipX+float64(i)*2.2, tipY+4)
			ctx.Call("lineTo", tipX+float64(i)*3.5, tipY-3.5)
			ctx.Call("stroke")
		}
	}
}

func (e *Engine) drawCatPlaying(cx, cy float64) {
	ctx := e.ctx
	ctx.Call("save")
	ctx.Call("translate", cx, cy)
	ctx.Call("scale", 1.14, 1.14)
	ctx.Call("translate", -cx, -cy)
	defer ctx.Call("restore")

	_, _, featherX, featherY := e.toyFeatherPose(toyX)
	pawCycle := math.Sin(e.time * 5.4)
	leftLift := math.Max(0, pawCycle)
	rightLift := math.Max(0, -pawCycle)
	bodyBob := math.Abs(pawCycle) * 1.2
	headTrack := (featherX - toyX) * 0.12
	tailFlick := math.Sin(e.time*5.0) * 4
	baseX := cx - 34

	// Tail behind the cat, slightly animated with excitement.
	ctx.Set("strokeStyle", colCatPointL)
	ctx.Set("lineWidth", 8)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", baseX+6, cy-10)
	ctx.Call("bezierCurveTo", baseX-18, cy-8+tailFlick*0.2, baseX-28, cy-46+tailFlick, baseX-10, cy-58)
	ctx.Call("stroke")
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 6)
	ctx.Call("beginPath")
	ctx.Call("moveTo", baseX-14, cy-43+tailFlick*0.15)
	ctx.Call("bezierCurveTo", baseX-24, cy-49+tailFlick*0.2, baseX-14, cy-60, baseX-10, cy-58)
	ctx.Call("stroke")

	// Seated hindquarters, close to the default sitting pose.
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", baseX-7, cy-16+bodyBob*0.2, 10, 14, 0.15, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("ellipse", baseX-6, cy-4+bodyBob*0.1, 9, 5.2, 0.1, 0, math.Pi*2)
	ctx.Call("fill")

	// Upright seated body, only slightly leaning toward the toy.
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", baseX-1, cy-28+bodyBob, 22, 28, -0.08, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatChest)
	ctx.Call("beginPath")
	ctx.Call("ellipse", baseX+3, cy-25+bodyBob*0.9, 12, 17, -0.06, 0, math.Pi*2)
	ctx.Call("fill")

	// Head turned toward the toy in side view.
	hx := baseX + 15.0 + headTrack
	hy := cy - 52.0 + bodyBob*0.5
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, hy, 17, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("arc", hx+2, hy+3, 11, 0, math.Pi)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPointL)
	ctx.Call("beginPath")
	ctx.Call("ellipse", hx+7, hy-1, 6, 4.5, 0.2, 0, math.Pi*2)
	ctx.Call("fill")
	e.drawEar(hx+5, hy-13, 1, true)

	// One visible eye in side view.
	eyeX := hx + 8
	eyeY := hy - 2
	ctx.Set("fillStyle", colCatEye)
	ctx.Call("beginPath")
	ctx.Call("arc", eyeX, eyeY, 5.2, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPupil)
	ctx.Call("beginPath")
	ctx.Call("ellipse", eyeX+0.5, eyeY, 2.0, 4.2, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", "rgba(255,255,255,0.75)")
	ctx.Call("beginPath")
	ctx.Call("arc", eyeX+2, eyeY-2, 1.5, 0, math.Pi*2)
	ctx.Call("fill")

	// Nose
	ctx.Set("fillStyle", "#d98fa0")
	ctx.Call("beginPath")
	ctx.Call("arc", hx+11, hy+5, 2.1, 0, math.Pi*2)
	ctx.Call("fill")

	// Whiskers — angled forward in hunt/play focus.
	ctx.Set("strokeStyle", "rgba(255,255,255,0.75)")
	ctx.Set("lineWidth", 1)
	ctx.Set("lineCap", "round")
	for _, wy := range []float64{hy + 2, hy + 5, hy + 8} {
		ctx.Call("beginPath")
		ctx.Call("moveTo", hx+9, wy)
		ctx.Call("lineTo", hx+27, wy-2)
		ctx.Call("stroke")
	}

	// Two front paws alternately reach toward the toy.
	shoulderX := baseX + 6
	shoulderY := cy - 36 + bodyBob*0.5
	type pawSpec struct {
		lift  float64
		phase float64
	}
	for _, paw := range []pawSpec{
		{lift: leftLift, phase: -1.5},
		{lift: rightLift, phase: 1.5},
	} {
		pawX := featherX - 16 + paw.lift*10
		pawY := featherY + 10 - paw.lift*12 + paw.phase
		ctx.Set("strokeStyle", colCatBody)
		ctx.Set("lineWidth", 8)
		ctx.Set("lineCap", "round")
		ctx.Call("beginPath")
		ctx.Call("moveTo", shoulderX, shoulderY)
		ctx.Call("quadraticCurveTo", baseX+18+paw.lift*7, cy-30-paw.lift*8, pawX, pawY)
		ctx.Call("stroke")
		ctx.Set("fillStyle", colCatBody)
		ctx.Call("beginPath")
		ctx.Call("ellipse", pawX, pawY, 8.5, 5.3, -0.25, 0, math.Pi*2)
		ctx.Call("fill")
		ctx.Set("fillStyle", colCatPoint)
		ctx.Call("beginPath")
		ctx.Call("ellipse", pawX+1.0, pawY+0.7, 6.0, 3.6, -0.25, 0, math.Pi*2)
		ctx.Call("fill")
	}

	_ = featherY
}

func (e *Engine) drawCatScratching(cx, cy float64) {
	ctx := e.ctx

	// Side-profile view: cat stands upright facing the post (to the right in
	// local coords; renderCat mirrors if Direction==-1 so it faces left in world).
	// Two front paws alternate up/down on the post surface.
	stroke := math.Sin(e.time * 4.5) // -1..1
	pawHiY := cy - 62 + stroke*9     // upper paw
	pawLoY := cy - 44 - stroke*9     // lower paw
	pawX := cx + 28.0                // horizontal reach toward post
	shoulderX := cx + 6.0
	shoulderY := cy - 52.0

	// Tail — raised and curved behind (away from post)
	tailSway := stroke * 4
	ctx.Set("strokeStyle", colCatPointL)
	ctx.Set("lineWidth", 8)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx-10, cy-20)
	ctx.Call("bezierCurveTo", cx-34, cy-14+tailSway, cx-42, cy-52, cx-22, cy-62)
	ctx.Call("stroke")
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 6)
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx-30, cy-48)
	ctx.Call("bezierCurveTo", cx-40+tailSway, cy-56, cx-24, cy-64, cx-22, cy-62)
	ctx.Call("stroke")

	// Body — upright, slight forward lean; chest visible from side
	rock := stroke * 1.5
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-4, cy-32, 15, 32, 0.14+rock*0.02, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatChest)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+1, cy-30, 9, 20, 0.1, 0, math.Pi*2)
	ctx.Call("fill")

	// Back leg (single profile leg)
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-8, cy-10, 9, 14, 0.25, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-7, cy-2, 9, 5, 0.1, 0, math.Pi*2)
	ctx.Call("fill")

	// Arms from shoulder to post
	ctx.Set("strokeStyle", colCatBody)
	ctx.Set("lineWidth", 9)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", shoulderX, shoulderY-4)
	ctx.Call("lineTo", pawX, pawHiY)
	ctx.Call("stroke")
	ctx.Call("beginPath")
	ctx.Call("moveTo", shoulderX, shoulderY+5)
	ctx.Call("lineTo", pawX, pawLoY)
	ctx.Call("stroke")

	// Vertical scratch marks on post surface
	ctx.Set("strokeStyle", "rgba(100,60,20,0.40)")
	ctx.Set("lineWidth", 1.5)
	for i := 0; i < 5; i++ {
		mx := pawX + 4 + float64(i)*3
		my := cy - 36 - float64(i)*7 + stroke*3
		ctx.Call("beginPath")
		ctx.Call("moveTo", mx, my)
		ctx.Call("lineTo", mx+1, my+11)
		ctx.Call("stroke")
	}

	// Paw tips
	for _, p := range [][2]float64{{pawX, pawHiY}, {pawX, pawLoY}} {
		ctx.Set("fillStyle", colCatBody)
		ctx.Call("beginPath")
		ctx.Call("ellipse", p[0], p[1], 8, 5, -0.3, 0, math.Pi*2)
		ctx.Call("fill")
		ctx.Set("fillStyle", colCatPoint)
		ctx.Call("beginPath")
		ctx.Call("ellipse", p[0]+2, p[1], 5, 3.5, -0.3, 0, math.Pi*2)
		ctx.Call("fill")
	}

	// Head — side profile, facing the post (right), focused gaze
	hx := cx + 6.0
	hy := cy - 70.0
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, hy, 17, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("arc", hx+3, hy+3, 11, 0, math.Pi)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPointL)
	ctx.Call("beginPath")
	ctx.Call("ellipse", hx+7, hy-1, 6, 4.5, 0.2, 0, math.Pi*2)
	ctx.Call("fill")

	// One ear visible (top/far side)
	e.drawEar(hx+5, hy-13, 1, true)

	// One eye — focused, slightly narrowed
	ctx.Set("fillStyle", colCatEye)
	ctx.Call("beginPath")
	ctx.Call("arc", hx+8, hy-2, 4, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPupil)
	ctx.Call("beginPath")
	ctx.Call("arc", hx+9, hy-2, 2.2, 0, math.Pi*2)
	ctx.Call("fill")

	// Whiskers pointing forward toward post
	ctx.Set("strokeStyle", "rgba(255,255,255,0.75)")
	ctx.Set("lineWidth", 1)
	ctx.Set("lineCap", "round")
	for _, wy := range []float64{hy + 2, hy + 5, hy + 8} {
		ctx.Call("beginPath")
		ctx.Call("moveTo", hx+10, wy)
		ctx.Call("lineTo", hx+28, wy-1)
		ctx.Call("stroke")
	}
}

func (e *Engine) drawCatLitter(cx, cy float64) {
	ctx := e.ctx

	// Squatting posture — body lower and more compressed
	// Digging paw alternates: left paw scoops forward/back
	dig := math.Sin(e.time * 3.8) // -1..1, digging rhythm

	// Tail raised up and slightly curved
	ctx.Set("strokeStyle", colCatPointL)
	ctx.Set("lineWidth", 8)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx-12, cy-18)
	ctx.Call("bezierCurveTo", cx-30, cy-30, cx-26, cy-70, cx-10, cy-75)
	ctx.Call("stroke")
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 6)
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx-22, cy-55)
	ctx.Call("bezierCurveTo", cx-28, cy-68, cx-18, cy-77, cx-10, cy-75)
	ctx.Call("stroke")

	// Body — lower and crouched (wider, flatter ellipse)
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx, cy-14, 28, 16, 0.15, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatChest)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-4, cy-14, 15, 9, 0.15, 0, math.Pi*2)
	ctx.Call("fill")

	// Head turned away (to the side) — offset to back, rotated
	hx := cx - 14.0
	hy := cy - 34.0
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, hy, 16, 0, math.Pi*2)
	ctx.Call("fill")
	// Seal mask — lower half
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("arc", hx, hy+2, 11, 0, math.Pi)
	ctx.Call("fill")

	// Ears — both visible when head turned
	e.drawEar(hx-8, hy-12, -1, false)
	e.drawEar(hx+8, hy-12, 1, false)

	// Eyes — squinting/averted (looking away from us)
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 2)
	ctx.Set("lineCap", "round")
	for _, ex := range []float64{hx - 5, hx + 5} {
		ctx.Call("beginPath")
		ctx.Call("moveTo", ex-3, hy-2)
		ctx.Call("lineTo", ex+3, hy-2)
		ctx.Call("stroke")
	}

	// Digging front paw — left paw scoops forward and back
	digPawX := cx + 14 + dig*12
	digPawY := cy - 6 + math.Abs(dig)*4 // lifts slightly at extremes
	ctx.Set("strokeStyle", colCatBody)
	ctx.Set("lineWidth", 9)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx+2, cy-18)
	ctx.Call("lineTo", digPawX, digPawY)
	ctx.Call("stroke")
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", digPawX, digPawY, 10, 7, dig*0.3, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("ellipse", digPawX+dig*2, digPawY, 7, 5, dig*0.3, 0, math.Pi*2)
	ctx.Call("fill")

	// Stationary rear paw
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-10, cy-4, 11, 7, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-10, cy-3, 8, 5, 0, 0, math.Pi*2)
	ctx.Call("fill")

	// Small sand/litter dust puffs when digging (at paw position)
	if math.Abs(dig) > 0.6 {
		dustAlpha := (math.Abs(dig) - 0.6) * 1.8
		ctx.Set("fillStyle", fmt.Sprintf("rgba(210,190,150,%.2f)", dustAlpha*0.6))
		for i := 0; i < 3; i++ {
			ox := digPawX + float64(i-1)*8 + dig*4
			oy := digPawY + float64(i)*3
			ctx.Call("beginPath")
			ctx.Call("arc", ox, oy, 4+float64(i), 0, math.Pi*2)
			ctx.Call("fill")
		}
	}
}

func (e *Engine) drawCatHead(cx, cy float64, wideEyes bool) {
	ctx := e.ctx

	// Ears first (behind head)
	e.drawEar(cx-13, cy-14, -1, true)
	e.drawEar(cx+13, cy-14, 1, true)

	// Head — cream base, slightly flattened top (ragdoll round face)
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("arc", cx, cy, 19, 0, math.Pi*2)
	ctx.Call("fill")

	// Seal face mask: darker around eyes and lower face
	// Lower mask — inverted U shape covering cheeks and chin
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("arc", cx, cy+2, 13, 0, math.Pi)
	ctx.Call("fill")
	// Eye mask patches (raccoon-like in seal points)
	ctx.Set("fillStyle", colCatPointL)
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx-7, cy-1, 7, 5, -0.2, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("ellipse", cx+7, cy-1, 7, 5, 0.2, 0, math.Pi*2)
	ctx.Call("fill")

	// Eyes — bright ragdoll blue
	eyeR := 5.0
	if wideEyes {
		eyeR = 6.5
	}
	blink := math.Sin(e.time*0.6+0.5) > 0.96
	eyeY := cy - 2.0
	if blink {
		// Closed — curved line
		ctx.Set("strokeStyle", colCatPoint)
		ctx.Set("lineWidth", 2.5)
		ctx.Set("lineCap", "round")
		for _, ex := range []float64{cx - 7, cx + 7} {
			ctx.Call("beginPath")
			ctx.Call("moveTo", ex-eyeR+1, eyeY)
			ctx.Call("quadraticCurveTo", ex, eyeY-3, ex+eyeR-1, eyeY)
			ctx.Call("stroke")
		}
	} else {
		for _, ex := range []float64{cx - 7, cx + 7} {
			// Iris — blue
			ctx.Set("fillStyle", colCatEye)
			ctx.Call("beginPath")
			ctx.Call("arc", ex, eyeY, eyeR, 0, math.Pi*2)
			ctx.Call("fill")
			// Pupil — vertical slit
			ctx.Set("fillStyle", colCatPupil)
			ctx.Call("beginPath")
			ctx.Call("ellipse", ex, eyeY, eyeR*0.30, eyeR*0.80, 0, 0, math.Pi*2)
			ctx.Call("fill")
			// Cornea highlight
			ctx.Set("fillStyle", "rgba(255,255,255,0.75)")
			ctx.Call("beginPath")
			ctx.Call("arc", ex+2, eyeY-2, 1.8, 0, math.Pi*2)
			ctx.Call("fill")
		}
	}

	// Nose — small pink triangle
	ctx.Set("fillStyle", colCatNose)
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx, cy+5)
	ctx.Call("lineTo", cx-3.5, cy+1)
	ctx.Call("lineTo", cx+3.5, cy+1)
	ctx.Call("closePath")
	ctx.Call("fill")

	// Mouth — gentle M shape
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 1.5)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx-4, cy+7)
	ctx.Call("quadraticCurveTo", cx-2, cy+10, cx, cy+8)
	ctx.Call("stroke")
	ctx.Call("beginPath")
	ctx.Call("moveTo", cx+4, cy+7)
	ctx.Call("quadraticCurveTo", cx+2, cy+10, cx, cy+8)
	ctx.Call("stroke")

	// Whiskers — long, slight curve
	ctx.Set("strokeStyle", "rgba(80,60,40,0.55)")
	ctx.Set("lineWidth", 1)
	ctx.Set("lineCap", "round")
	for _, side := range []float64{-1, 1} {
		for j, angle := range []float64{-0.08, 0.04, 0.18} {
			length := 26.0 - float64(j)*2
			x0 := cx + side*5
			y0 := cy + 3.0
			x1 := x0 + side*length*math.Cos(angle)
			y1 := y0 + length*math.Sin(angle)
			ctx.Call("beginPath")
			ctx.Call("moveTo", x0, y0)
			ctx.Call("lineTo", x1, y1)
			ctx.Call("stroke")
		}
	}
}

func (e *Engine) drawThoughtBubble(cx, cy float64, icon string) {
	ctx := e.ctx
	bx := cx + 35
	by := cy - 80
	bw := 52.0
	bh := 40.0

	// Bubble dots
	ctx.Set("fillStyle", "rgba(255,255,255,0.9)")
	for i, r := range []float64{4, 5, 6} {
		px := cx + 20 + float64(i)*8
		py := cy - 55 - float64(i)*6
		ctx.Call("beginPath")
		ctx.Call("arc", px, py, r, 0, math.Pi*2)
		ctx.Call("fill")
	}

	// Bubble body
	ctx.Set("fillStyle", "rgba(255,255,255,0.95)")
	ctx.Set("strokeStyle", "#ccc")
	ctx.Set("lineWidth", 1.5)
	roundRect(ctx, bx, by, bw, bh, 10)
	ctx.Call("fill")
	roundRect(ctx, bx, by, bw, bh, 10)
	ctx.Call("stroke")

	// Icon
	ctx.Set("font", "22px sans-serif")
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", icon, bx+bw/2, by+bh*0.72)
}

// drawRuffledFur draws spiky tufts around the cat when Coat need is low.
// intensity: 0 = invisible, 1 = fully messy.
func (e *Engine) drawRuffledFur(cx, cy float64, intensity float64) {
	ctx := e.ctx
	alpha := intensity * 0.85
	ctx.Set("strokeStyle", fmt.Sprintf("rgba(160,100,60,%.2f)", alpha))
	ctx.Set("lineWidth", 1.5)
	ctx.Set("lineCap", "round")

	// Tuft positions around the cat body (relative to cx, cy)
	// Each entry: [base_x, base_y, angle_rad, length]
	tufts := [][4]float64{
		// Along back
		{-14, -42, -0.7, 7 + intensity*5},
		{-4, -50, -0.9, 7 + intensity*5},
		{6, -48, -1.1, 8 + intensity*6},
		{14, -44, -1.3, 7 + intensity*5},
		// Sides / belly
		{-20, -30, -2.8, 6 + intensity*4},
		{20, -30, 0.0, 6 + intensity*4},
		{-22, -18, -2.6, 5 + intensity*4},
		{22, -18, 0.2, 5 + intensity*4},
		// Neck / chest
		{10, -56, 0.4, 6 + intensity*4},
		{-8, -52, -1.8, 5 + intensity*3},
		// Tail base
		{-16, -12, 2.4, 6 + intensity*4},
	}

	for _, t := range tufts {
		bx := cx + t[0]
		by := cy + t[1]
		angle := t[2]
		length := t[3]
		// Slight animated wobble
		wobble := math.Sin(e.time*3.5+t[0]*0.3) * 0.2
		a := angle + wobble
		ex := bx + math.Cos(a)*length
		ey := by + math.Sin(a)*length
		ctx.Call("beginPath")
		ctx.Call("moveTo", bx, by)
		ctx.Call("lineTo", ex, ey)
		ctx.Call("stroke")
		// Fork at tip for extra messiness
		if intensity > 0.3 {
			fork := 0.35
			fl := length * 0.45
			ctx.Call("beginPath")
			ctx.Call("moveTo", ex, ey)
			ctx.Call("lineTo", ex+math.Cos(a+fork)*fl, ey+math.Sin(a+fork)*fl)
			ctx.Call("stroke")
			ctx.Call("beginPath")
			ctx.Call("moveTo", ex, ey)
			ctx.Call("lineTo", ex+math.Cos(a-fork)*fl, ey+math.Sin(a-fork)*fl)
			ctx.Call("stroke")
		}
	}
}

// ── need bars ─────────────────────────────────────────────────────────────────

func (e *Engine) renderNeedBars() {
	ctx := e.ctx

	// Panel background
	ctx.Set("fillStyle", "#3a2a18")
	ctx.Call("fillRect", 0, barsY, canvasW, barsH)

	// 8 bars: 4 per row, 2 rows
	barH := 28.0
	pad := 8.0
	barW := (canvasW - pad*5) / 4

	for i := 0; i < 8; i++ {
		col := float64(i % 4)
		row := float64(i / 4)
		x := pad + col*(barW+pad)
		y := barsY + 8 + row*(barH+14)
		e.renderOneBar(i, x, y, barW, barH)
	}
}

func (e *Engine) renderOneBar(i int, x, y, w, h float64) {
	ctx := e.ctx
	v := e.needs.get(i)

	// Background track
	ctx.Set("fillStyle", "#5a4030")
	roundRect(ctx, x, y, w, h, 6)
	ctx.Call("fill")

	// Fill colour based on value
	var fillColor string
	switch {
	case v > 0.60:
		fillColor = "#4caf50"
	case v > 0.35:
		fillColor = "#ffc107"
	default:
		fillColor = "#f44336"
	}

	// Pulse when critical
	if v < 0.20 {
		pulse := 0.7 + 0.3*math.Sin(e.time*8)
		ctx.Set("globalAlpha", pulse)
	}

	fillW := (w - 4) * v
	if fillW > 0 {
		grad := ctx.Call("createLinearGradient", x+2, y, x+2+fillW, y)
		grad.Call("addColorStop", 0, fillColor)
		grad.Call("addColorStop", 1, lighten(fillColor))
		ctx.Set("fillStyle", grad)
		roundRect(ctx, x+2, y+2, fillW, h-4, 5)
		ctx.Call("fill")
	}

	ctx.Set("globalAlpha", 1)

	// Icon and label
	ctx.Set("font", "14px sans-serif")
	ctx.Set("textAlign", "left")
	ctx.Set("fillStyle", "#fff8e8")
	ctx.Call("fillText", needIcons[i]+" "+needNames[i], x+6, y+h*0.72)

	// Percent
	pct := int(v * 100)
	ctx.Set("font", "bold 13px monospace")
	ctx.Set("textAlign", "right")
	ctx.Set("fillStyle", "#fff8e8")
	ctx.Call("fillText", itoa(pct)+"%", x+w-6, y+h*0.72)
}

// ── HUD ───────────────────────────────────────────────────────────────────────

func (e *Engine) renderHUD() {
	ctx := e.ctx

	ctx.Set("fillStyle", "#2a1a08")
	ctx.Call("fillRect", 0, 0, canvasW, hudH)

	// Day name
	dayName := dayNames[e.day-1]
	ctx.Set("fillStyle", "#f5e6c8")
	ctx.Set("font", "bold 16px 'Segoe UI', Arial, sans-serif")
	ctx.Set("textAlign", "left")
	ctx.Call("fillText", "Day "+itoa(e.day)+" — "+dayName, 14, 28)

	// Clock
	ctx.Set("font", "bold 18px monospace")
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", e.gameTimeStr(), canvasW/2, 28)

	// Mood
	ctx.Set("font", "22px sans-serif")
	ctx.Set("textAlign", "right")
	ctx.Call("fillText", e.moodEmoji(), canvasW-80, 30)

	// Score
	ctx.Set("font", "bold 15px monospace")
	ctx.Set("fillStyle", "#ffd700")
	ctx.Call("fillText", "★ "+itoa(e.score), canvasW-12, 28)

	// Speed indicator
	if e.speed > 1 {
		ctx.Set("fillStyle", "#80c0ff")
		ctx.Set("font", "12px monospace")
		ctx.Set("textAlign", "left")
		ctx.Call("fillText", fmt.Sprintf("×%.0f", e.speed), canvasW/2+60, 28)
	}
}

// ── flash message ─────────────────────────────────────────────────────────────

func (e *Engine) renderFlash() {
	if e.flash == nil {
		return
	}
	ctx := e.ctx
	alpha := e.flash.Timer / 2.5
	if alpha > 1 {
		alpha = 1
	}
	ctx.Set("font", "bold 15px 'Segoe UI', Arial, sans-serif")
	ctx.Set("textAlign", "center")
	metrics := ctx.Call("measureText", e.flash.Text)
	tw := metrics.Get("width").Float()
	pw, ph := tw+20, 26.0
	px := canvasW/2 - pw/2
	py := roomY + 10.0

	ctx.Set("globalAlpha", alpha)
	ctx.Set("fillStyle", "rgba(30,20,10,0.85)")
	roundRect(ctx, px, py, pw, ph, 8)
	ctx.Call("fill")
	ctx.Set("fillStyle", "#ffe8a0")
	ctx.Call("fillText", e.flash.Text, canvasW/2, py+ph*0.75)
	ctx.Set("globalAlpha", 1)
}

// ── overlays ──────────────────────────────────────────────────────────────────

func (e *Engine) renderPaused() {
	ctx := e.ctx
	ctx.Set("fillStyle", "rgba(0,0,0,0.55)")
	ctx.Call("fillRect", 0, 0, canvasW, canvasH)
	e.drawCentrePanel("⏸  PAUSED", "Press P to resume  |  Q to quit to menu", "#fff8e8", "#c0a880")
}

func (e *Engine) renderNight() {
	ctx := e.ctx
	// Darkness overlay
	ctx.Set("fillStyle", fmt.Sprintf("rgba(10,10,40,%.2f)", e.nightAlpha))
	ctx.Call("fillRect", 0, 0, canvasW, canvasH)

	if !e.nightDone {
		return
	}

	// Summary panel
	px, py := canvasW/2-180.0, canvasH/2-110.0
	pw, ph := 360.0, 220.0
	ctx.Set("fillStyle", "rgba(20,10,40,0.92)")
	roundRect(ctx, px, py, pw, ph, 14)
	ctx.Call("fill")
	ctx.Set("strokeStyle", "#8070b0")
	ctx.Set("lineWidth", 2)
	roundRect(ctx, px, py, pw, ph, 14)
	ctx.Call("stroke")

	ctx.Set("textAlign", "center")
	ctx.Set("fillStyle", "#f0e8ff")

	ctx.Set("font", "bold 22px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", "🌙 Night "+itoa(e.day), canvasW/2, py+38)

	ctx.Set("font", "14px 'Segoe UI', Arial, sans-serif")
	ctx.Set("fillStyle", "#c0b0e0")
	ctx.Call("fillText", "Your cat is resting…", canvasW/2, py+64)

	// Need averages
	ctx.Set("font", "bold 13px monospace")
	ctx.Set("fillStyle", "#e0d8ff")
	for i := 0; i < 6; i++ {
		col := i % 3
		row := i / 3
		nx := float64(px) + 40 + float64(col)*110
		ny := py + 100 + float64(row)*32
		v := e.needs.get(i)
		bar := ""
		filled := int(v * 8)
		for j := 0; j < 8; j++ {
			if j < filled {
				bar += "█"
			} else {
				bar += "░"
			}
		}
		ctx.Set("textAlign", "left")
		ctx.Call("fillText", needIcons[i]+" "+bar, nx, ny)
	}

	ctx.Set("textAlign", "center")
	ctx.Set("fillStyle", "#ffd700")
	ctx.Set("font", "bold 18px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", "Score: ★ "+itoa(e.score), canvasW/2, py+178)

	ctx.Set("fillStyle", "#80ff80")
	ctx.Set("font", "14px 'Segoe UI', Arial, sans-serif")
	if e.day < MaxDays {
		ctx.Call("fillText", "Press Space or click to start Day "+itoa(e.day+1), canvasW/2, py+206)
	} else {
		ctx.Call("fillText", "Press Space or click to see the ending!", canvasW/2, py+206)
	}
}

func (e *Engine) renderAlert() {
	if e.activeAlert == nil {
		return
	}
	ctx := e.ctx
	ctx.Set("fillStyle", "rgba(0,0,0,0.5)")
	ctx.Call("fillRect", 0, 0, canvasW, canvasH)

	pulse := 0.7 + 0.3*math.Sin(e.time*5)
	px, py := canvasW/2-200.0, canvasH/2-70.0
	pw, ph := 400.0, 140.0

	ctx.Set("fillStyle", "rgba(60,20,10,0.95)")
	roundRect(ctx, px, py, pw, ph, 14)
	ctx.Call("fill")

	ctx.Set("strokeStyle", fmt.Sprintf("rgba(255,100,60,%.2f)", pulse))
	ctx.Set("lineWidth", 3)
	roundRect(ctx, px, py, pw, ph, 14)
	ctx.Call("stroke")

	ctx.Set("textAlign", "center")
	ctx.Set("font", "28px sans-serif")
	ctx.Call("fillText", "⚠", canvasW/2, py+46)

	ctx.Set("fillStyle", "#ffe8d0")
	ctx.Set("font", "bold 15px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", e.activeAlert.Message, canvasW/2, py+80)

	ctx.Set("fillStyle", "#ffcc80")
	ctx.Set("font", "13px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", "Click anywhere to continue", canvasW/2, py+114)
}

func (e *Engine) renderGameOver() {
	ctx := e.ctx
	ctx.Set("fillStyle", "rgba(0,0,0,0.7)")
	ctx.Call("fillRect", 0, 0, canvasW, canvasH)

	px, py := canvasW/2-200.0, canvasH/2-120.0
	pw, ph := 400.0, 240.0

	ctx.Set("fillStyle", "rgba(40,10,10,0.96)")
	roundRect(ctx, px, py, pw, ph, 14)
	ctx.Call("fill")
	ctx.Set("strokeStyle", "#993333")
	ctx.Set("lineWidth", 2)
	roundRect(ctx, px, py, pw, ph, 14)
	ctx.Call("stroke")

	ctx.Set("textAlign", "center")
	ctx.Set("font", "40px sans-serif")
	ctx.Call("fillText", "😿", canvasW/2, py+54)

	ctx.Set("fillStyle", "#ff8080")
	ctx.Set("font", "bold 22px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", "Your cat got sick…", canvasW/2, py+92)

	ctx.Set("fillStyle", "#ffb090")
	ctx.Set("font", "14px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", "You made it to Day "+itoa(e.lastDays), canvasW/2, py+120)

	ctx.Set("fillStyle", "#ffd700")
	ctx.Set("font", "bold 18px monospace")
	ctx.Call("fillText", "★ "+itoa(e.lastScore)+" points", canvasW/2, py+152)

	ctx.Set("fillStyle", "#c0a880")
	ctx.Set("font", "13px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", "Space / click — try again  |  Q — menu", canvasW/2, py+200)
}

func (e *Engine) renderVictory() {
	ctx := e.ctx
	ctx.Set("fillStyle", "rgba(0,0,0,0.6)")
	ctx.Call("fillRect", 0, 0, canvasW, canvasH)

	// Confetti-like dots
	for i := 0; i < 20; i++ {
		phase := e.time*2 + float64(i)*0.8
		x := canvasW/2 + math.Sin(phase)*200*float64(i%3+1)/3
		y := (e.time*60 + float64(i*30))
		for y > canvasH {
			y -= canvasH
		}
		colors := []string{"#ff6b6b", "#ffd93d", "#6bcb77", "#4d96ff", "#ff9f40"}
		ctx.Set("fillStyle", colors[i%len(colors)])
		ctx.Call("fillRect", x, y, 8, 8)
	}

	px, py := canvasW/2-210.0, canvasH/2-130.0
	pw, ph := 420.0, 260.0

	ctx.Set("fillStyle", "rgba(10,30,10,0.95)")
	roundRect(ctx, px, py, pw, ph, 14)
	ctx.Call("fill")
	ctx.Set("strokeStyle", "#44cc44")
	ctx.Set("lineWidth", 2)
	roundRect(ctx, px, py, pw, ph, 14)
	ctx.Call("stroke")

	ctx.Set("textAlign", "center")
	ctx.Set("font", "42px sans-serif")
	ctx.Call("fillText", "😸🎉", canvasW/2, py+56)

	ctx.Set("fillStyle", "#88ff88")
	ctx.Set("font", "bold 24px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", "You're a great cat owner!", canvasW/2, py+96)

	ctx.Set("fillStyle", "#ccffcc")
	ctx.Set("font", "15px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", "7 happy days completed 🐱", canvasW/2, py+126)

	ctx.Set("fillStyle", "#ffd700")
	ctx.Set("font", "bold 26px monospace")
	ctx.Call("fillText", "★ "+itoa(e.lastScore)+" points", canvasW/2, py+168)

	ctx.Set("fillStyle", "#80ff80")
	ctx.Set("font", "14px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", "Click or Space to return to menu", canvasW/2, py+220)
}

// ── main menu ─────────────────────────────────────────────────────────────────

func (e *Engine) renderMainMenu() {
	ctx := e.ctx

	e.renderMenuBg()
	e.renderMenuCat()

	// Title with glow
	titleBob := math.Sin(e.time*1.4) * 2
	ctx.Set("shadowBlur", 14+8*math.Sin(e.time*1.8))
	ctx.Set("shadowColor", "rgba(255,210,60,0.75)")
	ctx.Set("textAlign", "center")
	ctx.Set("fillStyle", "#ffd700")
	ctx.Set("font", "bold 44px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", "Purr & Care", canvasW/2, 208+titleBob)
	ctx.Set("shadowBlur", 0)

	subtitleAlpha := 0.75 + 0.25*math.Sin(e.time*0.9)
	ctx.Set("globalAlpha", subtitleAlpha)
	ctx.Set("fillStyle", "#f5d080")
	ctx.Set("font", "15px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", "A cat care simulation for curious kids", canvasW/2, 234)
	ctx.Set("globalAlpha", 1)

	e.renderMenuCards()

	// Speed
	ctx.Set("fillStyle", "#80c0ff")
	ctx.Set("font", "13px monospace")
	ctx.Set("textAlign", "center")
	ctx.Call("fillText", fmt.Sprintf("Speed: ×%.0f  (S to change)", e.speed), canvasW/2, 494)

	// Start button with glow
	pulse := 0.72 + 0.28*math.Sin(e.time*2.8)
	ctx.Set("globalAlpha", pulse)
	ctx.Set("shadowBlur", 16*pulse)
	ctx.Set("shadowColor", "rgba(255,220,0,0.9)")
	ctx.Set("fillStyle", "#ffe44d")
	ctx.Set("font", "bold 22px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", "▶  Press Space or click to start", canvasW/2, 532)
	ctx.Set("globalAlpha", 1)
	ctx.Set("shadowBlur", 0)

	// Last result hint
	if e.lastScore > 0 {
		ctx.Set("fillStyle", "rgba(200,180,120,0.7)")
		ctx.Set("font", "12px monospace")
		ctx.Call("fillText", fmt.Sprintf("Last game: %d pts, day %d", e.lastScore, e.lastDays), canvasW/2, 554)
	}

	e.renderMenuScores()
}

func (e *Engine) renderMenuBg() {
	ctx := e.ctx
	t := e.time

	// Background gradient
	grad := ctx.Call("createLinearGradient", 0, 0, 0, canvasH)
	grad.Call("addColorStop", 0, "#1e1008")
	grad.Call("addColorStop", 0.5, "#2e1c0c")
	grad.Call("addColorStop", 1, "#3e2810")
	ctx.Set("fillStyle", grad)
	ctx.Call("fillRect", 0, 0, canvasW, canvasH)

	// Twinkling stars
	for i := 0; i < 28; i++ {
		seed := float64(i) * 2.618
		sx := math.Mod(seed*317.4, canvasW)
		sy := math.Mod(seed*213.7, canvasH*0.72)
		alpha := 0.15 + 0.22*math.Sin(t*1.1+seed)
		r := 0.8 + 0.7*math.Sin(t*0.7+seed*1.3)
		ctx.Set("fillStyle", fmt.Sprintf("rgba(255,240,200,%.2f)", alpha))
		ctx.Call("beginPath")
		ctx.Call("arc", sx, sy, r, 0, math.Pi*2)
		ctx.Call("fill")
	}

	// Floating particles — hearts and paw prints rising slowly
	for i := 0; i < 12; i++ {
		seed := float64(i)*1.618 + 0.4
		cycleT := math.Mod(t*0.10+seed, 1.0)
		px := 30 + math.Mod(seed*61.3, canvasW-60) + math.Sin(t*0.35+seed)*18
		py := canvasH - cycleT*(canvasH+40) + 30
		alpha := math.Sin(cycleT*math.Pi) * 0.45
		if alpha < 0.02 {
			continue
		}
		ctx.Set("globalAlpha", alpha)
		ctx.Set("textAlign", "center")
		if i%3 == 0 {
			ctx.Set("fillStyle", "#ff6090")
			ctx.Set("font", "16px sans-serif")
			ctx.Call("fillText", "♥", px, py)
		} else if i%3 == 1 {
			ctx.Set("font", "14px sans-serif")
			ctx.Call("fillText", "🐾", px, py)
		} else {
			ctx.Set("fillStyle", "rgba(255,230,100,1)")
			ctx.Set("font", "11px sans-serif")
			ctx.Call("fillText", "✦", px, py)
		}
	}
	ctx.Set("globalAlpha", 1)
}

func (e *Engine) renderMenuCat() {
	ctx := e.ctx
	t := e.time
	cx, cy := canvasW/2-30.0, 158.0

	// Animated toy feather to the right — inviting the cat
	toyx := cx + 90.0
	toySwing := math.Sin(t*3.0) * 14
	ctx.Set("strokeStyle", "#8b6040")
	ctx.Set("lineWidth", 2.5)
	ctx.Set("lineCap", "round")
	ctx.Call("beginPath")
	ctx.Call("moveTo", toyx, cy-80)
	ctx.Call("lineTo", toyx+toySwing, cy-58)
	ctx.Call("stroke")
	for _, ang := range []float64{-0.35, 0, 0.35} {
		ctx.Set("strokeStyle", "#e07030")
		ctx.Set("lineWidth", 2.5)
		ctx.Call("beginPath")
		fx := toyx + toySwing + math.Sin(ang)*12
		fy := cy - 58 + math.Cos(ang)*8
		ctx.Call("moveTo", toyx+toySwing, cy-58)
		ctx.Call("lineTo", fx, fy)
		ctx.Call("stroke")
	}

	// Cat drawn at 1.4× scale
	ctx.Call("save")
	ctx.Call("translate", cx, cy)
	ctx.Call("scale", 1.4, 1.4)

	breathe := math.Sin(t*1.9) * 2.0
	tailSway := math.Sin(t*1.4) * 12.0
	headSway := math.Sin(t*0.65) * 5.0

	// Tail
	ctx.Set("lineWidth", 9)
	ctx.Set("lineCap", "round")
	ctx.Set("strokeStyle", colCatPointL)
	ctx.Call("beginPath")
	ctx.Call("moveTo", 14, -8+breathe)
	ctx.Call("bezierCurveTo", 42, -8+breathe, 44+tailSway, -52, 22, -58)
	ctx.Call("stroke")
	ctx.Set("strokeStyle", colCatPoint)
	ctx.Set("lineWidth", 7)
	ctx.Call("beginPath")
	ctx.Call("moveTo", 34, -46)
	ctx.Call("bezierCurveTo", 42+tailSway, -52, 30, -60, 22, -58)
	ctx.Call("stroke")

	// Body
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", 0, -30+breathe, 24, 28, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatChest)
	ctx.Call("beginPath")
	ctx.Call("ellipse", 2, -26+breathe, 13, 17, 0, 0, math.Pi*2)
	ctx.Call("fill")

	// Paws
	ctx.Set("fillStyle", colCatBody)
	ctx.Call("beginPath")
	ctx.Call("ellipse", -13, -5, 11, 7, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("ellipse", 13, -5, 11, 7, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Set("fillStyle", colCatPoint)
	ctx.Call("beginPath")
	ctx.Call("ellipse", -13, -3, 8, 5, 0, 0, math.Pi*2)
	ctx.Call("fill")
	ctx.Call("beginPath")
	ctx.Call("ellipse", 13, -3, 8, 5, 0, 0, math.Pi*2)
	ctx.Call("fill")

	// Head with sway — looking toward the toy
	hx := headSway + 4
	hy := -66.0 + breathe
	e.drawCatHead(hx, hy, false)

	ctx.Call("restore")

	// Periodic "purr~" text drifting off the cat
	purrCycle := math.Mod(t*0.22, 1.0)
	if purrCycle > 0.55 {
		pp := (purrCycle - 0.55) / 0.45
		pAlpha := math.Sin(pp * math.Pi)
		ctx.Set("globalAlpha", pAlpha*0.85)
		ctx.Set("fillStyle", "#ffb8d8")
		ctx.Set("font", "italic 13px 'Segoe UI', Arial, sans-serif")
		ctx.Set("textAlign", "center")
		ctx.Call("fillText", "purr~", cx-55+pp*10, cy-105-pp*22)
		ctx.Set("globalAlpha", 1)
	}

	// Periodic heart floating above cat
	heartCycle := math.Mod(t*0.18+0.3, 1.0)
	if heartCycle > 0.6 {
		hp := (heartCycle - 0.6) / 0.4
		hAlpha := math.Sin(hp * math.Pi)
		ctx.Set("globalAlpha", hAlpha*0.9)
		ctx.Set("fillStyle", "#ff6090")
		ctx.Set("font", "18px sans-serif")
		ctx.Set("textAlign", "center")
		ctx.Call("fillText", "♥", cx+10, cy-105-hp*30)
		ctx.Set("globalAlpha", 1)
	}
}

func (e *Engine) renderMenuCards() {
	ctx := e.ctx
	type card struct{ icon, label string }
	cards := []card{
		{"🍽", "Feed"},
		{"💧", "Water"},
		{"🧸", "Play"},
		{"😴", "Rest"},
		{"🚿", "Clean"},
		{"🤝", "Love"},
		{"🪮", "Brush"},
		{"🐾", "Scratch"},
	}
	cw := 86.0
	gap := 6.0
	totalW := float64(len(cards))*cw + float64(len(cards)-1)*gap
	startX := (canvasW - totalW) / 2
	for i, c := range cards {
		bob := math.Sin(e.time*2.2+float64(i)*0.55) * 4
		x := startX + float64(i)*(cw+gap)
		y := 258.0 + bob
		bright := 0.10 + 0.08*math.Sin(e.time*1.8+float64(i)*0.8)
		ctx.Set("fillStyle", fmt.Sprintf("rgba(255,220,140,%.2f)", bright))
		roundRect(ctx, x, y, cw, 64, 8)
		ctx.Call("fill")
		borderA := 0.20 + 0.15*math.Sin(e.time*2.0+float64(i)*0.7)
		ctx.Set("strokeStyle", fmt.Sprintf("rgba(255,200,100,%.2f)", borderA))
		ctx.Set("lineWidth", 1.2)
		roundRect(ctx, x, y, cw, 64, 8)
		ctx.Call("stroke")
		ctx.Set("font", "20px sans-serif")
		ctx.Set("textAlign", "center")
		ctx.Set("fillStyle", "#ffe8c0")
		ctx.Call("fillText", c.icon, x+cw/2, y+26)
		ctx.Set("font", "10px 'Segoe UI', Arial, sans-serif")
		ctx.Set("fillStyle", "#c8a860")
		ctx.Call("fillText", c.label, x+cw/2, y+42)
	}
}

func (e *Engine) renderMenuScores() {
	ctx := e.ctx
	if len(e.topScores) == 0 {
		return
	}
	ctx.Set("fillStyle", "rgba(0,0,0,0.4)")
	roundRect(ctx, canvasW/2-160, 548, 320, 34, 8)
	ctx.Call("fill")

	ctx.Set("textAlign", "center")
	ctx.Set("fillStyle", "#c0a860")
	ctx.Set("font", "12px monospace")

	text := "🏆 "
	for i, s := range e.topScores {
		if i > 0 {
			text += "  "
		}
		text += s.Nick + " " + itoa(s.Score)
		if i >= 4 {
			break
		}
	}
	ctx.Call("fillText", text, canvasW/2, 570)
}

// ── utility ───────────────────────────────────────────────────────────────────

func (e *Engine) drawCentrePanel(title, sub, titleCol, subCol string) {
	ctx := e.ctx
	px, py := canvasW/2-200.0, canvasH/2-70.0
	pw, ph := 400.0, 140.0
	ctx.Set("fillStyle", "rgba(20,14,6,0.92)")
	roundRect(ctx, px, py, pw, ph, 14)
	ctx.Call("fill")
	ctx.Set("strokeStyle", "rgba(180,150,80,0.5)")
	ctx.Set("lineWidth", 2)
	roundRect(ctx, px, py, pw, ph, 14)
	ctx.Call("stroke")
	ctx.Set("textAlign", "center")
	ctx.Set("fillStyle", titleCol)
	ctx.Set("font", "bold 26px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", title, canvasW/2, py+50)
	ctx.Set("fillStyle", subCol)
	ctx.Set("font", "15px 'Segoe UI', Arial, sans-serif")
	ctx.Call("fillText", sub, canvasW/2, py+90)
}

func roundRect(ctx js.Value, x, y, w, h, r float64) {
	ctx.Call("beginPath")
	ctx.Call("moveTo", x+r, y)
	ctx.Call("lineTo", x+w-r, y)
	ctx.Call("quadraticCurveTo", x+w, y, x+w, y+r)
	ctx.Call("lineTo", x+w, y+h-r)
	ctx.Call("quadraticCurveTo", x+w, y+h, x+w-r, y+h)
	ctx.Call("lineTo", x+r, y+h)
	ctx.Call("quadraticCurveTo", x, y+h, x, y+h-r)
	ctx.Call("lineTo", x, y+r)
	ctx.Call("quadraticCurveTo", x, y, x+r, y)
	ctx.Call("closePath")
}

func lighten(hex string) string {
	// Simple colour map for gradient tops
	switch hex {
	case "#4caf50":
		return "#81c784"
	case "#ffc107":
		return "#ffd54f"
	case "#f44336":
		return "#e57373"
	}
	return hex
}

// ScoreEntry is duplicated here for the WASM build (server version is in server/).
type ScoreEntry struct {
	Nick      string `json:"nick"`
	Score     int    `json:"score"`
	Days      int    `json:"days"`
	Timestamp string `json:"timestamp"`
}
