package game

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/tilemap_packed.png
var tilemapPNG []byte

var tilemap *ebiten.Image

func init() {
	img, _, err := image.Decode(bytes.NewReader(tilemapPNG))
	if err != nil {
		panic(err)
	}
	tilemap = ebiten.NewImageFromImage(img)
}

// tileAt вырезает тайл 16х16 из атласа по колонке и строке (счёт с нуля)
func tileAt(col, row int) *ebiten.Image {
	r := image.Rect(col*16, row*16, col*16+16, row*16+16)
	return tilemap.SubImage(r).(*ebiten.Image)
}

var (
	sprFloor  *ebiten.Image
	sprWall   *ebiten.Image
	sprStairs *ebiten.Image
	sprPlayer *ebiten.Image
	sprGoblin *ebiten.Image
	sprSword  *ebiten.Image
	sprShield *ebiten.Image
	sprPotion *ebiten.Image
)

func init() {
	img, _, err := image.Decode(bytes.NewReader(tilemapPNG))
	if err != nil {
		panic(err)
	}
	tilemap = ebiten.NewImageFromImage(img)

	// заполнение спрайтов
	sprFloor = tileAt(0, 0)
	sprWall = tileAt(0, 3)
	sprStairs = tileAt(2, 0)
	sprPlayer = tileAt(0, 8)
	sprGoblin = tileAt(0, 9)
	sprSword = tileAt(7, 8)
	sprShield = tileAt(8, 8)
	sprPotion = tileAt(9, 9)
}