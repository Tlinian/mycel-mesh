# Changelog

All notable changes to Mycel Mesh will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- Multi-subnet support
- Exit node functionality
- Traffic statistics and monitoring
- Key rotation mechanism

---

## [0.2.0] - 2026-04-07

### Added
- **Web UI Framework** - React + TypeScript SPA with Ant Design components
- **Node Management Page** - Visual node list with status, latency, and traffic display
- **ACL Service** - Access control list management via CLI (`mycelctl acl add/list`)
- **NAT Traversal - STUN** - STUN client for NAT type detection
  - `internal/pkg/stun/client.go` - STUN client with multi-server query
  - `internal/pkg/stun/nat.go` - NAT type detection (Full Cone, Symmetric, etc.)
- **NAT Traversal - Hole Punching** - UDP hole punching for P2P connections
  - `internal/coordinator/nat/punch.go` - Hole punching logic
  - `internal/coordinator/nat/manager.go` - P2P connection manager

### Changed
- Updated Phase 2 progress tracking documents

### Technical
- Added `github.com/pion/stun/v2` dependency
- Created `internal/pkg/stun/` package
- Created `internal/coordinator/nat/` package

### Fixed
- NAT type detection for symmetric NAT scenarios

---

## [0.1.0] - 2026-04-07

### Added
- **Project Initialization**
  - Go module setup (`go.mod`)
  - Directory structure (`cmd/`, `internal/`, `pkg/`, `web/`, `test/`)
  - Makefile with build targets
  - `.gitignore` configuration

- **WireGuard Integration**
  - `internal/pkg/wireguard/config.go` - WireGuard configuration generator
  - `internal/pkg/wireguard/keygen.go` - Key generation using curve25519
  - WGQuick config output format

- **CLI Tool (mycelctl)**
  - `internal/cli/cmd/root.go` - Root command
  - `internal/cli/cmd/init.go` - Network initialization
  - `internal/cli/cmd/join.go` - Join network command
  - `internal/cli/cmd/list.go` - List nodes command
  - `internal/cli/cmd/acl.go` - ACL management commands

- **Coordinator Service**
  - `internal/coordinator/api/gateway.go` - gRPC/HTTP gateway
  - `internal/coordinator/service/node.go` - Node registration service
  - `internal/coordinator/service/network.go` - Auto IP allocation (DHCP-style)
  - `internal/coordinator/service/acl.go` - ACL rule management
  - `internal/coordinator/store/schema.sql` - PostgreSQL database schema
  - `internal/coordinator/store/postgres.go` - PostgreSQL store implementation

- **Database Schema**
  - `networks` table - Virtual network definitions
  - `nodes` table - Node registration and status
  - `auth_tokens` table - Authentication tokens
  - `acl_rules` table - Access control rules
  - `audit_logs` table - Audit logging

- **Web UI**
  - React 18 + TypeScript project structure
  - `web/src/pages/Login.tsx` - Login page
  - `web/src/pages/Nodes.tsx` - Node management page
  - Ant Design component library integration
  - React Router configuration

- **Testing**
  - Unit tests for WireGuard config generation
  - Unit tests for CLI commands
  - Unit tests for node service
  - Integration tests for basic connectivity
  - Test report documentation

- **Documentation**
  - `README.md` - Project overview and quick start
  - `docs/quickstart.md` - Quick start guide
  - `docs/api.md` - API documentation
  - `docs/prd-wireguard-vpn.md` - PRD document
  - `docs/wireguard-system-design.md` - System design document

### Technical Stack
- Backend: Go 1.21+
- Database: PostgreSQL 15+
- WireGuard: wireguard-go
- CLI: Cobra + Viper
- Frontend: React 18 + TypeScript + Ant Design 5.0+
- gRPC: gRPC-Gateway v2

---

## Version History

| Version | Date | Status | Key Features |
|---------|------|--------|--------------|
| 0.1.0 | 2026-04-07 | Released | MVP - Core WireGuard networking |
| 0.2.0 | 2026-04-07 | Beta | Web UI, ACL, NAT Traversal |
