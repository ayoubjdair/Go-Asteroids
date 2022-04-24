package main

// Game/GoLang Imports
import (
	"fmt"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Game Mode Type include -> [Playing, Paused, Lost, Win, Menus]
type Mode int

// Game Constants
const (

	// Different Game Levels
	ModeStart  Mode = 0
	ModeLevels Mode = 1
	ModePlay   Mode = 3
	ModePause  Mode = 4
	ModeOver   Mode = 5
	ModeWon    Mode = 6

	// Game Window Size
	windowWidth  = 800
	windowHeight = 600

	// Game Assets Sizes
	shipWidth          = 50
	shipHeight         = 80
	asteroidWidth      = 100
	asteroidHeight     = 80
	miniAsteroidWidth  = 50
	miniAsteroidHeight = 40

	// Asteroid Spin/Rotation Speed/Angle
	maxAngle      = 256
	maxDifficulty = 40
)

//Global Variables
var (

	// Count of asteroids present in game
	AsteroidsInGame     int
	miniAsteroidsInGame int

	// Minimum amount of asteroids in a game
	MinDifficulty int

	// Mutex for use of accessing cirital sections throughout the game
	mu sync.Mutex

	// Variables for recording how many goroutines are generated
	generationGoroutines uint32
	updateGoroutines     uint32
)

// Game Object Type
type Game struct {

	// Game Objects
	ship                 *ebiten.Image
	rocket               *ebiten.Image
	gameLogo             *ebiten.Image
	gameInstructions     *ebiten.Image
	gameLevels           *ebiten.Image
	gameOver             *ebiten.Image
	gamePaused           *ebiten.Image
	gameWon              *ebiten.Image
	gameConcurrencyRadar *ebiten.Image
	gamePlayerHealth     *ebiten.Image

	asteroidImage     *ebiten.Image
	miniAsteroidImage *ebiten.Image

	// Game Object Coordinates
	asteroidXPos, asteroidYPos float64
	shipXPos, shipYPos         float64
	rocketXPos, rocketYPos     float64

	mode         Mode
	shooting     bool
	playerHealth int

	asteroids     Asteroids
	miniAsteroids Asteroids
	drawOps       ebiten.DrawImageOptions
	inited        bool
	stars         [1024]Star
	MinDifficulty int
}

// Asteroid Object Type

type Asteroid struct {
	width  int
	height int
	x      float64
	y      float64
	vx     float64
	vy     float64
	angle  float64
}

// Asteroids type containts list of tpe Asteroid
type Asteroids struct {
	asteroidsList []*Asteroid
}

// Start type for live background
type Star struct {
	fromx, fromy, tox, toy, brightness float64
}

// Game initialisation function
func (g *Game) init(difficulty int) {

	defer func() {
		g.inited = true
	}()

	g.playerHealth = 100
	MinDifficulty = difficulty

	g.asteroids.asteroidsList = make([]*Asteroid, MinDifficulty, maxDifficulty)
	g.miniAsteroids.asteroidsList = make([]*Asteroid, maxDifficulty)

	AsteroidsInGame = len(g.asteroids.asteroidsList)
	miniAsteroidsInGame = 0
	generationGoroutines = 0
	updateGoroutines = 0

	g.shipXPos = float64(windowWidth/2) - float64(shipWidth/2)
	g.shipYPos = float64(windowHeight) - float64(shipHeight*2)

	g.rocketXPos = g.shipXPos + float64(shipWidth/2) - 1.5
	g.rocketYPos = g.shipYPos + float64(shipHeight/2)

	generateAsteroids(g)
}

// Generate Large Asteroid using Go Routines
func generateAsteroids(g *Game) {

	var wg sync.WaitGroup

	for i := range g.asteroids.asteroidsList {

		wg.Add(1)
		go func(i int) {
			atomic.AddUint32(&generationGoroutines, 1)
			w := asteroidWidth
			h := asteroidHeight
			x, y := rand.Intn(windowWidth-w), rand.Intn((windowHeight-h)/2)
			vx, vy := 2*rand.Intn(2)-1, 2*rand.Intn(2)-1
			a := rand.Intn(maxAngle)
			g.asteroids.asteroidsList[i] = &Asteroid{
				width:  w,
				height: h,
				x:      float64(x),
				y:      float64(y),
				vx:     float64(vx),
				vy:     float64(vy),
				angle:  float64(a),
			}
			fmt.Printf("Generation Go routine %d finished \n", i)
			wg.Done()
		}(i)
	}

	wg.Wait()
	fmt.Printf("%d Asteroids Generated concurrently \n", len(g.asteroids.asteroidsList))

}

// Concurrent Update function for Asteroids in game
func (s *Asteroids) Update() {

	var wg sync.WaitGroup

	for i := 0; i < AsteroidsInGame; i++ {

		wg.Add(1)
		go func(i int) {
			atomic.AddUint32(&updateGoroutines, 1)
			s.asteroidsList[i].Update()
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func (s *Asteroids) miniUpdate() {

	var wg sync.WaitGroup

	for i := 0; i < miniAsteroidsInGame; i++ {

		wg.Add(1)
		go func(i int) {
			atomic.AddUint32(&updateGoroutines, 1)
			s.asteroidsList[i].Update()
			wg.Done()
		}(i)
	}

	wg.Wait()
}

// Update function for individual asteroids
func (s *Asteroid) Update() {

	s.x += s.vx
	s.y += s.vy

	if s.x < 0 {
		s.x = -s.x
		s.vx = -s.vx
	} else if mx := float64(windowWidth) - float64(s.width); mx <= s.x {
		s.x = 2*mx - s.x
		s.vx = -s.vx
	}

	if s.y < 0 {
		s.y = -s.y
		s.vy = -s.vy
	} else if my := float64(windowHeight) - float64(s.height); my <= s.y {
		s.y = 2*my - s.y
		s.vy = -s.vy
	}

	s.angle++

	if s.angle == maxAngle {
		s.angle = 0
	}
}

// Update function
func (g *Game) Update() error {

	switch g.mode {
	case ModeStart:
		for _, x := range inpututil.PressedKeys() {
			if x == ebiten.KeySpace {
				g.mode = ModeLevels
			} else if x == ebiten.KeyQ {
				fmt.Println("Thanks for playing!")
				os.Exit(1)
			}
		}
	case ModeLevels:
		for _, x := range inpututil.PressedKeys() {
			if x == ebiten.Key1 {
				if !g.inited {
					g.init(5)
					g.mode = ModePlay
				}
			} else if x == ebiten.Key2 {
				if !g.inited {
					g.init(10)
					g.mode = ModePlay
				}
			} else if x == ebiten.Key3 {
				if !g.inited {
					g.init(20)
					g.mode = ModePlay
				}
			}
		}
	case ModePlay:
		// capture user input using Ebiten input utils
		for _, x := range inpututil.PressedKeys() {
			if x == ebiten.KeyRight {
				g.shipXPos += 10
				g.rocketXPos += 10
			} else if x == ebiten.KeyLeft {
				g.shipXPos -= 10
				g.rocketXPos -= 10
			} else if x == ebiten.KeyDown {
				g.shipYPos += 10
				g.rocketYPos += 10
			} else if x == ebiten.KeyUp {
				g.shipYPos -= 4
				g.rocketYPos -= 4
			} else if x == ebiten.KeyRight {
				g.shipXPos += 10
				g.rocketXPos += 10
			} else if x == ebiten.KeyA {
				g.shipXPos -= 10
				g.rocketXPos -= 10
			} else if x == ebiten.KeyS {
				g.shipYPos += 10
				g.rocketYPos += 10
			} else if x == ebiten.KeyW {
				g.shipYPos -= 4
				g.rocketYPos -= 4
			} else if x == ebiten.KeyD {
				g.shipXPos += 10
				g.rocketXPos += 10
			} else if x == ebiten.KeySpace {
				g.shooting = true
			} else if x == ebiten.KeyP {
				g.mode = ModePause
			}
		}

		// Do not allow ship to fly out of bounds
		// - Don't allow ship to pass side boundaries
		if g.shipXPos >= float64(windowWidth)-float64(shipWidth) {
			g.shipXPos = float64(windowWidth) - float64(shipWidth)
			g.rocketXPos = g.shipXPos + float64(shipWidth/2) - 1.5
			g.rocketYPos = g.shipYPos + float64(shipHeight/2)
		}
		if g.shipXPos <= 0 {
			g.shipXPos = 0
			g.rocketXPos = g.shipXPos + float64(shipWidth/2) - 1.5
			g.rocketYPos = g.shipYPos + float64(shipHeight/2)
		}

		// - Don't allow ship to pass top-down boundaries
		if g.shipYPos <= 0 {
			g.shipYPos = 0
			g.rocketXPos = g.shipXPos + float64(shipWidth/2) - 1.5
			g.rocketYPos = g.shipYPos + float64(shipHeight/2)
		}
		if g.shipYPos >= float64(windowHeight)-float64(shipHeight) {
			g.shipYPos = float64(windowHeight) - float64(shipHeight)
			g.rocketXPos = g.shipXPos + float64(shipWidth/2) - 1.5
			g.rocketYPos = g.shipYPos + float64(shipHeight/2)
		}

		// shooting rocket
		if g.shooting {
			g.shootRocket()
		}
		if g.rocketYPos <= 0 {
			g.shooting = false
			g.rocketXPos = g.shipXPos + float64(shipWidth/2) - 1.5
			g.rocketYPos = g.shipYPos + float64(shipHeight/2)
		}
		if g.rocketYPos <= g.asteroidYPos+float64(asteroidHeight) && g.rocketXPos <= g.asteroidXPos+float64(asteroidHeight) && g.rocketXPos >= g.asteroidXPos {
			g.shooting = false
			g.rocketXPos = g.shipXPos + float64(shipWidth/2) - 1.5
			g.rocketYPos = g.shipYPos + float64(shipHeight/2)
		}

		// Check if rocket has hit an asteroid
		x, y := g.hit()
		if x != -1 || y != -1 {
			var wg sync.WaitGroup
			for i := miniAsteroidsInGame; i < miniAsteroidsInGame+2 && i < maxDifficulty; i++ {
				wg.Add(1)
				go func(i int) {
					atomic.AddUint32(&generationGoroutines, 1)
					w := miniAsteroidWidth
					h := miniAsteroidWidth
					x, y := x+rand.Float64()*(500-400), y+rand.Float64()*(500-400)
					vx, vy := 3*rand.Intn(2)-1, 2*rand.Intn(2)-1
					a := rand.Intn(maxAngle)
					g.miniAsteroids.asteroidsList[i] = &Asteroid{
						width:  w,
						height: h,
						x:      float64(x),
						y:      float64(y),
						vx:     float64(vx),
						vy:     float64(vy),
						angle:  float64(a),
					}
					miniAsteroidsInGame = miniAsteroidsInGame + 1
					fmt.Printf("Split off Go routine %d finished, new mini asteroid generated \n", i)
					wg.Done()
				}(i)
			}
			wg.Wait()
		}

		// Check if rocket has hit an asteroid
		g.miniHit()

		// Check for collission with asteroids
		g.collissonCheck()

		// Update asteroid trajectory/movement
		g.asteroids.Update()
		g.miniAsteroids.miniUpdate()

		// Check if player health remains above 0
		if g.playerHealth <= 0 {
			g.mode = ModeOver
		}

		// Check if player has blown up all asteroids
		if AsteroidsInGame == 0 && miniAsteroidsInGame == 0 {
			g.mode = ModeWon
		}

	case ModePause:
		for _, x := range inpututil.PressedKeys() {
			if x == ebiten.KeyR {
				g.mode = ModePlay
			} else if x == ebiten.KeyQ {
				os.Exit(1)
			} else if x == ebiten.KeyM {
				g.inited = false
				g.mode = ModeStart
			}
		}
	case ModeOver:
		g.inited = false
		for _, x := range inpututil.PressedKeys() {
			if x == ebiten.KeyR {
				g.miniAsteroids.asteroidsList = g.miniAsteroids.asteroidsList[:0]
				g.mode = ModeLevels
			} else if x == ebiten.KeyQ {
				fmt.Println("Thanks for playing!")
				os.Exit(1)
			}
		}
	case ModeWon:
		g.inited = false
		for _, x := range inpututil.PressedKeys() {
			if x == ebiten.KeyP {
				g.mode = ModeStart
			} else if x == ebiten.KeyQ {
				fmt.Println("Thanks for playing!")
				os.Exit(1)
			}
		}
	}
	return nil
}

// Moves rocket position when firing
func (g *Game) shootRocket() {
	// rocket position
	g.rocketYPos -= 15
}

// Checks if rocket has hit an asteroid
func (g *Game) hit() (float64, float64) {

	var x, y float64 = -1, -1
	w, h := g.rocket.Size()

	for i := 0; i < AsteroidsInGame; i++ {

		if g.rocketXPos < g.asteroids.asteroidsList[i].x+asteroidWidth &&
			g.rocketXPos+float64(w) > g.asteroids.asteroidsList[i].x &&
			g.rocketYPos < g.asteroids.asteroidsList[i].y+asteroidHeight &&
			g.rocketYPos+float64(h) > g.asteroids.asteroidsList[i].y {

			g.shooting = false
			g.rocketXPos = g.shipXPos + float64(shipWidth/2) - 1.5
			g.rocketYPos = g.shipYPos + float64(shipHeight/2)
			x = g.asteroids.asteroidsList[i].x
			y = g.asteroids.asteroidsList[i].y
			g.asteroids.asteroidsList = blowUp(g.asteroids.asteroidsList, i)
			AsteroidsInGame = AsteroidsInGame - 1

		}
	}

	return x, y
}

// Checks if rocket has hit a mini asteroid
func (g *Game) miniHit() {

	w, h := g.rocket.Size()
	for i := 0; i < miniAsteroidsInGame; i++ {

		if g.rocketXPos < g.miniAsteroids.asteroidsList[i].x+miniAsteroidWidth &&
			g.rocketXPos+float64(w) > g.miniAsteroids.asteroidsList[i].x &&
			g.rocketYPos < g.miniAsteroids.asteroidsList[i].y+miniAsteroidHeight &&
			g.rocketYPos+float64(h) > g.miniAsteroids.asteroidsList[i].y {

			g.shooting = false
			g.rocketXPos = g.shipXPos + float64(shipWidth/2) - 1.5
			g.rocketYPos = g.shipYPos + float64(shipHeight/2)
			g.miniAsteroids.asteroidsList = blowUp(g.miniAsteroids.asteroidsList, i)
			miniAsteroidsInGame = miniAsteroidsInGame - 1

		}
	}
}

// Function to Reduce Player Health
func reduce_health(g *Game, wg *sync.WaitGroup) {
	mu.Lock()
	g.playerHealth -= 1
	mu.Unlock()
	wg.Done()
}

// Check for collisions
func (g *Game) collissonCheck() {
	g.ship_hit_asteroid()
	g.ship_hit_mini_asteroid()
}

// Checks if ship has collided with a asteroid
func (g *Game) ship_hit_asteroid() {

	for i := 0; i < AsteroidsInGame; i++ {

		if g.shipXPos < g.asteroids.asteroidsList[i].x+asteroidWidth &&
			g.shipXPos+shipWidth > g.asteroids.asteroidsList[i].x &&
			g.shipYPos < g.asteroids.asteroidsList[i].y+asteroidHeight &&
			shipHeight+g.shipYPos > g.asteroids.asteroidsList[i].y {

			var wg sync.WaitGroup
			wg.Add(1)
			go reduce_health(g, &wg)
			wg.Wait()

		}
	}
}

// Checks if ship has collided with a mini asteroid
func (g *Game) ship_hit_mini_asteroid() {

	for i := 0; i < miniAsteroidsInGame; i++ {

		if g.shipXPos < g.miniAsteroids.asteroidsList[i].x+miniAsteroidWidth &&
			g.shipXPos+shipWidth > g.miniAsteroids.asteroidsList[i].x &&
			g.shipYPos < g.miniAsteroids.asteroidsList[i].y+miniAsteroidHeight &&
			shipHeight+g.shipYPos > g.miniAsteroids.asteroidsList[i].y {

			var wg sync.WaitGroup
			wg.Add(1)
			go reduce_health(g, &wg)
			wg.Wait()

		}
	}
}

// Removes the hit asteroid from the list of asteroids
func blowUp(asteroids []*Asteroid, index int) []*Asteroid {
	return append(asteroids[:index], asteroids[index+1:]...)
}

// Drawing functions - to render images on screen
func (g *Game) Draw(screen *ebiten.Image) {

	screen.Fill(color.Black)
	g.drawStars(screen)

	if g.mode == ModePlay {
		g.drawConcurrencyRadar(screen)
		g.drawShip(screen)
		g.drawAstroids(screen)
		g.drawMiniAstroids(screen)
		g.drawRocket(screen)
		updateStars(g, g.shipXPos, g.shipYPos)
	}

	if g.mode == ModeStart {
		g.drawStartScreen(screen)
		updateStars(g, float64(windowWidth/2), float64(windowHeight/2))
	}

	if g.mode == ModeLevels {
		g.drawLevels(screen)
		updateStars(g, float64(windowWidth), float64(windowHeight/2))
	}

	if g.mode == ModePause {
		g.drawGamePausedScreen(screen)
	}

	if g.mode == ModeOver {
		g.drawGameOverScreen(screen)
	}

	if g.mode == ModeWon {
		g.drawGameWonScreen(screen)
	}
}

func (g *Game) drawConcurrencyRadar(screen *ebiten.Image) {

	drawOptions := &ebiten.DrawImageOptions{}

	drawOptions.GeoM.Translate(0, 10)
	screen.DrawImage(g.gameConcurrencyRadar, drawOptions)

	drawOptions.GeoM.Translate(0, 550)
	screen.DrawImage(g.gamePlayerHealth, drawOptions)

	health := fmt.Sprintf("%d", g.playerHealth)
	asteroids := fmt.Sprintf("Number of Asteroids (Go Routines): %d", AsteroidsInGame)
	minAsteroids := fmt.Sprintf("Number of Mini-Asteroids (Sub Go Routines): %d", miniAsteroidsInGame)
	genThreads := fmt.Sprintf("Go routines used to generate Asteroids: %d", generationGoroutines)
	updateThreads := fmt.Sprintf("Go routines used to update Asteroids: %d", updateGoroutines)

	ebitenutil.DebugPrintAt(screen, health, 210, 572)
	ebitenutil.DebugPrintAt(screen, asteroids, 30, 50)
	ebitenutil.DebugPrintAt(screen, minAsteroids, 30, 70)
	ebitenutil.DebugPrintAt(screen, genThreads, 30, 90)
	ebitenutil.DebugPrintAt(screen, updateThreads, 30, 110)

}

func (g *Game) drawShip(screen *ebiten.Image) {
	drawOptions := &ebiten.DrawImageOptions{}
	drawOptions.GeoM.Translate(g.shipXPos, g.shipYPos)
	screen.DrawImage(g.ship, drawOptions)
}

func (g *Game) drawRocket(screen *ebiten.Image) {
	drawOptions3 := &ebiten.DrawImageOptions{}
	drawOptions3.GeoM.Translate(g.rocketXPos, g.rocketYPos)
	screen.DrawImage(g.rocket, drawOptions3)
}

func (g *Game) drawStartScreen(screen *ebiten.Image) {
	g.drawLogo(screen)
	g.drawInstructions(screen)
}

func (g *Game) drawLogo(screen *ebiten.Image) {
	drawOptions := &ebiten.DrawImageOptions{}
	x, y := g.gameLogo.Size()
	drawOptions.GeoM.Translate((windowWidth/2)-float64(x/2), (windowHeight/2)-float64(y))
	screen.DrawImage(g.gameLogo, drawOptions)
}

func (g *Game) drawInstructions(screen *ebiten.Image) {
	drawOptions := &ebiten.DrawImageOptions{}
	x, y := g.gameInstructions.Size()
	drawOptions.GeoM.Translate((windowWidth/2)-float64(x/2), (windowHeight/2)-float64(y/8))
	screen.DrawImage(g.gameInstructions, drawOptions)
}

func (g *Game) drawLevels(screen *ebiten.Image) {
	drawOptions := &ebiten.DrawImageOptions{}
	x, y := g.gameLevels.Size()
	drawOptions.GeoM.Translate((windowWidth/2)-float64(x/2), (windowHeight/2)-float64(y/2))
	screen.DrawImage(g.gameLevels, drawOptions)
}

func (g *Game) drawGameOverScreen(screen *ebiten.Image) {
	drawOptions := &ebiten.DrawImageOptions{}
	x, y := g.gameOver.Size()
	drawOptions.GeoM.Translate((windowWidth/2)-float64(x/2), (windowHeight/2)-float64(y/2))
	screen.DrawImage(g.gameOver, drawOptions)
}

func (g *Game) drawGamePausedScreen(screen *ebiten.Image) {
	drawOptions := &ebiten.DrawImageOptions{}
	x, y := g.gamePaused.Size()
	drawOptions.GeoM.Translate((windowWidth/2)-float64(x/2), (windowHeight/2)-float64(y/2))
	screen.DrawImage(g.gamePaused, drawOptions)
}

func (g *Game) drawGameWonScreen(screen *ebiten.Image) {
	drawOptions := &ebiten.DrawImageOptions{}
	x, y := g.gameWon.Size()
	drawOptions.GeoM.Translate((windowWidth/2)-float64(x/2), (windowHeight/2)-float64(y/2))
	screen.DrawImage(g.gameWon, drawOptions)
}

func (g *Game) drawAstroids(screen *ebiten.Image) {

	w, h := g.asteroidImage.Size()

	for i := 0; i < AsteroidsInGame; i++ {

		s := g.asteroids.asteroidsList[i]
		g.drawOps.GeoM.Reset()
		g.drawOps.GeoM.Translate(-float64(w)/2, -float64(h)/2)
		g.drawOps.GeoM.Rotate(2 * math.Pi * float64(s.angle) / maxAngle)
		g.drawOps.GeoM.Translate(float64(w)/2, float64(h)/2)
		g.drawOps.GeoM.Translate(float64(s.x), float64(s.y))
		screen.DrawImage(g.asteroidImage, &g.drawOps)

	}
}

func (g *Game) drawMiniAstroids(screen *ebiten.Image) {

	for i := 0; i < miniAsteroidsInGame; i++ {

		s := g.miniAsteroids.asteroidsList[i]
		g.drawOps.GeoM.Reset()
		g.drawOps.GeoM.Translate(-float64(miniAsteroidWidth)/2, -float64(miniAsteroidHeight)/2)
		g.drawOps.GeoM.Rotate(2 * math.Pi * float64(s.angle) / maxAngle)
		g.drawOps.GeoM.Translate(float64(miniAsteroidWidth)/2, float64(miniAsteroidHeight)/2)
		g.drawOps.GeoM.Translate(float64(s.x), float64(s.y))
		screen.DrawImage(g.miniAsteroidImage, &g.drawOps)

	}
}

// Stars Background Functions
// Initialise stars
func (s *Star) Init() {
	s.tox = rand.Float64() * windowWidth * 64
	s.fromx = s.tox
	s.toy = rand.Float64() * windowHeight * 64
	s.fromy = s.toy
	s.brightness = rand.Float64() * 0xff
}

// Update stars
func (s *Star) Update(x, y float64) {
	s.fromx = s.tox
	s.fromy = s.toy
	s.tox += (s.tox - x) / 32
	s.toy += (s.toy - y) / 32
	s.brightness += 1
	if 0xff < s.brightness {
		s.brightness = 0xff
	}
	if s.fromx < 0 || windowWidth*64 < s.fromx || s.fromy < 0 || windowHeight*64 < s.fromy {
		s.Init()
	}
}

// Calls .Draw() for 1024 stars
func (g *Game) drawStars(screen *ebiten.Image) {
	for i := 0; i < 1024; i++ {
		g.stars[i].Draw(screen)
	}
}

// Draws each individual star
func (s *Star) Draw(screen *ebiten.Image) {
	c := color.RGBA{R: uint8(0xbb * s.brightness / 0xff),
		G: uint8(0xdd * s.brightness / 0xff),
		B: uint8(0xff * s.brightness / 0xff),
		A: 0xff}
	ebitenutil.DrawLine(screen, s.fromx/64, s.fromy/64, s.tox/64, s.toy/64, c)
}

// Updates poisition for each star
func updateStars(g *Game, x, y float64) {
	for i := 0; i < 1024; i++ {
		g.stars[i].Update(float64(x*64), float64(y*64))
	}
}

// returns display layout
func (g *Game) Layout(outsideWidth, outsideHeight int) (windowWidth, windowHeight int) {
	return 800, 600
}

func loadAssets(g *Game) {
	shipIcon, _, err := ebitenutil.NewImageFromFile("GUI/GameAssets/ship.png")
	if err != nil {
		log.Fatalf("Error Loading Ship Icon: %v", err)
	}

	logo, _, err := ebitenutil.NewImageFromFile("GUI/GameScreens/gameLogo.png")
	if err != nil {
		log.Fatalf("Error Loading Go Asteroids Logo: %v", err)
	}

	instructions, _, err := ebitenutil.NewImageFromFile("GUI/GameScreens/gameInstructions.png")
	if err != nil {
		log.Fatalf("Error Loading Go Asteroids Instructions: %v", err)
	}

	levels, _, err := ebitenutil.NewImageFromFile("GUI/GameScreens/gameLevels.png")
	if err != nil {
		log.Fatalf("Error Loading Go Asteroids Levels: %v", err)
	}

	gameOvers, _, err := ebitenutil.NewImageFromFile("GUI/GameScreens/gameOver.png")
	if err != nil {
		log.Fatalf("Error Loading Go Asteroids Game Over Screen: %v", err)
	}

	gamePaused, _, err := ebitenutil.NewImageFromFile("GUI/GameScreens/gamePaused.png")
	if err != nil {
		log.Fatalf("Error Loading Go Asteroids Game Paused Screen: %v", err)
	}

	gameWon, _, err := ebitenutil.NewImageFromFile("GUI/GameScreens/gameWon.png")
	if err != nil {
		log.Fatalf("Error Loading Go Asteroids Won Screen: %v", err)
	}

	asteroidIcon, _, err := ebitenutil.NewImageFromFile("GUI/GameAssets/asteroid.png")
	if err != nil {
		log.Fatalf("Error Loading Asteroid Icon: %v", err)
	}

	miniAsteroidIcon, _, err := ebitenutil.NewImageFromFile("GUI/GameAssets/miniAsteroid.png")
	if err != nil {
		log.Fatalf("Error Loading Mini-Asteroid Icon: %v", err)
	}

	radarLogo, _, err := ebitenutil.NewImageFromFile("GUI/GameScreens/gameConcurrencyRadar.png")
	if err != nil {
		log.Fatalf("Error Loading Concurrency Radar Logo: %v", err)
	}

	HealthBackground, _, err := ebitenutil.NewImageFromFile("GUI/GameScreens/gamePlayerHealth.png")
	if err != nil {
		log.Fatalf("Error Loading Player Health Icon: %v", err)
	}

	g.ship = shipIcon
	g.asteroidImage = asteroidIcon
	g.miniAsteroidImage = miniAsteroidIcon

	g.gameLogo = logo
	g.gameInstructions = instructions
	g.gameConcurrencyRadar = radarLogo
	g.gamePlayerHealth = HealthBackground

	g.gameOver = gameOvers
	g.gamePaused = gamePaused
	g.gameWon = gameWon
	g.gameLevels = levels

	rocketIcon := ebiten.NewImage(2, 10)
	rocketIcon.Fill(color.White)
	g.rocket = rocketIcon

}

// Main Function
func main() {
	ebiten.SetWindowSize(windowWidth, windowHeight)
	ebiten.SetWindowTitle("Go Asteroids")

	fmt.Println("Welcome To Go Asteroids")
	fmt.Println("Go Routines will be printed here")

	g := &Game{}
	loadAssets(g)

	g.mode = ModeStart
	g.shooting = false
	if err := ebiten.RunGame(g); err != nil {
		log.Fatalf("Error Running Game: %v", err)
	}

}
