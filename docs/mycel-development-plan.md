# Mycel Mesh 开发计划文档

## 1. 项目背景

### 1.1 产品定位
Mycel Mesh 是一款基于 WireGuard 协议的轻量级虚拟组网工具，将分布在不同地理位置的设备和服务器无缝连接到虚拟局域网中。

### 1.2 文档依据
- PRD 文档：`docs/prd-wireguard-vpn.md` (v1.1)
- 系统设计文档：`docs/wireguard-system-design.md` (v1.0)

---

## 2. 开发范围

### 2.1 核心交付物
| 组件 | 描述 | 优先级 |
|------|------|--------|
| Coordinator | 协调服务器（控制面） | P0 |
| Agent | 客户端节点（数据面） | P0 |
| mycelctl | CLI 命令行工具 | P0 |
| Web UI | 可视化管理界面 | P1 |

### 2.2 功能模块优先级

#### Phase 1 - MVP (P0 功能)
- F1.1 节点注册
- F1.2 节点认证
- F1.4 节点列表
- F2.1 自动 IP 分配
- F2.4 NAT 穿透（基础）
- F3.1 连接状态监控
- F3.3 断线重连
- F5.2 CLI 工具
- F5.5 一键部署

#### Phase 2 - 完善 (P1 功能)
- F1.3 节点下线
- F1.5 节点重命名
- F1.6 节点标签
- F2.2 子网管理
- F2.3 路由配置
- F2.5 Exit Node
- F2.6 子网路由
- F3.2 流量统计
- F4.1 密钥轮换
- F4.2 访问控制 (ACL)
- F4.5 设备指纹
- F5.1 Web 控制台

#### Phase 3 - 生产就绪 (P2 功能)
- F3.4 连接历史
- F3.5 质量评分
- F4.3 审计日志
- F4.4 双因素认证
- F5.3 API 接口
- F5.4 配置导出

---

## 3. 技术架构

### 3.1 技术栈
| 组件 | 技术选型 | 版本 |
|------|----------|------|
| 后端语言 | Go | 1.21+ |
| 数据库 | PostgreSQL | 15.0+ |
| WireGuard | wireguard-go | 0.0.20231106 |
| CLI 框架 | Cobra + Viper | latest |
| 前端框架 | React + TypeScript | 18.2+ |
| UI 组件库 | Ant Design | 5.0+ |
| gRPC | gRPC-Gateway | v2.0+ |
| STUN | pion/stun | v0.6+ |
| 日志 | Zap | v1.26+ |
| 监控 | Prometheus + Grafana | 2.45+ / 10.0+ |

### 3.2 项目目录结构
```
mycel/
├── cmd/
│   ├── coordinator/       # Coordinator 入口
│   ├── agent/             # Agent 入口
│   └── mycelctl/          # CLI 入口
├── internal/
│   ├── coordinator/
│   │   ├── api/           # API Gateway
│   │   ├── service/       # 业务服务
│   │   ├── store/         # 数据持久化
│   │   └── nat/           # NAT 穿透
│   ├── agent/
│   │   ├── config/        # 配置管理
│   │   ├── connection/    # 连接管理
│   │   ├── wireguard/     # WG 接口
│   │   └── daemon/        # 守护进程
│   ├── cli/
│   │   ├── cmd/           # CLI 命令
│   │   └── ui/            # TUI 界面
│   └── pkg/
│       ├── wireguard/     # WG 工具库
│       ├── stun/          # STUN 客户端
│       └── crypto/        # 加密工具
├── pkg/                   # 公共库
├── proto/                 # Protobuf 定义
├── web/                   # Web UI 源码
├── deploy/                # 部署脚本
├── docs/                  # 文档
└── test/                  # 测试
```

---

## 4. 里程碑计划

### Phase 1 - MVP (4 周)

