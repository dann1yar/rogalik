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
	sprFloor     *ebiten.Image
	sprFloorVar1 *ebiten.Image
	sprFloorVar2 *ebiten.Image
	sprWall      *ebiten.Image
	sprGate      *ebiten.Image
	sprPlayer    *ebiten.Image
	sprGoblin    *ebiten.Image
	sprSword     *ebiten.Image
	sprShield    *ebiten.Image
	sprPotion    *ebiten.Image
)

func init() {
	img, _, err := image.Decode(bytes.NewReader(tilemapPNG))
	if err != nil {
		panic(err)
	}
	tilemap = ebiten.NewImageFromImage(img)

	// заполнение спрайтов
	sprFloor = tileAt(0, 0) // пол
	sprFloorVar1 = tileAt(0, 1)
	sprFloorVar2 = tileAt(0, 2)
	sprWall = tileAt(0, 3)   // стена
	sprGate = tileAt(9, 3)   // ворота
	sprPlayer = tileAt(0, 8) // игрок
	sprGoblin = tileAt(0, 9) // гоблин(призрак)
	sprSword = tileAt(8, 8)  // меч
	sprShield = tileAt(5, 5) // щит
	sprPotion = tileAt(6, 9) // зелье
}
