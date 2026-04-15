# Mycel Mesh 🕸️

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://go.dev/)
[![Release](https://img.shields.io/badge/release-v1.0.0-green.svg)](https://github.com/mycel/mesh/releases)
[![Coverage](https://img.shields.io/badge/coverage-78%25-brightgreen.svg)](./test/)

**Mycel Mesh** 是一款基于 WireGuard 协议的轻量级虚拟组网工具，将分布在不同地理位置的设备和服务器无缝连接到虚拟局域网中。

> **GA Release**: v1.0.0 - 生产就绪版本 ✅ 已发布

---

## ✨ 核心特性

- 🚀 **快速部署** - 一键初始化网络，节点自动加入
- 🔐 **安全可靠** - WireGuard ChaCha20-Poly1305 加密
- 🌐 **NAT 穿透** - STUN + UDP Hole Punching，穿透率 87%
- 📊 **可视管理** - Web UI 实时查看节点状态
- 🛡️ **访问控制** - ACL 规则精细管理网络权限
- 📦 **轻量级** - 单二进制文件，零依赖部署
- 📈 **监控告警** - Prometheus + Grafana 完整监控
- 🗂️ **多子网** - 支持多子网隔离与路由

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

## 🗂️ 子网管理

```bash
# 创建子网
mycelctl subnet create --name dev-subnet --cidr 10.0.1.0/24

# 创建隔离子网
mycelctl subnet create --name isolated-subnet --cidr 10.0.2.0/24 --isolated

# 列出子网
mycelctl subnet list

# 查看子网统计
mycelctl subnet stats --name dev-subnet
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

## 📈 监控告警

### Prometheus 指标

Mycel Mesh 提供 32+ Prometheus 指标：

- `mycel_nodes_total` - 总节点数
- `mycel_connections_active` - 活跃连接数
- `mycel_nat_punch_success_rate` - NAT 穿透成功率
- `mycel_traffic_bytes_sent` - 发送流量
- `mycel_traffic_bytes_received` - 接收流量

### Grafana Dashboard

内置 3 个 Grafana Dashboard：

1. **Nodes Status** - 节点状态监控
2. **Traffic Monitor** - 流量监控
3. **NAT Penetration** - NAT 穿透分析

---

## 🧪 运行测试

```bash
# 单元测试
go test ./... -short

# 单元测试 + 覆盖率
go test ./internal/... -cover

# 集成测试
go test ./test/integration/... -v

# 压力测试
go test ./test/stress/... -v
```

### 测试覆盖率

| 模块 | 覆盖率 |
|------|--------|
| internal/pkg/wireguard | 93.9% |
| internal/coordinator/service | 88.3% |
| internal/encoding | 80.0% |
| internal/coordinator/pool | 67.4% |
| internal/agent/config | 60.0% |
| **平均** | **78.0%** |

---

## 📚 文档

| 文档 | 说明 |
|------|------|
| [CHANGELOG.md](CHANGELOG.md) | 版本变更日志 |
| [RELEASE-v1.0.0.md](RELEASE-v1.0.0.md) | v1.0.0 GA 发布说明 |
| [RELEASE-v0.2.md](RELEASE-v0.2.md) | v0.2 Beta 发布说明 |
| [docs/quickstart.md](docs/quickstart.md) | 快速开始指南 |
| [docs/deployment-guide.md](docs/deployment-guide.md) | 部署指南 |
| [docs/prd-wireguard-vpn.md](docs/prd-wireguard-vpn.md) | PRD 文档 |
| [docs/wireguard-system-design.md](docs/wireguard-system-design.md) | 系统设计 |

---

## 🔜 路线图

### Phase 1 MVP ✅ 已完成
- [x] WireGuard 集成
- [x] CLI 基础命令
- [x] Coordinator 框架
- [x] 节点注册服务
- [x] 自动 IP 分配
- [x] MVP 测试

### Phase 2 Beta ✅ 已完成
- [x] Web UI 框架
- [x] 节点管理页面
- [x] ACL 服务
- [x] NAT 穿透-STUN
- [x] NAT 穿透-打洞
- [x] Beta 发布

### Phase 3 GA ✅ 已完成
- [x] 连接池优化
- [x] 压力测试
- [x] Prometheus 监控
- [x] Grafana Dashboard
- [x] 多子网支持
- [x] 子网路由
- [x] 最终测试
- [x] v1.0.0 GA 发布

### v1.1.0 (计划 2026-Q2)
- [ ] 流量统计与报表
- [ ] 审计日志系统
- [ ] 双因素认证
- [ ] 移动端支持 (iOS/Android)

### v1.2.0 (计划 2026-Q3)
- [ ] Exit Node 功能
- [ ] 子网路由优化
- [ ] 多 Coordinator 集群
- [ ] 自动化运维工具

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