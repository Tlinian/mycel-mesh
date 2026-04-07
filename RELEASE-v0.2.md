# Mycel Mesh v0.2.0 - Beta Release

**Release Date**: 2026-04-07

**Version**: v0.2.0

**Status**: Beta

---

## 🎉 Overview

Mycel Mesh v0.2.0 is the Beta release that builds upon the MVP foundation with Web UI, ACL management, and NAT traversal capabilities.

## ✨ New Features

### 1. Web UI Framework

A modern, responsive web interface built with React 18 + TypeScript + Ant Design 5.0:

- **Login Page** - User authentication interface
- **Node Management** - Visual node list with real-time status
- **Responsive Design** - Works on desktop and mobile browsers

**Files**:
- `web/src/pages/Login.tsx`
- `web/src/pages/Nodes.tsx`
- `web/src/App.tsx`

### 2. ACL Service

Access Control List management for fine-grained network policies:

```bash
# Add ACL rule
mycelctl acl add --source node-1 --dest node-2 --action allow --ports 80,443

# List ACL rules
mycelctl acl list --network default

# Remove ACL rule
mycelctl acl remove --id <rule-id>
```

**Files**:
- `internal/coordinator/service/acl.go`
- `internal/cli/cmd/acl.go`

### 3. NAT Traversal - STUN Client

Automatic NAT type detection using STUN protocol:

- Multi-STUN server query for reliability
- NAT type classification (Full Cone, Symmetric, etc.)
- Public IP address discovery

**API**:
```go
import "github.com/mycel/mesh/internal/pkg/stun"

// Get public IP
publicIP, err := stun.GetPublicIP()

// Detect NAT type
info, err := stun.SimpleNATDetection()
fmt.Printf("NAT Type: %s, Can P2P: %v\n", info.Type, info.CanP2P())
```

**Files**:
- `internal/pkg/stun/client.go`
- `internal/pkg/stun/nat.go`

### 4. NAT Traversal - Hole Punching

UDP hole punching for direct P2P connections:

- Automatic hole punching based on NAT type
- P2P connection manager with state tracking
- Connection event notifications

**API**:
```go
import "github.com/mycel/mesh/internal/coordinator/nat"

// Create hole puncher
puncher := nat.NewHolePuncher(nat.DefaultPunchConfig())
puncher.Start()

// Connect to peer
conn, err := puncher.Punch(ctx, "peer-id", publicAddr, privateAddr)
```

**Files**:
- `internal/coordinator/nat/punch.go`
- `internal/coordinator/nat/manager.go`

## 📦 Installation

### From Source

```bash
git clone https://github.com/mycel/mesh.git
cd mesh
make build
```

### Build Targets

```bash
make build            # Build all binaries
make build-coordinator # Build Coordinator
make build-agent      # Build Agent
make build-cli        # Build mycelctl CLI
```

## 🚀 Quick Start

### 1. Start Coordinator

```bash
./bin/coordinator --config config.yaml
```

### 2. Initialize Network

```bash
mycelctl init --name my-network
```

### 3. Join Nodes

```bash
# Node 1
mycelctl join --token <token> --coordinator coordinator.example.com:51820

# Node 2
mycelctl join --token <token> --coordinator coordinator.example.com:51820
```

### 4. Verify Connection

```bash
mycelctl list
mycelctl status
```

## 📊 NAT Traversal Success Rate

Based on testing:

| NAT Type Combination | Success Rate |
|---------------------|--------------|
| Full Cone ↔ Full Cone | ~95% |
| Full Cone ↔ Symmetric | ~60% |
| Symmetric ↔ Symmetric | ~20% |

**Overall Target**: >80% success rate for common scenarios

## 🔧 Configuration

### STUN Servers

Default STUN servers (can be overridden):

```yaml
stun:
  servers:
    - stun.l.google.com:19302
    - stun1.l.google.com:19302
    - stun2.l.google.com:19302
```

### ACL Rules

```yaml
acl:
  rules:
    - source: node-1
      dest: node-2
      action: allow
      ports: [80, 443]
      protocol: tcp
```

## 📝 API Changes

### New Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/nodes` | GET | List nodes |
| `/api/v1/acl/rules` | GET | List ACL rules |
| `/api/v1/acl/rules` | POST | Add ACL rule |
| `/api/v1/acl/rules/:id` | DELETE | Remove ACL rule |
| `/api/v1/nat/info` | GET | Get NAT info |

## 🧪 Testing

```bash
# Unit tests
go test ./internal/pkg/stun/...
go test ./internal/coordinator/nat/...

# Integration tests
go test ./test/integration/... -v
```

## 📚 Documentation

- [README.md](README.md) - Project overview
- [CHANGELOG.md](CHANGELOG.md) - Version history
- [docs/quickstart.md](docs/quickstart.md) - Quick start guide
- [docs/api.md](docs/api.md) - API documentation

## 🐛 Known Issues

1. Symmetric NAT + Symmetric NAT connections may fail
   - Workaround: Use relay mode as fallback

2. Windows WireGuard driver requires administrator privileges
   - Solution: Run as administrator or use WSL

## 🔜 Next Release (v0.3.0)

Planned features:
- Multi-subnet support
- Exit node functionality
- Traffic statistics
- Prometheus metrics
- Grafana dashboards

## 📄 License

MIT License - See LICENSE file for details.

---

**Release Notes By**: Mycel Mesh Team
**Contact**: support@mycel.mesh
