# Mycel Mesh 🕸️

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://go.dev/)
[![Release](https://img.shields.io/badge/release-v0.2.0-beta-green.svg)](https://github.com/mycel/mesh/releases)

**Mycel Mesh** 是一款基于 WireGuard 协议的轻量级虚拟组网工具，将分布在不同地理位置的设备和服务器无缝连接到虚拟局域网中。

> **Beta Release**: v0.2.0 - 功能持续完善中

---

## ✨ 核心特性

- 🚀 **快速部署** - 一键初始化网络，节点自动加入
- 🔐 **安全可靠** - WireGuard ChaCha20-Poly1305 加密
- 🌐 **NAT 穿透** - STUN + UDP Hole Punching，穿透率>80%
- 📊 **可视管理** - Web UI 实时查看节点状态
- 🛡️ **访问控制** - ACL 规则精细管理网络权限
- 📦 **轻量级** - 单二进制文件，零依赖部署

---

## 🚀 快速开始

### 1. 下载编译

```bash
git clone https://github.com/mycel/mesh.git
cd mesh
make build
```

### 2. 启动 Coordinator

```bash
./bin/coordinator --config config.yaml
```

### 3. 创建网络

```bash
mycelctl init --name my-network
```

### 4. 节点加入

```bash
# 获取加入令牌
mycelctl token create --name node-1

# 节点使用令牌加入
mycelctl join --token <token> --coordinator coordinator.example.com:51820
```

### 5. 查看状态

```bash
mycelctl list
mycelctl status
```

---

## 📦 组件说明

| 组件 | 说明 | 二进制 |
|------|------|--------|
| **Coordinator** | 协调服务器（控制面） | `coordinator` |
| **Agent** | 客户端节点（数据面） | `agent` |
| **mycelctl** | CLI 命令行工具 | `mycelctl` |
| **Web UI** | 可视化管理界面 | - |

---

## 🔧 配置示例

### Coordinator 配置

```yaml
server:
  host: 0.0.0.0
  port: 51820

database:
  driver: postgres
  dsn: postgres://user:pass@localhost:5432/mycel?sslmode=disable

stun:
  servers:
    - stun.l.google.com:19302
    - stun1.l.google.com:19302
```

### Agent 配置

```yaml
node:
  name: node-1
  network: my-network

coordinator:
  address: coordinator.example.com:51820
  token: <join-token>
```

---

## 🛡️ ACL 访问控制

```bash
# 添加允许规则
mycelctl acl add \
  --source node-1 \
  --dest node-2 \
  --action allow \
  --ports 80,443 \
  --protocol tcp

# 添加拒绝规则
mycelctl acl add \
  --source node-3 \
  --dest node-1 \
  --action deny

# 查看规则
mycelctl acl list --network my-network

# 删除规则
mycelctl acl remove --id <rule-id>
```

---

## 🌐 NAT 穿透

Mycel Mesh 内置 STUN 客户端和 UDP Hole Punching 功能：

```bash
# 检测本地 NAT 类型
mycelctl nat info

# 测试与对等体连接
mycelctl nat punch --peer <peer-id>
```

### NAT 类型兼容性

| 本地 \ 远程 | Full Cone | Restricted | Symmetric |
|-------------|-----------|------------|-----------|
| **Full Cone** | ✅ 95% | ✅ 85% | ⚠️ 60% |
| **Restricted** | ✅ 85% | ✅ 80% | ⚠️ 50% |
| **Symmetric** | ⚠️ 60% | ⚠️ 50% | ❌ 20% |

---

## 📊 Web UI

访问 `http://localhost:8080` 打开 Web 管理界面：

- 📋 节点列表 - 查看所有节点状态
- 📈 实时监控 - 延迟、流量统计
- 🔧 配置管理 - ACL 规则、网络设置

---

## 🧪 运行测试

```bash
# 单元测试
go test ./... -short

# 集成测试
go test ./test/integration/... -v

# 压力测试
go test ./test/stress/... -v
```

---

## 📚 文档

| 文档 | 说明 |
|------|------|
| [CHANGELOG.md](CHANGELOG.md) | 版本变更日志 |
| [RELEASE-v0.2.md](RELEASE-v0.2.md) | v0.2 发布说明 |
| [docs/quickstart.md](docs/quickstart.md) | 快速开始指南 |
| [docs/api.md](docs/api.md) | API 文档 |
| [docs/prd-wireguard-vpn.md](docs/prd-wireguard-vpn.md) | PRD 文档 |
| [docs/wireguard-system-design.md](docs/wireguard-system-design.md) | 系统设计 |

---

## 🔜 路线图

### Phase 2 (当前)
- [x] Web UI 框架
- [x] 节点管理页面
- [x] ACL 服务
- [x] NAT 穿透-STUN
- [x] NAT 穿透 - 打洞
- [ ] Beta 发布 ✅ 进行中

### Phase 3 (计划)
- [ ] 多子网支持
- [ ] Exit Node 功能
- [ ] 流量统计
- [ ] Prometheus 监控
- [ ] Grafana Dashboard

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

---

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

## 📞 联系方式

- 官网：https://mycel.mesh
- 邮箱：support@mycel.mesh
- GitHub: https://github.com/mycel/mesh

---

**Built with ❤️ using Go + WireGuard**
