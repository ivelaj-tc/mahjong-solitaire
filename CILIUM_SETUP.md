# Cilium on kind (used in this project)

## 1) Disable default kindnet CNI
Edit `kind-config.yaml`:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  disableDefaultCNI: true
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"
    extraPortMappings:
      - containerPort: 80
        hostPort: 80
        protocol: TCP
      - containerPort: 443
        hostPort: 443
        protocol: TCP
  - role: worker
  - role: worker
  - role: worker
```

## 2) Recreate the cluster
```bash
kind delete cluster --name kind
kind create cluster --name kind --config kind-config.yaml
```

## 3) Add Cilium Helm repo
```bash
helm repo add cilium https://helm.cilium.io/
helm repo update
```

## 4) Install Cilium (keep kube-proxy)
```bash
helm install cilium cilium/cilium \
  --namespace kube-system \
  --set kubeProxyReplacement=false \
  --set k8sServiceHost=kind-control-plane \
  --set k8sServicePort=6443
```

## 5) Wait for Cilium to be Ready
```bash
kubectl -n kube-system rollout status ds/cilium --timeout=120s
kubectl -n kube-system get pods -o wide | grep cilium
```
