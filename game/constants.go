//go:build js && wasm

package main

const (
	canvasW = 800.0
	canvasH = 590.0

	// HUD strip at top
	hudH = 44.0

	// Room area
	roomX = 0.0
	roomY = hudH
	roomW = canvasW
	roomH = 350.0

	// Floor Y
	floorY = roomY + roomH

	// Need bars panel (below room)
	barsY = floorY + 6
	barsH = canvasH - barsY // ~190px for 2 rows of bars

	// Object floor anchor (center X, feet Y)
	scratcerX = 75.0
	foodX     = 175.0
	waterX    = 255.0
	brushX    = 325.0
	toyX      = 430.0
	bedX      = 570.0
	litterX   = 690.0
	windowX   = 400.0 // center of window in background

	objFootY = floorY // objects sit on floor

	// Cat
	catStartX = 400.0
	catFootY  = floorY
	catSpeed  = 140.0 // px/s

	MaxDays = 5

	// Need thresholds
	AlertThreshold = 0.20
	AlertTime      = 4.0 // real seconds below threshold before alert fires
	CriticalTime   = 8.0 // real seconds at 0 before game over
)

// dayConfig holds drain rates per in-game minute and bowl capacity.
type dayConfig struct {
	Hunger  float64
	Thirst  float64
	Fun     float64
	Energy  float64
	Hygiene float64
	Social  float64
	Coat    float64
	Claws   float64
	BowlCap int // visits before bowl/litter needs refilling
}

var dayConfigs = [MaxDays]dayConfig{
	{0.0018, 0.0016, 0.0014, 0.0009, 0.0011, 0.0011, 0.0010, 0.0009, 4},
	{0.0022, 0.0019, 0.0017, 0.0011, 0.0013, 0.0013, 0.0012, 0.0011, 4},
	{0.0026, 0.0022, 0.0019, 0.0013, 0.0016, 0.0016, 0.0014, 0.0013, 3},
	{0.0030, 0.0026, 0.0022, 0.0015, 0.0019, 0.0019, 0.0016, 0.0015, 3},
	{0.0034, 0.0029, 0.0025, 0.0017, 0.0022, 0.0022, 0.0019, 0.0017, 3},
}

// Scoring
const (
	ScoreTickLow     = 10
	ScoreTickHigh    = 25
	ScoreDayClean    = 300
	ScoreDayBonus    = 500
	ScorePet         = 5
	ScoreRefill      = 20
	ScoreCleanLitter = 20
)

// Day-of-week names
var dayNames = [MaxDays]string{
	"Monday", "Tuesday", "Wednesday", "Thursday", "Friday",
}

// Scoring
const ScoreBrush = 15

// Need names and icons
var needNames = [8]string{"Hunger", "Thirst", "Fun", "Energy", "Hygiene", "Social", "Coat", "Claws"}
var needIcons = [8]string{"🍽", "💧", "😄", "💤", "🚿", "🤝", "🪮", "🐾"}

// Alert messages per need
var alertMessages = [8]string{
	"Your cat is hungry! Fill the food bowl 🍽",
	"Your cat is thirsty! Top up the water bowl 💧",
	"Your cat is bored! Play with the toy wand 🧸",
	"Your cat is tired! Let it rest on the bed 😴",
	"The litter box needs cleaning! 🚿",
	"Your cat wants attention! Brush or pet it 🤝",
	"Your cat's fur is matted! Time for brushing 🪮",
	"Your cat needs to scratch! Use the scratcher 🐾",
}

// Colours
const (
	colRoom    = "#f5e6c8"
	colFloor   = "#c9a96e"
	colWall    = "#f0d8a8"
	colSky1    = "#87ceeb" // day blue
	colSkyNight = "#1a1a4e" // night

	// Ragdoll cat — seal colorpoint
	colCatBody   = "#f5eedd" // cream/ivory body
	colCatChest  = "#faf6ee" // lighter chest & belly
	colCatPoint  = "#7a4828" // seal dark points: ears, mask, paws, tail tip
	colCatPointL = "#a06840" // mid-tone for transitions
	colCatInner  = "#e8a0b0" // inner ear pink
	colCatEye    = "#2288ff" // bright ragdoll blue eyes
	colCatPupil  = "#112244" // dark pupil
	colCatNose   = "#d06878" // pink nose

	// Legacy aliases kept for non-cat drawing code
	colCat     = colCatBody
	colCatDark = colCatPoint
)