| 周次 | 任务 | 交付内容 | 验收标准 | 关键文件 |
|------|------|----------|----------|----------|
| **W1** | 项目初始化 | Go 模块、目录结构、CI/CD | `go.mod` 创建，基础构建通过 | `cmd/`, `internal/`, `.github/workflows/` |
| **W1** | WireGuard 核心集成 | wg 配置生成、接口创建 | 能手动创建 wg 接口，ping 通 | `internal/pkg/wireguard/` |
| **W1** | CLI 基础命令 | init/join/list/status | `mycelctl init/join` 可用 | `internal/cli/cmd/` |
| **W2** | Coordinator 框架 | gRPC/HTTP 服务、DB 连接 | 服务启动，API 可访问 | `cmd/coordinator/`, `internal/coordinator/api/` |
| **W2** | 节点注册服务 | Register/Heartbeat/List | 节点可注册，获取 IP | `internal/coordinator/service/node.go` |
| **W2** | 数据库 Schema | PostgreSQL 表结构 | 所有表、索引、触发器创建 | `internal/coordinator/store/schema.sql` |
| **W3** | 自动 IP 分配 | DHCP 式分配逻辑 | 新节点自动分配唯一 IP | `internal/coordinator/service/network.go` |
| **W3** | 基础连接测试 | 双节点互通 | ping 通虚拟 IP | `test/integration/basic_test.go` |
| **W4** | MVP 测试 | 内部测试、bug 修复 | 无 P0 bug，3 人天测试 | `test/` |
| **W4** | 文档编写 | 用户文档、API 文档 | README、快速开始指南 | `docs/README.md` |

### Phase 2 - 完善 (4 周)

| 周次 | 任务 | 交付内容 | 验收标准 | 关键文件 |
|------|------|----------|----------|----------|
| **W5** | Web UI 框架 | React 项目、登录页 | SPA 可访问 | `web/` |
| **W5** | 节点管理页面 | 列表、状态、详情 | 可视化展示节点 | `web/src/pages/Nodes.tsx` |
| **W6** | ACL 服务 | 规则管理、校验 | `mycelctl acl add/list` | `internal/coordinator/service/acl.go` |
| **W7** | NAT 穿透-STUN | 多 STUN 服务器查询 | 穿透率>80% | `internal/pkg/stun/` |
| **W7** | NAT 穿透 - 打洞 | UDP 打洞逻辑 | 直连建立 | `internal/coordinator/nat/punch.go` |
| **W8** | Beta 发布 | 外部用户测试 | 10 个用户试用，收集反馈 | Release v0.2 |

### Phase 3 - 生产就绪 (4 周)

| 周次 | 任务 | 交付内容 | 验收标准 | 关键文件 |
|------|------|----------|----------|----------|
| **W9** | 性能优化 | 连接池、批量处理 | 支持 100 节点并发 | 压测报告 |
| **W9** | 压力测试 | 并发、稳定性测试 | 无内存泄漏 | `test/stress/` |
| **W10** | 监控告警 | Prometheus metrics | `/metrics` 端点可用 | `internal/coordinator/metrics/` |
| **W10** | Grafana Dashboard | 可视化监控面板 | 3 个核心 Dashboard | `deploy/grafana/` |
| **W11** | 多子网支持 | 子网划分、路由 | 支持 3 个子网 | `internal/coordinator/service/subnet.go` |
| **W12** | GA 发布 | v1.0.0 正式发布 | 无 P0/P1 bug | Release v1.0.0 |

---

## 5. 关键文件清单

### 5.1 核心代码文件
| 文件路径 | 模块 | 描述 |
|----------|------|------|
| `cmd/coordinator/main.go` | Coordinator | 服务入口 |
| `cmd/agent/main.go` | Agent | 客户端入口 |
| `cmd/mycelctl/main.go` | CLI | 命令行入口 |
| `proto/node.proto` | API | gRPC 定义 |
| `proto/auth.proto` | API | 认证接口定义 |
| `internal/coordinator/api/gateway.go` | Coordinator | API 网关 |
| `internal/coordinator/service/node.go` | Coordinator | 节点服务 |
| `internal/coordinator/service/network.go` | Coordinator | 网络服务 |
| `internal/coordinator/store/postgres.go` | Coordinator | 数据存储 |
| `internal/agent/config/manager.go` | Agent | 配置管理 |
| `internal/agent/connection/manager.go` | Agent | 连接管理 |
| `internal/agent/wireguard/interface.go` | Agent | WG 接口 |
| `internal/pkg/wireguard/config.go` | Utils | WG 配置生成 |
| `internal/pkg/stun/client.go` | Utils | STUN 客户端 |
| `internal/cli/cmd/init.go` | CLI | 初始化命令 |
| `internal/cli/cmd/join.go` | CLI | 加入命令 |

