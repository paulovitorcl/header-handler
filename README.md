# Header Route Controller

[![Go Version](https://img.shields.io/badge/Go-1.22-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

Simple Kubernetes controller that watches `HeaderRoute` CRDs and generates Envoy routing configuration based on HTTP headers.

## 🎯 Overview

This controller enables **header-based routing** in Kubernetes. It allows you to route HTTP requests to different backend services based on custom header values (e.g., `X-App`, `X-Tenant`, `X-Version`).

```
                    ┌─────────────────┐
                    │   HeaderRoute   │
                    │     CRDs        │
                    └────────┬────────┘
                             │ watch
                             ▼
                    ┌─────────────────┐
                    │   Controller    │
                    │                 │
                    └────────┬────────┘
                             │ generates
                             ▼
                    ┌─────────────────┐
                    │   ConfigMap     │
                    │  (Envoy JSON)   │
                    └────────┬────────┘
                             │ reads
                             ▼
                    ┌─────────────────┐
                    │     Envoy       │
                    │     Proxy       │
                    └─────────────────┘
```

## ✨ Features

- **Header-based routing**: Route requests based on any HTTP header
- **Priority support**: Control route evaluation order
- **Default backend**: Fallback service for unmatched requests
- **404 response**: Return error when no route matches (if no default backend)
- **Dynamic configuration**: Envoy config updated automatically on CRD changes
- **Status reporting**: Track route configuration state

## 📦 What's Included

| Component | Description |
|-----------|-------------|
| `api/v1alpha1/` | HeaderRoute CRD type definitions |
| `controller/` | Reconciliation logic |
| `internal/envoy/` | Envoy configuration generator |
| `config/crd/` | CRD YAML manifest |
| `config/rbac/` | RBAC permissions |

## 🚫 What's NOT Included

This is a **reusable controller library**. It does NOT include:

- Envoy deployment
- Ingress/Gateway
- Backend applications
- Cluster setup (Kind/Minikube)
- Test infrastructure

👉 For a complete working example, see [header-route-poc](https://github.com/seu-user/header-route-poc)

## 📋 CRD Specification

```yaml
apiVersion: routing.example.com/v1alpha1
kind: HeaderRoute
metadata:
  name: route-tenant-a
  namespace: poc
spec:
  # Header to match
  headerName: X-Tenant
  headerValue: tenant-a
  
  # Target backend
  backend:
    name: app-tenant-a
    namespace: poc  # optional, defaults to HeaderRoute namespace
    port: 80
  
  # Route priority (higher = evaluated first)
  priority: 100
```

## ⚙️ Configuration

The controller is configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `CONFIGMAP_NAME` | `envoy-config` | Name of the ConfigMap to update |
| `CONFIGMAP_NAMESPACE` | `poc` | Namespace of the ConfigMap |
| `CONFIGMAP_KEY` | `envoy.json` | Key in ConfigMap for Envoy config |
| `WATCH_NAMESPACE` | `""` (all) | Limit to specific namespace |
| `DEFAULT_BACKEND_SERVICE` | `""` | Fallback service name |
| `DEFAULT_BACKEND_NAMESPACE` | `poc` | Fallback service namespace |
| `DEFAULT_BACKEND_PORT` | `80` | Fallback service port |

## 🚀 Quick Start

### Build

```bash
# Clone the repository
git clone https://github.com/seu-user/header-route-controller.git
cd header-route-controller

# Build locally
go build -o controller ./cmd/manager

# Build Docker image
docker build -t ghcr.io/seu-user/header-route-controller:latest .
```

### Deploy

```bash
# Install CRD
kubectl apply -f config/crd/headerroute.yaml

# Install RBAC
kubectl apply -f config/rbac/rbac.yaml

# Create a Deployment (example)
kubectl create deployment header-route-controller \
  --image=ghcr.io/seu-user/header-route-controller:latest \
  -n poc
```

### Create a Route

```yaml
apiVersion: routing.example.com/v1alpha1
kind: HeaderRoute
metadata:
  name: route-app-a
  namespace: poc
spec:
  headerName: X-App
  headerValue: A
  backend:
    name: app-a
    port: 80
```

## 🔄 How It Works

1. **Watch**: Controller watches all `HeaderRoute` resources
2. **Collect**: On any change, it lists all routes
3. **Sort**: Routes are sorted by priority (highest first)
4. **Generate**: Creates Envoy configuration with:
   - Routes for each HeaderRoute (header match → cluster)
   - Default backend route (if configured)
   - 404 response (if no default backend)
5. **Update**: Writes config to ConfigMap
6. **Reload**: Envoy reads the ConfigMap (via volume mount)

## 📊 Generated Envoy Config

The controller generates a complete Envoy configuration:

```json
{
  "static_resources": {
    "listeners": [{
      "name": "listener_0",
      "address": { "socket_address": { "address": "0.0.0.0", "port_value": 8080 }},
      "filter_chains": [{
        "filters": [{
          "name": "envoy.filters.network.http_connection_manager",
          "typed_config": {
            "route_config": {
              "virtual_hosts": [{
                "routes": [
                  { "match": { "prefix": "/", "headers": [{"name": "X-App", "exact_match": "A"}]}, "route": {"cluster": "poc_app-a"} },
                  { "match": { "prefix": "/" }, "route": {"cluster": "poc_default-backend"} }
                ]
              }]
            }
          }
        }]
      }]
    }],
    "clusters": [
      { "name": "poc_app-a", "load_assignment": { "endpoints": [{"address": "app-a.poc.svc.cluster.local:80"}]}},
      { "name": "poc_default-backend", "load_assignment": { "endpoints": [{"address": "default-backend.poc.svc.cluster.local:80"}]}}
    ]
  }
}
```

## 🧪 Development

```bash
# Run tests
go test ./...

# Run locally (requires kubeconfig)
go run ./cmd/manager

# Generate manifests (if using controller-gen)
make manifests
```

## 📈 Roadmap

- [ ] Webhook validation
- [ ] Multiple header matching (AND/OR)
- [ ] Regex header matching
- [ ] Path prefix support
- [ ] Metrics/observability
- [ ] Gateway API integration

## 📄 License

Apache License 2.0

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
