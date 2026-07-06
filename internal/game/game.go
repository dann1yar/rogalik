// Package game implements a small top-down pixel-art medieval roguelike.
// It is intentionally dependency-light: Ebiten for rendering/input, and an
// optional best-effort telemetry client that talks to cmd/server.
package game

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	TileSize   = 16
	MapWidth   = 32
	MapHeight  = 20
	ScreenW    = MapWidth * TileSize
	ScreenH    = MapHeight*TileSize + 32 // + HUD strip
)

type tile int

const (
	tileWall tile = iota
	tileFloor
	tileStairs
)

type itemKind int

const (
	itemNone itemKind = iota
	itemSword
	itemShield
	itemPotion
)

type entity struct {
	x, y   int
	hp     int
	maxHP  int
	atk    int
	def    int
	alive  bool
	name   string
	glyph  rune
	colorK color.RGBA
}

// Game holds all mutable state for a single play session.
type Game struct {
	rng       *rand.Rand
	level     int
	tiles     [MapHeight][MapWidth]tile
	items     map[[2]int]itemKind
	player    entity
	enemies   []*entity
	turn      int
	message   string
	gameOver  bool
	telemetry *Telemetry
}

// New creates a fresh game at dungeon level 1.
func New(t *Telemetry) *Game {
	g := &Game{
		rng:       rand.New(rand.NewSource(rand.Int63())),
		telemetry: t,
	}
	g.player = entity{hp: 20, maxHP: 20, atk: 3, def: 1, alive: true, name: "Knight", glyph: '@'}
	g.generateLevel()
	g.telemetry.Event("game_started", g.level)
	return g
}

// --- Dungeon generation -----------------------------------------------

type room struct{ x, y, w, h int }

func (r room) center() (int, int) { return r.x + r.w/2, r.y + r.h/2 }

func (g *Game) generateLevel() {
	for y := 0; y < MapHeight; y++ {
		for x := 0; x < MapWidth; x++ {
			g.tiles[y][x] = tileWall
		}
	}
	g.items = map[[2]int]itemKind{}

	numRooms := 6 + g.rng.Intn(3)
	var rooms []room
	for i := 0; i < numRooms; i++ {
		w := 3 + g.rng.Intn(4)
		h := 3 + g.rng.Intn(3)
		x := 1 + g.rng.Intn(MapWidth-w-2)
		y := 1 + g.rng.Intn(MapHeight-h-2)
		r := room{x, y, w, h}
		g.carveRoom(r)
		if len(rooms) > 0 {
			px, py := rooms[len(rooms)-1].center()
			cx, cy := r.center()
			g.carveCorridor(px, py, cx, cy)
		}
		rooms = append(rooms, r)
	}

	// Place player in the first room.
	if len(rooms) > 0 {
		px, py := rooms[0].center()
		g.player.x, g.player.y = px, py
	}

	// Stairs in the last room.
	if len(rooms) > 0 {
		sx, sy := rooms[len(rooms)-1].center()
		g.tiles[sy][sx] = tileStairs
	}

	// Scatter loot and enemies in the remaining rooms.
	g.enemies = nil
	for i, r := range rooms {
		if i == 0 {
			continue
		}
		if g.rng.Intn(100) < 55 {
			ex, ey := r.center()
			ex += g.rng.Intn(3) - 1
			ey += g.rng.Intn(3) - 1
			g.enemies = append(g.enemies, g.spawnEnemy(ex, ey))
		}
		if g.rng.Intn(100) < 35 {
			ix, iy := r.center()
			ix -= g.rng.Intn(2)
			kinds := []itemKind{itemSword, itemShield, itemPotion}
			g.items[[2]int{ix, iy}] = kinds[g.rng.Intn(len(kinds))]
		}
	}
}

func (g *Game) carveRoom(r room) {
	for y := r.y; y < r.y+r.h; y++ {
		for x := r.x; x < r.x+r.w; x++ {
			if x >= 0 && x < MapWidth && y >= 0 && y < MapHeight {
				g.tiles[y][x] = tileFloor
			}
		}
	}
}

func (g *Game) carveCorridor(x1, y1, x2, y2 int) {
	x, y := x1, y1
	for x != x2 {
		g.tiles[y][x] = tileFloor
		if x < x2 {
			x++
		} else {
			x--
		}
	}
	for y != y2 {
		g.tiles[y][x] = tileFloor
		if y < y2 {
			y++
		} else {
			y--
		}
	}
	g.tiles[y][x] = tileFloor
}

func (g *Game) spawnEnemy(x, y int) *entity {
	hp := 4 + g.level*2
	return &entity{
		x: x, y: y, hp: hp, maxHP: hp,
		atk: 1 + g.level, def: g.level / 2,
		alive: true, name: "Goblin", glyph: 'g',
		colorK: color.RGBA{80, 140, 70, 255},
	}
}

// --- Update loop --------------------------------------------------------

