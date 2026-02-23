---
description: Start the RLaaS Kubernetes cluster after a device restart
---

# Start RLaaS Server (after device restart)

Run all commands from the `E:\rlaas` directory.
// turbo-all

## 1. Start Minikube
```powershell
minikube start
```

## 2. Point Docker CLI at Minikube's Docker daemon
```powershell
minikube docker-env --shell powershell | Invoke-Expression
```

## 3. Clean up stale Docker containers (prevents CreateContainerError)
```powershell
docker container prune -f
```

## 4. Build the Docker image inside Minikube
```powershell
docker build -t rlaas-rlaas:latest .
```

## 5. Apply all Kubernetes manifests
```powershell
kubectl apply -f k8s/
```

## 6. Start minikube tunnel (in a separate terminal — keep it open!)
```powershell
minikube tunnel
```
This assigns `localhost` to your LoadBalancer services. Keep this terminal open.

## 7. Wait ~30 seconds, then verify pods are running
```powershell
kubectl get pods
```
All pods should show `Running` status.

## 8. If pods are stuck in CreateContainerError or CrashLoopBackOff
```powershell
minikube docker-env --shell powershell | Invoke-Expression
docker container prune -f
kubectl rollout restart deployment/coredns -n kube-system
kubectl rollout restart deployment/postgres deployment/redis deployment/rlaas
```

## 9. Verify services have external IPs
```powershell
kubectl get svc
```
The rlaas service should show `EXTERNAL-IP: 127.0.0.1`.

## 10. Test connectivity
```powershell
go run .\client\main.go
```

## 11. Start the dashboard (optional, in a separate terminal)
```powershell
cd E:\rlaas\dashboard
npm run dev
```

## Prerequisites (must be installed on any PC)
- Docker Desktop (with WSL 2)
- Minikube (`winget install minikube`)
- kubectl (`winget install Kubernetes.kubectl`)
- Go (`https://go.dev/dl/`)
- Node.js (for the dashboard)
