package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 640
	screenHeight = 480

	ballRadius     = 8
	initialPaddleW = 96
	paddleHeight   = 16

	brickRows = 5
	brickCols = 10
	brickW    = 56
	brickH    = 24
	brickGap  = 4
)

type Brick struct {
	Visible bool
	Color   color.RGBA
}

type Game struct {
	ballX, ballY   float64
	ballVX, ballVY float64

	paddleX float64
	paddleW float64

	bricks [][]Brick

	score  int
	lives  int
	level  int
	status string // "playing", "gameover", "win"

	rng *rand.Rand

	ballImage *ebiten.Image
}

func NewGame() *Game {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	g := &Game{
		ballX:   screenWidth / 2,
		ballY:   screenHeight / 2,
		ballVX:  3,
		ballVY:  -3,
		paddleX: screenWidth/2 - initialPaddleW/2,
		paddleW: initialPaddleW,
		lives:   3,
		level:   1,
		status:  "playing",
		rng:     rng,
	}

	// Create ball image once (circle)
	g.ballImage = ebiten.NewImage(ballRadius*2, ballRadius*2)
	drawCircle(g.ballImage, ballRadius, ballRadius, ballRadius, color.White)

	g.resetLevel()
	return g
}

func (g *Game) resetLevel() {
	g.bricks = make([][]Brick, brickRows)
	for i := range g.bricks {
		g.bricks[i] = make([]Brick, brickCols)
		for j := range g.bricks[i] {
			visible := g.rng.Float64() > 0.2 || g.level == 1
			g.bricks[i][j] = Brick{
				Visible: visible,
				Color: color.RGBA{
					R: uint8(g.rng.Intn(256)),
					G: uint8(g.rng.Intn(256)),
					B: uint8(g.rng.Intn(256)),
					A: 255,
				},
			}
		}
	}

	// Reset ball and paddle position
	g.ballX = screenWidth / 2
	g.ballY = screenHeight / 2
	speedIncrease := 1 + float64(g.level-1)*0.2
	g.ballVX = 3 * speedIncrease
	g.ballVY = -3 * speedIncrease
	g.paddleW = initialPaddleW

	// Shrink paddle every other level (min 3x ball diameter)
	minPaddleW := ballRadius * 2 * 3
	for l := 2; l <= g.level; l += 2 {
		if g.paddleW > float64(minPaddleW) {
			g.paddleW -= 8
		}
	}

	g.paddleX = screenWidth/2 - g.paddleW/2
	g.status = "playing"
}

func (g *Game) Update() error {
	if g.status == "gameover" || g.status == "win" {
		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			if g.status == "win" {
				g.level++
			} else {
				g.level = 1
				g.score = 0
				g.lives = 3
			}
			g.resetLevel()
		}
		return nil
	}

	// Paddle movement
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.paddleX -= 5
		if g.paddleX < 0 {
			g.paddleX = 0
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.paddleX += 5
		if g.paddleX+g.paddleW > screenWidth {
			g.paddleX = screenWidth - g.paddleW
		}
	}

	// Move ball
	g.ballX += g.ballVX
	g.ballY += g.ballVY

	// Bounce off walls
	if g.ballX-ballRadius < 0 {
		g.ballX = ballRadius
		g.ballVX = -g.ballVX
	}
	if g.ballX+ballRadius > screenWidth {
		g.ballX = screenWidth - ballRadius
		g.ballVX = -g.ballVX
	}
	if g.ballY-ballRadius < 0 {
		g.ballY = ballRadius
		g.ballVY = -g.ballVY
	}

	// Ball hits paddle
	if g.ballY+ballRadius > float64(screenHeight-paddleHeight) &&
		g.ballX > g.paddleX && g.ballX < g.paddleX+g.paddleW {
		g.ballY = float64(screenHeight-paddleHeight) - ballRadius
		g.ballVY = -g.ballVY

		// Add a bit of horizontal velocity based on where it hits the paddle
		paddleCenter := g.paddleX + g.paddleW/2
		delta := g.ballX - paddleCenter
		g.ballVX += delta * 0.1
	}

	// Ball missed paddle
	if g.ballY-ballRadius > screenHeight {
		g.lives--
		if g.lives <= 0 {
			g.status = "gameover"
		} else {
			// Reset ball and paddle position for next life
			g.ballX = screenWidth / 2
			g.ballY = screenHeight / 2
			g.ballVX = 3 + float64(g.level-1)*0.2
			g.ballVY = -3 - float64(g.level-1)*0.2
			g.paddleX = screenWidth/2 - g.paddleW/2
		}
	}

	// Ball hits bricks
	for i := range g.bricks {
		for j := range g.bricks[i] {
			b := &g.bricks[i][j]
			if !b.Visible {
				continue
			}
			brickX := float64(j*(brickW+brickGap) + brickGap)
			brickY := float64(i*(brickH+brickGap) + brickGap)

			if g.ballX+ballRadius > brickX &&
				g.ballX-ballRadius < brickX+brickW &&
				g.ballY+ballRadius > brickY &&
				g.ballY-ballRadius < brickY+brickH {

				b.Visible = false
				g.ballVY = -g.ballVY
				g.score += 10
			}
		}
	}

	// Check win condition (all bricks cleared)
	allCleared := true
	for i := range g.bricks {
		for j := range g.bricks[i] {
			if g.bricks[i][j].Visible {
				allCleared = false
				break
			}
		}
		if !allCleared {
			break
		}
	}
	if allCleared {
		g.status = "win"
	}

	return nil
}

// drawCircle draws a filled circle onto the given ebiten.Image at (cx, cy) with radius r.
func drawCircle(img *ebiten.Image, cx, cy, r int, clr color.Color) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				img.Set(cx+x, cy+y, clr)
			}
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Clear screen
	screen.Fill(color.RGBA{0x10, 0x10, 0x10, 0xff})

	// Draw bricks
	for i := range g.bricks {
		for j := range g.bricks[i] {
			b := g.bricks[i][j]
			if b.Visible {
				x := float64(j*(brickW+brickGap) + brickGap)
				y := float64(i*(brickH+brickGap) + brickGap)
				ebitenutil.DrawRect(screen, x, y, brickW, brickH, b.Color)
			}
		}
	}

	// Draw paddle
	ebitenutil.DrawRect(screen, float64(g.paddleX), float64(screenHeight-paddleHeight), g.paddleW, paddleHeight, color.White)

	// Draw ball using pre-rendered image
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.ballX-ballRadius, g.ballY-ballRadius)
	screen.DrawImage(g.ballImage, op)

	// Draw score, lives, level
	statusText := fmt.Sprintf("Score: %d  Lives: %d  Level: %d", g.score, g.lives, g.level)
	ebitenutil.DebugPrint(screen, statusText)

	// Draw game over or win message
	if g.status == "gameover" {
		msg := "GAME OVER - Press SPACE to Restart"
		x := (screenWidth - len(msg)*7) / 2
		y := screenHeight / 2
		ebitenutil.DebugPrintAt(screen, msg, x, y)
	} else if g.status == "win" {
		msg := "YOU WIN! - Press SPACE for Next Level"
		x := (screenWidth - len(msg)*7) / 2
		y := screenHeight / 2
		ebitenutil.DebugPrintAt(screen, msg, x, y)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Go Breakout")

	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
