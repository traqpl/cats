//go:build js && wasm

package main

import "math"

const inGameDayStart = 0.0  // 08:00
const inGameNightAt = 840.0 // 22:00 (14 hours × 60 min)

func (e *Engine) Update(dt float64) {
	e.time += dt

	switch e.state {
	case StatePlaying:
		e.updatePlaying(dt)
	case StateNight:
		e.updateNight(dt)
	case StateAlert:
		// game time paused; just animate FX
		e.updateFX(dt)
	}
}

func (e *Engine) updatePlaying(dt float64) {
	cfg := dayConfigs[e.day-1]

	// Advance in-game time (1 real sec = 1 in-game min × speed)
	e.gameTime += dt * e.speed

	// Check for nightfall
	if e.gameTime >= inGameNightAt {
		e.gameTime = inGameNightAt
		e.enterNight()
		return
	}

	// Drain needs (per in-game minute)
	drainMin := dt * e.speed // in-game minutes elapsed
	e.needs.Hunger = clamp01(e.needs.Hunger - cfg.Hunger*drainMin)
	e.needs.Thirst = clamp01(e.needs.Thirst - cfg.Thirst*drainMin)
	e.needs.Fun = clamp01(e.needs.Fun - cfg.Fun*drainMin)
	e.needs.Energy = clamp01(e.needs.Energy - cfg.Energy*drainMin)
	e.needs.Hygiene = clamp01(e.needs.Hygiene - cfg.Hygiene*drainMin)
	e.needs.Social = clamp01(e.needs.Social - cfg.Social*drainMin)
	e.needs.Coat = clamp01(e.needs.Coat - cfg.Coat*drainMin)
	e.needs.Claws = clamp01(e.needs.Claws - cfg.Claws*drainMin)

	// Alert and zero timers (use real seconds, divided by speed so alerts feel
	// proportional to game pace)
	realDt := dt
	for i := 0; i < 8; i++ {
		v := e.needs.get(i)
		if v < AlertThreshold {
			e.alertTimers[i] += realDt
		} else {
			e.alertTimers[i] = math.Max(0, e.alertTimers[i]-realDt*2)
		}
		if v <= 0 {
			e.zeroTimers[i] += realDt
		} else {
			e.zeroTimers[i] = 0
		}
	}

	// Fire alert if threshold exceeded and no active alert
	if e.activeAlert == nil && len(e.pendingAlerts) == 0 {
		for i := 0; i < 8; i++ {
			if e.alertTimers[i] >= cfg.AlertTime {
				e.alertTimers[i] = 0
				e.pendingAlerts = append(e.pendingAlerts, Alert{NeedIdx: i, Message: alertMessages[i]})
				e.dayHadAlert = true
			}
		}
	}
	if e.activeAlert == nil && len(e.pendingAlerts) > 0 {
		a := e.pendingAlerts[0]
		e.pendingAlerts = e.pendingAlerts[1:]
		e.activeAlert = &a
		e.meowCooldown = 0 // alert overrides cooldown
		e.meow()
		e.state = StateAlert
		return
	}

	// Game over — need at 0 for too long
	for i := 0; i < 8; i++ {
		if e.zeroTimers[i] >= cfg.CriticalTime {
			e.lastScore = e.score
			e.lastDays = e.day
			e.meowCooldown = 0
			e.meow()
			e.state = StateGameOver
			return
		}
	}

	// Score tick every real second
	e.scoreTick += dt
	if e.scoreTick >= 1.0 {
		e.scoreTick -= 1.0
		allOK := true
		allGreat := true
		for i := 0; i < 8; i++ {
			v := e.needs.get(i)
			if v <= 0.5 {
				allOK = false
			}
			if v <= 0.8 {
				allGreat = false
			}
		}
		if allOK {
			e.score += ScoreTickLow
		}
		if allGreat {
			e.score += ScoreTickHigh
		}
	}

	// Meow cooldown
	if e.meowCooldown > 0 {
		e.meowCooldown -= dt
	}

	// Cat AI
	e.updateCat(dt)

	// FX
	e.updateFX(dt)
}

