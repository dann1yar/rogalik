// Command server is a small headless companion service for the game.
// It has zero graphics dependencies (no cgo, no Ebiten) so it builds as a
// static binary and runs happily in a scratch/distroless container. It
// exposes:
//
//	GET  /health   liveness/readiness probe
//	GET  /metrics  Prometheus metrics
//	POST /event    fire-and-forget events sent by the game client
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	gamesStarted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rogalik_games_started_total",
		Help: "Total number of game sessions started.",
	})
	enemiesKilled = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rogalik_enemies_killed_total",
		Help: "Total number of enemies killed across all sessions.",
	})
	playerDeaths = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rogalik_player_deaths_total",
		Help: "Total number of player deaths across all sessions.",
	})
	maxLevelReached = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rogalik_max_level_reached",
		Help: "Highest dungeon level reached by any session since startup.",
	})

	startedAt   = time.Now()
	eventsTotal int64
)

type eventPayload struct {
	Type  string `json:"type"`
	Level int    `json:"level"`
}

func handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var p eventPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	atomic.AddInt64(&eventsTotal, 1)

	switch p.Type {
	case "game_started":
		gamesStarted.Inc()
	case "enemy_killed":
		enemiesKilled.Inc()
	case "player_death":
		playerDeaths.Inc()
	}
	if p.Type == "level_up" || p.Type == "game_started" {
		maxLevelMu.Lock()
		if float64(p.Level) > maxLevelSeen {
			maxLevelSeen = float64(p.Level)
			maxLevelReached.Set(maxLevelSeen)
		}
		maxLevelMu.Unlock()
	}
	w.WriteHeader(http.StatusAccepted)
}

var (
	maxLevelMu   sync.Mutex
	maxLevelSeen float64
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"uptime": time.Since(startedAt).String(),
		"events": atomic.LoadInt64(&eventsTotal),
	})
}

func main() {
	addr := envDefault("ROGALIK_ADDR", ":8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/event", handleEvent)
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	log.Printf("rogalik telemetry server listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
