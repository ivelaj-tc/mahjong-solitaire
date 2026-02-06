#!/bin/bash
set -e

DOCKER_USER="hungrycoexistance51"
BACKEND_IMAGE="$DOCKER_USER/mahjong-solitaire"
FRONTEND_IMAGE="$DOCKER_USER/mahjong-solitarie-front"
TAG="${1:-latest}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== Building and deploying with tag: $TAG ==="

# Build backend
echo ""
echo "=== Building backend ==="
docker build -t "$BACKEND_IMAGE:$TAG" "$SCRIPT_DIR/backend"

# Build frontend
echo ""
echo "=== Building frontend ==="
docker build -t "$FRONTEND_IMAGE:$TAG" "$SCRIPT_DIR/frontend"

# Push images
echo ""
echo "=== Pushing images ==="
docker push "$BACKEND_IMAGE:$TAG"
docker push "$FRONTEND_IMAGE:$TAG"

# Update deployment manifests with new tag
echo ""
echo "=== Updating manifests to use tag: $TAG ==="
sed -i.bak "s|image: $BACKEND_IMAGE:.*|image: $BACKEND_IMAGE:$TAG|" "$SCRIPT_DIR/k8s/backend-deployment.yaml"
sed -i.bak "s|image: $FRONTEND_IMAGE:.*|image: $FRONTEND_IMAGE:$TAG|" "$SCRIPT_DIR/k8s/frontend-deployment.yaml"
rm -f "$SCRIPT_DIR/k8s/"*.bak

# Apply Kubernetes manifests
echo ""
echo "=== Applying Kubernetes manifests ==="
kubectl apply -f "$SCRIPT_DIR/k8s/namespace.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/backend-pvc.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/redis-master-service.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/redis-master-statefulset.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/redis-replica-service.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/redis-replica-statefulset.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/redis-sentinel-configmap.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/redis-sentinel-service.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/redis-sentinel-statefulset.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/backend-deployment.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/backend-service.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/frontend-configmap.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/frontend-deployment.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/frontend-service.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/backend-hpa.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/frontend-hpa.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/backend-pdb.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/frontend-pdb.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/redis-master-pdb.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/redis-replica-pdb.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/redis-sentinel-pdb.yaml"
kubectl apply -f "$SCRIPT_DIR/k8s/traefik-ingress.yaml"

# Load images into kind cluster (if using kind)
if kind get clusters 2>/dev/null | grep -q .; then
  echo ""
  echo "=== Loading images into kind cluster ==="
  kind load docker-image "$BACKEND_IMAGE:$TAG" "$FRONTEND_IMAGE:$TAG" --name kind
fi

# Restart deployments to pick up new images
echo ""
echo "=== Restarting deployments ==="
kubectl -n mahjong-solitaire rollout restart deployment/backend
kubectl -n mahjong-solitaire rollout restart deployment/frontend

echo ""
echo "=== Done! ==="
echo "Backend:  $BACKEND_IMAGE:$TAG"
echo "Frontend: $FRONTEND_IMAGE:$TAG"