### 5.2 配置文件
| 文件路径 | 描述 |
|----------|------|
| `go.mod` | Go 模块依赖 |
| `go.sum` | 依赖校验 |
| `docker-compose.yml` | 本地开发环境 |
| `deploy/docker/Dockerfile.coordinator` | Coordinator 镜像 |
| `deploy/docker/Dockerfile.agent` | Agent 镜像 |
| `deploy/systemd/mycel-coordinator.service` | systemd 服务 |
| `deploy/systemd/mycel-agent.service` | systemd 服务 |

### 5.3 测试文件
| 文件路径 | 描述 |
|----------|------|
| `test/unit/coordinator/node_test.go` | 节点服务单元测试 |
| `test/unit/agent/config_test.go` | 配置管理单元测试 |
| `test/integration/basic_test.go` | 基础集成测试 |
| `test/integration/acl_test.go` | ACL 集成测试 |
| `test/stress/concurrent_test.go` | 并发压力测试 |

---

## 6. 验收标准

### 6.1 功能验收
- [ ] 节点能成功加入网络并获取 IP
- [ ] 两个节点之间能互相 ping 通
- [ ] NAT 穿透成功率 > 85%
- [ ] 断线后 5 秒内自动重连
- [ ] CLI 所有命令正常工作
- [ ] Web UI 能查看节点状态

### 6.2 性能验收
- [ ] 连接建立时间 < 3s
- [ ] 单隧道吞吐量 > 500 Mbps
- [ ] 同地域延迟 < 10ms
- [ ] Coordinator 支持 100 并发节点

### 6.3 安全验收
- [ ] 无认证密钥无法接入网络
- [ ] API 需 JWT 认证访问
- [ ] 抓包无法解密流量内容
- [ ] 登录失败 5 次自动锁定

---

## 7. 风险与依赖

### 7.1 技术风险
| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| NAT 类型兼容 | 中 | 高 | 支持中继转发模式 |
| 内核模块缺失 | 低 | 中 | 默认使用 wireguard-go |
| Windows 驱动签名 | 中 | 中 | 使用官方驱动 |
| iOS 后台限制 | 高 | 低 | Push Notification 唤醒 |

### 7.2 外部依赖
| 依赖 | 用途 | 备选方案 |
|------|------|----------|
| WireGuard 内核模块 | 数据面加密 | wireguard-go |
| STUN 服务器 | NAT 穿透 | coturn 自建 |
| 公网服务器 | 协调服务器部署 | 阿里云/腾讯云/AWS |

### 7.3 资源依赖
- UI 设计师 1 人周（Web 控制台）
- 多平台测试设备（Windows/Mac/Linux/iOS/Android）

---

## 8. 验证方法

### 8.1 本地开发验证
```bash
# 1. 启动 Coordinator
docker-compose up -d

# 2. 创建加入令牌
mycelctl token create --name test-token

# 3. 启动 Agent 加入网络
mycelctl init --name node-1
mycelctl join --token <token> --coordinator localhost:51820

# 4. 验证连接
mycelctl list
mycelctl status
```

### 8.2 集成测试验证
```bash
# 运行集成测试
go test ./test/integration/... -v

# 运行压力测试
go test ./test/stress/... -v
```

### 8.3 端到端验证
1. 部署 Coordinator 到公网服务器
2. 在 2-3 个不同网络环境的设备上安装 Agent
3. 验证节点间互连互通
4. 验证 NAT 穿透成功率
5. 验证断线重连机制

---

## 9. 参考文档

- PRD 文档：`docs/prd-wireguard-vpn.md`
- 系统设计：`docs/wireguard-system-design.md`
- WireGuard 官方文档：https://www.wireguard.com/
- STUN RFC 8445: https://tools.ietf.org/html/rfc8445

---

**文档版本**: v1.0
**创建日期**: 2026-04-07
**最后更新**: 2026-04-07
