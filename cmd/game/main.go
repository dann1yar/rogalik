// Command game is the playable desktop client: a small top-down pixel-art
// medieval roguelike. Cross-platform binaries are produced by
// .github/workflows/release.yml (one native build per OS, since Ebiten
// needs cgo + platform graphics headers and can't be cross-compiled from
// a single Linux host).
package main

import (
	"flag"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/dann1yar/rogalik/internal/game"
)

func main() {
	telemetryURL := flag.String("telemetry", envDefault("PIXELKEEP_TELEMETRY_URL", ""), "base URL of the headless telemetry server, empty disables it")
	flag.Parse()

	ebiten.SetWindowSize(game.ScreenW*2, game.ScreenH*2)
	ebiten.SetWindowTitle("Pixel Keep — pixel-art roguelike")
	ebiten.SetWindowResizable(true)

	g := game.New(game.NewTelemetry(*telemetryURL))
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
