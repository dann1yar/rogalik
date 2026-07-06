# Rogalik

Небольшой top-down пиксельный рогалик на Go (движок [Ebiten](https://ebitengine.org/)),
собранный не просто «для поиграть», а как учебный DevOps-полигон: кросс-платформенная
сборка, контейнеризация, CI/CD-релизы и базовый мониторинг.

## Что внутри

```
cmd/game     — игровой клиент (Ebiten, нужен cgo + графические заголовки ОС)
cmd/server   — headless-сервис телеметрии: /health, /metrics, /event (без графики, CGO_ENABLED=0)
internal/game— логика игры: генерация подземелья, бой, инвентарь
deploy/      — конфиг Prometheus
Dockerfile   — multi-stage сборка cmd/server → distroless-образ
docker-compose.yml — сервер + Prometheus + Grafana для локальной демонстрации
.github/workflows/release.yml — CI: matrix-сборка бинарников под 3 ОС, сборка и
                                  публикация Docker-образа, релиз на GitHub по тегу
```

## Игровой процесс

Рыцарь (`@`) спускается по процедурно генерируемым подземельям, сражается с гоблинами
(бамп-атака — просто иди на врага), подбирает меч/щит/зелье и ищет лестницу (`>`), чтобы
спуститься глубже. Управление: стрелки или WASD, Enter — новая попытка после смерти.

```bash
go run ./cmd/game
```

## Почему архитектура разделена на два бинарника

Ebiten требует cgo и нативные графические заголовки (X11/OpenGL на Linux, аналоги на
Windows/macOS) — такое нельзя честно кросс-компилировать из одного контейнера. Поэтому:

- **`cmd/game`** собирается нативно под каждую ОС в CI-матрице GitHub Actions
  (`ubuntu-latest`, `windows-latest`, `macos-latest`) — три параллельных джоба, три
  готовых бинарника в артефактах.
- **`cmd/server`** — обычный HTTP-сервис без графики, поэтому его можно честно собрать
  статически (`CGO_ENABLED=0`) и упаковать в multi-stage `Dockerfile` поверх
  `distroless`, получив минимальный контейнер без шелла и лишних библиотек.

Это реалистичный паттерн: «толстый» клиент собирается нативно под платформы
пользователей, а «тонкий» сервисный компонент — контейнеризируется и деплоится как
обычный микросервис.

## Локальный запуск стека мониторинга

```bash
docker compose up --build
```

- `http://localhost:8080/health` — health-check
- `http://localhost:8080/metrics` — метрики Prometheus (`rogalik_games_started_total`,
  `rogalik_enemies_killed_total`, `rogalik_player_deaths_total`,
  `rogalik_max_level_reached`)
- `http://localhost:9090` — Prometheus (скрейпит сервер каждые 5с)
- `http://localhost:3000` — Grafana (анонимный доступ включён для демо)

Чтобы игра слала события на сервер:

```bash
go run ./cmd/game -telemetry http://localhost:8080
```

Это best-effort и неблокирующий вызов: если сервер недоступен, игра просто продолжает
работать — телеметрия никогда не влияет на игровой процесс.

## CI/CD пайплайн (`.github/workflows/release.yml`)

1. **build-game** — матрица из 3 ОС, каждая собирает нативный бинарник и грузит его как
   артефакт GitHub Actions.
2. **build-server-image** — Docker multi-stage сборка `cmd/server`, публикация образа в
   GHCR (`ghcr.io/<repo>-server`) с тегами по ветке/семверу/sha.
3. **release** — срабатывает только на пуш тега `vX.Y.Z`: скачивает все артефакты,
   упаковывает в zip и публикует GitHub Release с автосгенерированными release notes.

Чтобы выпустить релиз:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Локальная сборка вручную

```bash
# Клиент (нужны графические заголовки ОС, см. workflow для списка apt-пакетов на Linux)
go build -o rogalik-game ./cmd/game

# Сервер (без графики, статический бинарник)
CGO_ENABLED=0 go build -o rogalik-server ./cmd/server
```

## Примечание про модули

`go.mod` содержит `replace`-директивы, перенаправляющие `golang.org/x/*` и
`google.golang.org/protobuf` на их зеркала на GitHub. Это не обязательно в окружении с
полным доступом к `proxy.golang.org` (например, в GitHub Actions это не нужно, но и не
мешает), но пригодилось при сборке в среде с ограниченным сетевым доступом — рабочий
пример на случай, если понадобится собирать за похожим прокси/файрволом.
