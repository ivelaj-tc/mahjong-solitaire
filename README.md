# Mahjong Solitaire

Mahjong Push Arena (solitaire-style) built with a Go backend and a Next.js frontend.
Players choose categories, play RPS, then push tiles onto a 5x5 board using
solitaire-like stacking rules.

## Features
- Real-time WebSocket gameplay (vs another player or bot)
- Category selection + RPS start
- Push rules: match the bottom-most tile in the column (blank can go anywhere)
- Light/Dark theme
- SVG tile assets

## Quick Start (Docker)
From this directory:

```
docker compose up -d --build
```

Open:
- Frontend: http://localhost:3000
- Backend health: http://localhost:8080/health

Stop:
```
docker compose down
```

## Configuration
Docker Compose sets these by default:
- Backend DB path: `DB_PATH=/data/mahjong.db`
- Frontend API: `NEXT_PUBLIC_BACKEND_URL` and `NEXT_PUBLIC_WS_URL`

## Tile Assets (SVG only)
Place SVG tiles in:
- `frontend/public/tiles/`
- or `frontend/public/tiles/svg/`

The filename should match the tile symbol (e.g. `kokushibo.svg`).

## Development (optional)
Backend:
```
go run ./cmd/mahjong-server
```

Frontend:
```
npm install
npm run dev
```
