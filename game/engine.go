//go:build js && wasm

package main

import (
	"math/rand"
	"syscall/js"
)

// ── enums ─────────────────────────────────────────────────────────────────────

type GameState int

const (
	StateMainMenu GameState = iota
	StatePlaying
	StatePaused
	StateNight
	StateAlert
	StateGameOver
	StateVictory
)

type CatState int

const (
	CatIdle CatState = iota
	CatWalking
	CatEating
	CatDrinking
	CatPlaying
	CatSleeping
	CatGrooming
	CatScratch
	CatLitter
	CatPetting
	CatLookOut
)

// ── types ─────────────────────────────────────────────────────────────────────

type Needs struct {
	Hunger  float64
	Thirst  float64
	Fun     float64
	Energy  float64
	Hygiene float64
	Social  float64
	Coat    float64
	Claws   float64
}

func (n *Needs) get(i int) float64 {
	switch i {
	case 0:
		return n.Hunger
	case 1:
		return n.Thirst
	case 2:
		return n.Fun
	case 3:
		return n.Energy
	case 4:
		return n.Hygiene
	case 5:
		return n.Social
	case 6:
		return n.Coat
	case 7:
		return n.Claws
	}
	return 1
}

func (n *Needs) set(i int, v float64) {
	v = clamp01(v)
	switch i {
	case 0:
		n.Hunger = v
	case 1:
		n.Thirst = v
	case 2:
		n.Fun = v
	case 3:
		n.Energy = v
	case 4:
		n.Hygiene = v
	case 5:
		n.Social = v
	case 6:
		n.Coat = v
	case 7:
		n.Claws = v
	}
}

func (n *Needs) add(i int, delta float64) {
	n.set(i, n.get(i)+delta)
}

func (n *Needs) avg() float64 {
	return (n.Hunger + n.Thirst + n.Fun + n.Energy + n.Hygiene + n.Social + n.Coat + n.Claws) / 8.0
}

type Cat struct {
	X, Y      float64
	TargetX   float64
	State     CatState
	StateTime float64 // time remaining in current action
	IdleTime  float64 // how long has been idle
	AnimFrame int
	AnimTime  float64
	Direction int // -1 left, +1 right
	// which object the cat is heading to / acting on
	TargetObj int // index into engine.objects, -1 if none
}

type RoomObject struct {
	Name    string
	Label   string // emoji + short name
	X       float64
	W, H    float64
	FootY   float64 // Y of bottom of object
	Filled  float64 // 0..1 remaining capacity (bowls, litter)
	Cap     int     // max visits before refill
	Visits  int     // visits since last refill
	Hovered bool
	// which need index this satisfies (primary)
	NeedIdx int
	// effect deltas applied when cat uses this object
	Effects [8]float64
}

type FlashMsg struct {
	Text  string
	Timer float64
}

type HeartFx struct {
	X, Y float64
	VY   float64
	T    float64
}

type Alert struct {
	NeedIdx int
	Message string
}

// ── engine ────────────────────────────────────────────────────────────────────

type Engine struct {
	canvas js.Value
	ctx    js.Value

	state GameState

	cat     Cat
	objects []RoomObject
	needs   Needs

	// Time
	gameTime float64 // in-game minutes since 08:00 (0 = 08:00, 840 = 22:00)
	day      int     // 1–7
	speed    float64 // time multiplier

	// Scoring
	score      int
	dayScore   int
	scoreTick  float64 // accumulator for per-second scoring
	dayHadAlert bool

	// Alert tracking
	alertTimers  [8]float64 // real seconds each need has been below AlertThreshold
	zeroTimers   [8]float64 // real seconds each need has been at 0
	pendingAlerts []Alert
	activeAlert  *Alert

	// Night transition
	nightAlpha float64 // 0→0.75 fade
	nightDone  bool    // recovery applied

	// FX
	flash  *FlashMsg
	hearts []HeartFx
	time   float64 // total elapsed real seconds (for animations)

	// Sound event — JS polls and clears this each frame
	soundEvent  string  // "meow" | "purr" | ""
	meowCooldown float64 // seconds until next meow is allowed

	// Leaderboard (fetched from server)
	topScores []ScoreEntry

	// Last result (for menu display)
	lastScore int
	lastDays  int
}

// ── constructor ───────────────────────────────────────────────────────────────

func NewEngine(canvas js.Value) *Engine {
	ctx := canvas.Call("getContext", "2d")
	e := &Engine{
		canvas: canvas,
		ctx:    ctx,
		speed:  5.0,
	}
	e.initObjects()
	e.enterMainMenu()
	go e.fetchScores()
	return e
}

