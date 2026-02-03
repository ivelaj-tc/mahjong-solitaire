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

## Kubernetes (kind + Traefik)
This repo includes manifests under `k8s/` for a kind cluster with a dedicated
namespace, a PVC for SQLite, and Traefik ingress routing.

### 1) Create a kind cluster (optional host port mapping)
Save as `kind-config.yaml` if you want Traefik exposed on localhost:8080:

```
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 80
        hostPort: 8080
        protocol: TCP
```

Create the cluster:
```
kind create cluster --name mahjong-solitaire --config kind-config.yaml
```

### 2) Build + push images to Docker Hub
Replace `YOUR_DOCKERHUB_USER` below:

```
docker build -t YOUR_DOCKERHUB_USER/mahjong-backend:latest backend
docker build -t YOUR_DOCKERHUB_USER/mahjong-frontend:latest frontend
docker push YOUR_DOCKERHUB_USER/mahjong-backend:latest
docker push YOUR_DOCKERHUB_USER/mahjong-frontend:latest
```

### 3) Install Traefik (ingress controller)
Traefik must be installed before applying the ingress manifest:

```
helm repo add traefik https://traefik.github.io/charts
helm repo update
helm install traefik traefik/traefik -n traefik --create-namespace
```

### 4) Update manifests + apply
Edit these files and replace `YOUR_DOCKERHUB_USER`:
- `k8s/backend-deployment.yaml`
- `k8s/frontend-deployment.yaml`

Update `k8s/frontend-configmap.yaml` to use ingress paths:

```
NEXT_PUBLIC_BACKEND_URL: /api
NEXT_PUBLIC_WS_URL: /ws
```

Apply everything:
```
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/backend-pvc.yaml
kubectl apply -f k8s/redis-master-service.yaml
kubectl apply -f k8s/redis-master-statefulset.yaml
kubectl apply -f k8s/redis-replica-service.yaml
kubectl apply -f k8s/redis-replica-statefulset.yaml
kubectl apply -f k8s/redis-sentinel-configmap.yaml
kubectl apply -f k8s/redis-sentinel-service.yaml
kubectl apply -f k8s/redis-sentinel-statefulset.yaml
kubectl apply -f k8s/backend-deployment.yaml
kubectl apply -f k8s/backend-service.yaml
kubectl apply -f k8s/frontend-configmap.yaml
kubectl apply -f k8s/frontend-deployment.yaml
kubectl apply -f k8s/frontend-service.yaml
kubectl apply -f k8s/traefik-ingress.yaml
```

Restart frontend to pick up ConfigMap changes:
```
kubectl -n mahjong-solitaire rollout restart deployment/frontend
```

Open:
- If you mapped hostPort 8080 -> Traefik: http://localhost:8080
- Or port-forward Traefik: `kubectl -n traefik port-forward svc/traefik 8080:80`

## Configuration
Docker Compose sets these by default:
- Backend DB path: `DB_PATH=/data/mahjong.db`
- Frontend API: `NEXT_PUBLIC_BACKEND_URL` and `NEXT_PUBLIC_WS_URL`

Redis (Sentinel) uses:
- `REDIS_SENTINEL_ADDRS` (comma-separated, e.g. `redis-sentinel-0.redis-sentinel:26379,...`)
- `REDIS_SENTINEL_MASTER` (default `mahjong-master`)
- `REDIS_SENTINEL_PASSWORD` (if needed)
- `REDIS_PASSWORD` and `REDIS_DB` (if needed)
- `REDIS_TTL` (duration string, e.g. `24h`) or `REDIS_TTL_HOURS` (e.g. `24`)

Single-node Redis (optional) uses:
- `REDIS_ADDR` (e.g. `redis:6379` in Kubernetes)

Kubernetes sets these via the frontend ConfigMap and writes them into
`/public/runtime-env.js` at container start so the browser can read them.

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