func (g *Game) Update() error {
	if g.gameOver {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			*g = *New(g.telemetry)
		}
		return nil
	}

	dx, dy := 0, 0
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyUp), inpututil.IsKeyJustPressed(ebiten.KeyW):
		dy = -1
	case inpututil.IsKeyJustPressed(ebiten.KeyDown), inpututil.IsKeyJustPressed(ebiten.KeyS):
		dy = 1
	case inpututil.IsKeyJustPressed(ebiten.KeyLeft), inpututil.IsKeyJustPressed(ebiten.KeyA):
		dx = -1
	case inpututil.IsKeyJustPressed(ebiten.KeyRight), inpututil.IsKeyJustPressed(ebiten.KeyD):
		dx = 1
	default:
		return nil
	}

	nx, ny := g.player.x+dx, g.player.y+dy
	if !g.inBounds(nx, ny) || g.tiles[ny][nx] == tileWall {
		return nil
	}

	// Bump attack.
	if target := g.enemyAt(nx, ny); target != nil {
		g.attack(&g.player, target)
		if !target.alive {
			g.message = fmt.Sprintf("Killed %s", target.name)
			g.telemetry.Event("enemy_killed", g.level)
			g.removeDeadEnemies()
		}
		g.turn++
		g.enemyTurn()
		return nil
	}

	g.player.x, g.player.y = nx, ny

	if kind, ok := g.items[[2]int{nx, ny}]; ok {
		g.applyItem(kind)
		delete(g.items, [2]int{nx, ny})
	}

	if g.tiles[ny][nx] == tileStairs {
		g.level++
		g.message = fmt.Sprintf("Descending to level %d", g.level)
		g.telemetry.Event("level_up", g.level)
		g.generateLevel()
		return nil
	}

	g.turn++
	g.enemyTurn()
	return nil
}

func (g *Game) enemyTurn() {
	for _, e := range g.enemies {
		if !e.alive {
			continue
		}
		distX := g.player.x - e.x
		distY := g.player.y - e.y
		if abs(distX)+abs(distY) <= 1 {
			g.attack(e, &g.player)
			if !g.player.alive {
				g.gameOver = true
				g.message = "You died. Press Enter"
				g.telemetry.Event("player_death", g.level)
			}
			continue
		}
		if abs(distX)+abs(distY) <= 6 {
			ex, ey := e.x, e.y
			if abs(distX) > abs(distY) {
				ex += sign(distX)
			} else {
				ey += sign(distY)
			}
			if g.inBounds(ex, ey) && g.tiles[ey][ex] != tileWall && g.enemyAt(ex, ey) == nil && !(ex == g.player.x && ey == g.player.y) {
				e.x, e.y = ex, ey
			}
		}
	}
}

func (g *Game) attack(a, d *entity) {
	dmg := a.atk - d.def
	if dmg < 1 {
		dmg = 1
	}
	d.hp -= dmg
	if d.hp <= 0 {
		d.hp = 0
		d.alive = false
	}
}

func (g *Game) applyItem(k itemKind) {
	switch k {
	case itemSword:
		g.player.atk++
		g.message = "Found a sword (+1 ATK)"
	case itemShield:
		g.player.def++
		g.message = "Found a shield (+1 DEF)"
	case itemPotion:
		g.player.hp += 6
		if g.player.hp > g.player.maxHP {
			g.player.hp = g.player.maxHP
		}
		g.message = "Found a potion (+6 HP)"
	}
}

func (g *Game) removeDeadEnemies() {
	alive := g.enemies[:0]
	for _, e := range g.enemies {
		if e.alive {
			alive = append(alive, e)
		}
	}
	g.enemies = alive
}

func (g *Game) enemyAt(x, y int) *entity {
	for _, e := range g.enemies {
		if e.alive && e.x == x && e.y == y {
			return e
		}
	}
	return nil
}

func (g *Game) inBounds(x, y int) bool {
	return x >= 0 && x < MapWidth && y >= 0 && y < MapHeight
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sign(v int) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}

// --- Rendering -----------------------------------------------------------

var (
	colFloor  = color.RGBA{58, 48, 40, 255}
	colWall   = color.RGBA{28, 24, 22, 255}
	colStairs = color.RGBA{200, 170, 60, 255}
	colPlayer = color.RGBA{210, 210, 220, 255}
	colSword  = color.RGBA{190, 190, 60, 255}
	colShield = color.RGBA{90, 130, 190, 255}
	colPotion = color.RGBA{200, 60, 90, 255}
)

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{10, 10, 14, 255})

	for y := 0; y < MapHeight; y++ {
		for x := 0; x < MapWidth; x++ {
			c := colFloor
			switch g.tiles[y][x] {
			case tileWall:
				c = colWall
			case tileStairs:
				c = colStairs
			}
			drawTile(screen, x, y, c)
		}
	}

	for pos, kind := range g.items {
		c := colSword
		switch kind {
		case itemShield:
			c = colShield
		case itemPotion:
			c = colPotion
		}
		drawGlyph(screen, pos[0], pos[1], c, 0.5)
	}

	for _, e := range g.enemies {
		if e.alive {
			drawGlyph(screen, e.x, e.y, e.colorK, 0.8)
		}
	}

	drawGlyph(screen, g.player.x, g.player.y, colPlayer, 0.9)

	hud := fmt.Sprintf("HP: %d/%d  ATK: %d  DEF: %d  Level: %d  Turn: %d",
		g.player.hp, g.player.maxHP, g.player.atk, g.player.def, g.level, g.turn)
	ebitenutil.DebugPrintAt(screen, hud, 4, ScreenH-28)
	if g.message != "" {
		ebitenutil.DebugPrintAt(screen, g.message, 4, ScreenH-14)
	}
	if g.gameOver {
		ebitenutil.DebugPrintAt(screen, "GAME OVER — press Enter to retry", ScreenW/2-140, ScreenH/2)
	}
}

func drawTile(screen *ebiten.Image, x, y int, c color.RGBA) {
	ebitenutil.DrawRect(screen, float64(x*TileSize), float64(y*TileSize), TileSize-1, TileSize-1, c)
}

func drawGlyph(screen *ebiten.Image, x, y int, c color.RGBA, scale float64) {
	size := float64(TileSize-2) * scale
	off := (TileSize - size) / 2
	ebitenutil.DrawRect(screen, float64(x*TileSize)+off, float64(y*TileSize)+off, size, size, c)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenW, ScreenH
}