func (e *Engine) initObjects() {
	cfg := dayConfigs[0]
	e.objects = []RoomObject{
		{
			Name: "scratcher", Label: "🌿 Scratcher",
			X: scratcerX, W: 50, H: 90, FootY: objFootY,
			Cap: 99, NeedIdx: 7,
			Effects: [8]float64{0, 0, 0.08, 0, 0.05, 0, 0, 0.35},
		},
		{
			Name: "food", Label: "🍽 Food Bowl",
			X: foodX, W: 60, H: 30, FootY: objFootY,
			Cap: cfg.BowlCap, NeedIdx: 0,
			Effects: [8]float64{0.35, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			Name: "water", Label: "💧 Water Bowl",
			X: waterX, W: 55, H: 28, FootY: objFootY,
			Cap: cfg.BowlCap, NeedIdx: 1,
			Effects: [8]float64{0, 0.35, 0, 0, 0, 0, 0, 0},
		},
		{
			Name: "toy", Label: "🧸 Toy Wand",
			X: toyX, W: 60, H: 80, FootY: objFootY,
			Cap: 99, NeedIdx: 2,
			Effects: [8]float64{0, 0, 0.30, -0.08, 0, 0, 0, 0},
		},
		{
			Name: "bed", Label: "😴 Cat Bed",
			X: bedX, W: 100, H: 45, FootY: objFootY,
			Cap: 99, NeedIdx: 3,
			Effects: [8]float64{0, 0, 0.05, 0.40, 0, 0, 0, 0},
		},
		{
			Name: "litter", Label: "🚿 Litter Box",
			X: litterX, W: 70, H: 40, FootY: objFootY,
			Cap: cfg.BowlCap, NeedIdx: 4,
			Effects: [8]float64{0, 0, 0, 0, 0.30, 0, 0, 0},
		},
		{
			Name: "brush", Label: "🪮 Brush",
			X: brushX, W: 44, H: 70, FootY: objFootY,
			Cap: 99, NeedIdx: 6,
			Effects: [8]float64{0, 0, 0.05, 0, 0.05, 0.15, 0.40, 0},
		},
	}
	for i := range e.objects {
		e.objects[i].Filled = 1.0
	}
}

// resetObjectCaps updates bowl capacities for the current day.
func (e *Engine) resetObjectCaps() {
	cfg := dayConfigs[e.day-1]
	for i := range e.objects {
		if e.objects[i].Name == "food" || e.objects[i].Name == "water" || e.objects[i].Name == "litter" {
			e.objects[i].Cap = cfg.BowlCap
			e.objects[i].Filled = 1.0
			e.objects[i].Visits = 0
		}
	}
}

// ── state transitions ─────────────────────────────────────────────────────────

func (e *Engine) enterMainMenu() {
	e.state = StateMainMenu
	e.flash = nil
	e.hearts = nil
}

func (e *Engine) newGame() {
	e.needs = Needs{1, 1, 1, 1, 1, 1, 1, 1}
	e.day = 1
	e.gameTime = 0
	e.score = 0
	e.dayScore = 0
	e.scoreTick = 0
	e.dayHadAlert = false
	e.alertTimers = [8]float64{}
	e.zeroTimers = [8]float64{}
	e.pendingAlerts = nil
	e.activeAlert = nil
	e.nightAlpha = 0
	e.nightDone = false
	e.flash = nil
	e.hearts = nil
	e.time = 0

	e.resetObjectCaps()

	e.cat = Cat{
		X: catStartX, Y: catFootY,
		TargetX:   catStartX,
		State:     CatIdle,
		Direction: 1,
		TargetObj: -1,
	}

	e.state = StatePlaying
}

func (e *Engine) enterNight() {
	e.state = StateNight
	e.nightAlpha = 0
	e.nightDone = false
	e.cat.State = CatIdle
	e.cat.TargetObj = -1
	// send cat to bed
	e.sendCatTo(4) // bed index
}

func (e *Engine) nextDay() {
	e.day++
	if e.day > MaxDays {
		e.lastScore = e.score
		e.lastDays = MaxDays
		e.state = StateVictory
		return
	}
	e.gameTime = 0
	e.nightAlpha = 0
	e.nightDone = false
	e.dayScore = 0
	e.dayHadAlert = false
	e.alertTimers = [8]float64{}
	e.zeroTimers = [8]float64{}
	e.pendingAlerts = nil
	e.activeAlert = nil
	e.resetObjectCaps()
	e.cat.State = CatIdle
	e.cat.TargetObj = -1
	e.cat.StateTime = 0
	e.state = StatePlaying
}

// ── cat steering ──────────────────────────────────────────────────────────────

// sendCatTo directs the cat toward object at index i.
func (e *Engine) sendCatTo(i int) {
	if i < 0 || i >= len(e.objects) {
		return
	}
	obj := &e.objects[i]
	targetX := obj.X
	if obj.Name == "scratcher" {
		targetX = obj.X + 34 // stand to the right; paws reach back to post face
	}
	e.cat.TargetX = targetX
	e.cat.TargetObj = i
	if e.cat.X < targetX {
		e.cat.Direction = 1
	} else {
		e.cat.Direction = -1
	}
	e.cat.State = CatWalking
	e.cat.StateTime = 0
}

// applyObjectEffect applies an object's effects to cat needs and marks a visit.
func (e *Engine) applyObjectEffect(i int) {
	if i < 0 || i >= len(e.objects) {
		return
	}
	obj := &e.objects[i]
	for j, delta := range obj.Effects {
		if delta != 0 {
			e.needs.add(j, delta)
		}
	}
	if obj.Name == "food" || obj.Name == "water" || obj.Name == "litter" {
		obj.Visits++
		if obj.Cap > 0 {
			obj.Filled = 1.0 - float64(obj.Visits)/float64(obj.Cap)
			if obj.Filled < 0 {
				obj.Filled = 0
			}
		}
	}
	if obj.Name == "brush" {
		e.score += ScoreBrush
		e.spawnHearts()
	}
}

// refillObject resets a bowl/litter box.
func (e *Engine) refillObject(i int) {
	if i < 0 || i >= len(e.objects) {
		return
	}
	obj := &e.objects[i]
	obj.Visits = 0
	obj.Filled = 1.0
	e.score += ScoreRefill
	e.setFlash("Refilled! +" + itoa(ScoreRefill))
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (e *Engine) audioScene() string {
	switch e.state {
	case StateMainMenu:
		return "menu"
	case StateNight:
		return "night"
	case StateGameOver:
		return "gameover"
	case StateVictory:
		return "victory"
	default:
		return "playing"
	}
}

func (e *Engine) stateName() string {
	switch e.state {
	case StateMainMenu:
		return "menu"
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	case StateNight:
		return "night"
	case StateAlert:
		return "alert"
	case StateGameOver:
		return "gameover"
	case StateVictory:
		return "victory"
	}
	return "unknown"
}

func (e *Engine) setFlash(text string) {
	e.flash = &FlashMsg{Text: text, Timer: 2.5}
}

func (e *Engine) meow() {
	if e.meowCooldown > 0 {
		return
	}
	e.soundEvent = "meow"
	e.meowCooldown = 8.0 // minimum 8 real seconds between meows
}

func (e *Engine) moodEmoji() string {
	avg := e.needs.avg()
	switch {
	case avg > 0.75:
		return "😄"
	case avg > 0.50:
		return "😊"
	case avg > 0.35:
		return "😐"
	case avg > 0.20:
		return "😟"
	default:
		return "😿"
	}
}

func (e *Engine) gameTimeStr() string {
	// gameTime 0 = 08:00, 840 = 22:00
	totalMin := int(e.gameTime) + 8*60
	h := (totalMin / 60) % 24
	m := totalMin % 60
	s := ""
	if h < 10 {
		s = "0"
	}
	s += itoa(h) + ":"
	if m < 10 {
		s += "0"
	}
	s += itoa(m)
	return s
}

func (e *Engine) lowestNeed() int {
	idx, val := 0, e.needs.get(0)
	for i := 1; i < 6; i++ {
		v := e.needs.get(i)
		if v < val {
			val = v
			idx = i
		}
	}
	return idx
}

// objectForNeed returns index of the object that satisfies need i.
var needToObj = [8]int{1, 2, 3, 4, 5, 6, 6, 0} // hunger→food, thirst→water, fun→toy, energy→bed, hygiene→litter, social→brush, coat→brush, claws→scratcher

func (e *Engine) objectForNeed(needIdx int) int {
	switch needIdx {
	case 0:
		return 1 // food bowl
	case 1:
		return 2 // water bowl
	case 2:
		return 3 // toy
	case 3:
		return 4 // bed
	case 4:
		return 5 // litter
	case 5:
		return 6 // brush — satisfies social
	case 6:
		return 6 // brush — satisfies coat
	case 7:
		return 0 // scratcher — satisfies claws
	}
	return -1
}

// catStateForObj returns the CatState for acting on object i.
func catStateForObj(i int) (CatState, float64) {
	switch i {
	case 0:
		return CatScratch, 3.5 // scratcher
	case 1:
		return CatEating, 4.0 // food
	case 2:
		return CatDrinking, 3.0 // water
	case 3:
		return CatPlaying, 5.0 // toy
	case 4:
		return CatSleeping, 8.0 // bed
	case 5:
		return CatLitter, 4.0 // litter
	case 6:
		return CatGrooming, 5.0 // brush
	}
	return CatIdle, 0
}

func (e *Engine) fetchScores() {
	// Fetch top scores via JS fetch
	promise := js.Global().Call("fetch", "/api/scores?n=5")
	// We use a simplified approach: set a global that JS can check
	// For now scores are loaded on game start via JS
	_ = promise
}

// ── misc helpers ──────────────────────────────────────────────────────────────

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func randF(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}