func (e *Engine) updateCat(dt float64) {
	cat := &e.cat

	switch cat.State {
	case CatWalking:
		dx := cat.TargetX - cat.X
		if absF(dx) < catSpeed*dt {
			cat.X = cat.TargetX
			// Arrived — start the action
			if cat.TargetObj >= 0 {
				cs, dur := catStateForObj(cat.TargetObj)
				cat.State = cs
				cat.StateTime = dur
				cat.AnimFrame = 0
				// Sound on arrival
				switch cat.TargetObj {
				case 0:
					e.soundEvent = "scratch"
				case 1:
					e.soundEvent = "eat"
				case 2:
					e.soundEvent = "drink"
				case 5:
					e.soundEvent = "litter"
				case 6:
					e.soundEvent = "brush"
				}
				// Meow when hungry/thirsty (if no other sound took the slot)
				if (cat.TargetObj == 1 || cat.TargetObj == 2) && e.soundEvent == "" {
					e.meow()
				}
			} else {
				cat.State = CatIdle
			}
		} else {
			if dx > 0 {
				cat.X += catSpeed * dt
				cat.Direction = 1
			} else {
				cat.X -= catSpeed * dt
				cat.Direction = -1
			}
		}

	case CatIdle:
		cat.IdleTime += dt
		thinkTime := randF(8, 15)
		if cat.IdleTime >= thinkTime {
			cat.IdleTime = 0
			// Meow occasionally when lonely or very bored
			if e.needs.Social < 0.35 || e.needs.Fun < 0.30 {
				e.meow()
			}
			e.catDecide()
		}

	default:
		// Active state (eating, drinking, etc.)
		cat.StateTime -= dt
		if cat.StateTime <= 0 {
			// Apply effect and return to idle
			if cat.TargetObj >= 0 {
				e.applyObjectEffect(cat.TargetObj)
			}
			cat.TargetObj = -1
			cat.State = CatIdle
			cat.StateTime = 0
			cat.IdleTime = 0
		}
	}

	// Animate
	cat.AnimTime += dt
	frames := catAnimFrames(cat.State)
	if cat.AnimTime >= 0.25 {
		cat.AnimTime = 0
		cat.AnimFrame = (cat.AnimFrame + 1) % frames
	}
}

func catAnimFrames(s CatState) int {
	switch s {
	case CatWalking:
		return 4
	case CatSleeping:
		return 2
	case CatPlaying:
		return 4
	case CatGrooming:
		return 3
	default:
		return 2
	}
}

// catDecide picks the lowest need below 0.65 and sends cat to matching object.
func (e *Engine) catDecide() {
	// Find lowest need
	lowest := e.lowestNeed()
	v := e.needs.get(lowest)

	if v < 0.65 {
		objIdx := e.objectForNeed(lowest)
		if objIdx >= 0 {
			// Check bowl not empty
			obj := &e.objects[objIdx]
			if obj.Filled <= 0 {
				// bowl is empty — cat wanders to scratcher instead
				e.sendCatTo(0)
				return
			}
			e.sendCatTo(objIdx)
			return
		}
	}

	// Nothing urgent — random wander
	r := randF(0, 1)
	if r < 0.35 {
		// Look out window — move to center area
		e.cat.TargetX = windowX + randF(-50, 50)
		e.cat.State = CatWalking
		e.cat.TargetObj = -1
		if e.cat.X < e.cat.TargetX {
			e.cat.Direction = 1
		} else {
			e.cat.Direction = -1
		}
	} else if r < 0.55 {
		e.sendCatTo(0) // scratcher
	}
	// else stay idle a bit longer
}

func (e *Engine) updateNight(dt float64) {
	// Fade in darkness
	if e.nightAlpha < 0.72 {
		e.nightAlpha += dt * 0.5
		if e.nightAlpha > 0.72 {
			e.nightAlpha = 0.72
		}
	}

	// Apply overnight recovery once (when fade is complete)
	if e.nightAlpha >= 0.70 && !e.nightDone {
		e.nightDone = true
		e.needs.Energy = clamp01(e.needs.Energy + 0.40)
		e.needs.Fun = clamp01(e.needs.Fun + 0.10)
		// day score
		if !e.dayHadAlert {
			pts := ScoreDayClean
			e.score += pts
		}
		if e.needs.avg() > 0.7 {
			e.score += ScoreDayBonus
		}
	}

	e.updateFX(dt)
}

func (e *Engine) updateFX(dt float64) {
	// Flash
	if e.flash != nil {
		e.flash.Timer -= dt
		if e.flash.Timer <= 0 {
			e.flash = nil
		}
	}

	// Hearts
	next := e.hearts[:0]
	for _, h := range e.hearts {
		h.Y += h.VY * dt
		h.T -= dt
		if h.T > 0 {
			next = append(next, h)
		}
	}
	e.hearts = next
}

func (e *Engine) spawnHearts() {
	for i := 0; i < 3; i++ {
		e.hearts = append(e.hearts, HeartFx{
			X:  e.cat.X + randF(-20, 20),
			Y:  e.cat.Y - 60 - randF(0, 20),
			VY: -40 - randF(0, 20),
			T:  1.5,
		})
	}
	e.soundEvent = "purr"
}
