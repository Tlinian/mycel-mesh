# Mycel Mesh - 系统设计文档 (System Design)

## 1. 文档信息

| 项目 | 内容 |
|------|------|
| 产品名称 | Mycel Mesh |
| 文档版本 | v1.0 |
| 创建日期 | 2026-04-07 |
| 文档类型 | 技术设计文档 |
| 关联 PRD | `docs/prd-wireguard-vpn.md` |

---

## 2. 系统概述

### 2.1 设计目标

基于 PRD 需求，本系统设计实现以下核心目标：

1. **虚拟组网**：将分布在不同地理位置的设备通过 WireGuard 协议连接到一个虚拟局域网
2. **自动穿透**：STUN/TURN 自动 NAT 穿透，直连成功率 > 85%
3. **零配置**：节点加入后自动获取 IP、自动配置路由、自动发现 peers
4. **安全可靠**：基于 WireGuard 的 ChaCha20-Poly1305 加密，密钥对身份认证

### 2.2 系统边界

```
┌─────────────────────────────────────────────────────────────────┐
│                        Mycel Mesh 系统边界                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  内部系统                     外部依赖                          │
│  ┌──────────────┐            ┌──────────────┐                  │
│  │  Coordinator │◄──────────►│  STUN Server │                  │
│  │  (控制面)    │            │  (公网)      │                  │
│  └──────────────┘            └──────────────┘                  │
│         │                              │                         │
│         ▼                              ▼                         │
│  ┌──────────────┐            ┌──────────────┐                  │
│  │   Agent      │◄──────────►│ WireGuard    │                  │
│  │  (数据面)    │            │  Kernel/Go   │                  │
│  └──────────────┘            └──────────────┘                  │
│                                                                 │
│  用户界面                     第三方服务                        │
│  ┌──────────────┐            ┌──────────────┐                  │
│  │  CLI/Web UI  │            │  Prometheus  │                  │
│  │  (管理面)    │            │  Grafana     │                  │
│  └──────────────┘            └──────────────┘                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 架构设计

### 3.1 整体架构

系统采用 **控制面与数据面分离** 的架构设计：

```
                    ┌─────────────────────────┐
                    │      Control Plane      │
                    │    (协调服务器)          │
                    │                         │
                    │  ┌───────────────────┐  │
                    │  │   API Gateway     │  │
                    │  │  (HTTP/gRPC/WS)   │  │
                    │  └─────────┬─────────┘  │
                    │            │            │
                    │   ┌────────┴────────┐   │
                    │   ▼                 ▼   │
                    │ ┌─────┐         ┌─────┐ │
                    │ │Node │         │Net  │ │
                    │ │ Mgr │         │ Mgr │ │
                    │ └─────┘         └─────┘ │
                    │            │            │
                    │   ┌────────┴────────┐   │
                    │   ▼                 ▼   │
                    │ ┌─────┐         ┌─────┐ │
                    │ │Signal│        │ ACL │ │
                    │ │ Mgr │         │ Mgr │ │
                    │ └─────┘         └─────┘ │
                    │            │            │
                    │   ┌────────┴────────┐   │
                    │   ▼                 ▼   │
                    │ ┌─────┐         ┌─────┐ │
                    │ │ NAT │         │ DB  │ │
                    │ │Trail│         │(PG)  │
                    │ └─────┘         └─────┘ │
                    └─────────────────────────┘
                              │
                              │ gRPC/HTTPS
                              │
              ┌───────────────┼───────────────┐
              │               │               │
              ▼               ▼               ▼
        ┌──────────┐   ┌──────────┐   ┌──────────┐
        │  Agent   │   │  Agent   │   │  Agent   │
        │ (Node A) │   │ (Node B) │   │ (Node C) │
        │10.0.0.2  │   │10.0.0.3  │   │10.0.0.4  │
        └────┬─────┘   └────┬─────┘   └────┬─────┘
             │              │              │
             └──────────────┴──────────────┘
                        │
                  WireGuard
                  Data Plane
             (加密直连或中继)
```

### 3.2 技术栈详细选型

| 层级 | 组件 | 技术选型 | 版本 | 选型原因 |
|------|------|----------|------|----------|
| **控制面** | Coordinator | Go | 1.21+ | 跨平台、并发好、wireguard-go 官方支持 |
| **控制面** | API Gateway | gRPC-Gateway | v2.0+ | 同时支持 gRPC 和 REST |
| **控制面** | Database | PostgreSQL | 15.0+ | 生产级、支持 JSONB、行级锁、多 Coordinator 共享 |
| **控制面** | Cache | Redis | 7.0+ | 可选，用于会话和临时数据 |
| **数据面** | Agent | Go | 1.21+ | 单一二进制、跨平台 |
| **数据面** | WireGuard | wireguard-go | 0.0.20231106 | 用户态实现，无需内核模块 |
| **数据面** | TUN/TAP | gvisor.dev/gvisor | pkg/tcpip | 跨平台 TUN 接口 |
| **管理面** | CLI | Cobra + Viper | latest | Go 标准 CLI 框架 |
| **管理面** | Web UI | React 18 + TypeScript | 18.2+ | 生态成熟、类型安全 |
| **管理面** | UI 组件库 | Ant Design | 5.0+ | 企业级组件 |
| **基础设施** | STUN | pion/stun | v0.6+ | Go 实现，纯客户端 |
| **基础设施** | TURN | coturn | 4.3+ | 行业标准，可选部署 |
| **运维** | 监控 | Prometheus | 2.45+ | 指标收集 |
| **运维** | 可视化 | Grafana | 10.0+ | Dashboard |
| **运维** | 日志 | Zap | v1.26+ | 高性能结构化日志 |

---

## 4. 模块设计

### 4.1 Coordinator 模块设计

#### 4.1.1 服务架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Coordinator Service                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   API Layer                            │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌───────────────┐  │  │
│  │  │ HTTP/REST   │  │   gRPC      │  │  WebSocket    │  │  │
│  │  │ (Gateway)   │  │  (Internal) │  │   (Realtime)  │  │  │
│  │  └─────────────┘  └─────────────┘  └───────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                  Business Layer                        │  │
│  │                                                        │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────────┐   │  │
│  │  │ AuthService│  │ NodeService│  │ NetworkService │   │  │
│  │  │            │  │            │  │                │   │  │
│  │  │ - Login    │  │ - Register │  │ - Create       │   │  │
│  │  │ - Refresh  │  │ - Join     │  │ - Update       │   │  │
│  │  │ - 2FA      │  │ - Leave    │  │ - Delete       │   │  │
│  │  │            │  │ - List     │  │ - GetConfig    │   │  │
│  │  └────────────┘  │ - Heartbeat│  │                │   │  │
│  │                  └────────────┘  └────────────────┘   │  │
│  │                                                        │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────────┐   │  │
│  │  │SignalService│ │ NATService │  │  ACLService    │   │  │
│  │  │            │  │            │  │                │   │  │
│  │  │ - Exchange │  │ - STUN     │  │ - AddRule      │   │  │
│  │  │ - KeyDist  │  │ - PortMap  │  │ - Check        │   │  │
│  │  │ - Notify   │  │ - Relay    │  │ - List         │   │  │
│  │  └────────────┘  └────────────┘  └────────────────┘   │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   Data Layer                           │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌───────────────┐  │  │
│  │  │  SQLite DB  │  │   Redis     │  │  File Store   │  │  │
│  │  │  (Persistent)│ │  (Cache)    │  │  (Config)     │  │  │
│  │  └─────────────┘  └─────────────┘  └───────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### 4.1.2 核心接口定义 (gRPC)

```protobuf
// proto/node.proto
syntax = "proto3";
package mycel.v1;

service NodeService {
  // 节点注册
  rpc Register(RegisterRequest) returns (RegisterResponse);
  // 节点心跳
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  // 获取节点列表
  rpc ListNodes(ListNodesRequest) returns (ListNodesResponse);
  // 节点下线
  rpc Unregister(UnregisterRequest) returns (UnregisterResponse);
  // 获取 Peer 配置
  rpc GetPeerConfig(GetPeerConfigRequest) returns (GetPeerConfigResponse);
}

message RegisterRequest {
  string name = 1;
  string public_key = 2;
  string auth_token = 3;
  NodeInfo info = 4;
}

message NodeInfo {
  string os = 1;
  string arch = 2;
  string version = 3;
  repeated Endpoint endpoints = 4;
}

message Endpoint {
  string ip = 1;
  int32 port = 2;
  string type = 3; // "public", "local", "stun"
}

message RegisterResponse {
  string node_id = 1;
  string assigned_ip = 2;
  string subnet_mask = 3;
  repeated PeerConfig peers = 4;
}

message PeerConfig {
  string node_id = 1;
  string name = 2;
  string public_key = 3;
  string private_ip = 4;
  repeated string endpoints = 5;
}

// 心跳
message HeartbeatRequest {
  string node_id = 1;
  int64 timestamp = 2;
  NodeStatus status = 3;
}

message NodeStatus {
  bool online = 1;
  int64 rx_bytes = 2;
  int64 tx_bytes = 3;
  int32 latency_ms = 4;
}

message HeartbeatResponse {
  bool success = 1;
  int64 next_heartbeat = 2;
}
```

#### 4.1.3 数据库 Schema (PostgreSQL)

```sql
-- 扩展：UUID 支持
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 网络表
CREATE TABLE networks (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            TEXT NOT NULL UNIQUE,
    subnet_cidr     CIDR NOT NULL,
    coordinator_host TEXT NOT NULL,
    mtu             INTEGER DEFAULT 1420,
    created_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- 节点表
CREATE TABLE nodes (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    network_id      UUID NOT NULL REFERENCES networks(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    public_key      TEXT NOT NULL UNIQUE,
    private_ip      INET NOT NULL,
    endpoint        TEXT,
    status          TEXT DEFAULT 'offline',
    os              TEXT,
    arch            TEXT,
    version         TEXT,
    rx_bytes        BIGINT DEFAULT 0,
    tx_bytes        BIGINT DEFAULT 0,
    last_seen       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- 访问令牌表
CREATE TABLE auth_tokens (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    network_id      UUID NOT NULL REFERENCES networks(id) ON DELETE CASCADE,
    token           TEXT NOT NULL UNIQUE,
    name            TEXT,
    expires_at      TIMESTAMPTZ,
    usage_limit     INTEGER,
    used_count      INTEGER DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- 访问控制表
CREATE TABLE acl_rules (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    network_id      UUID NOT NULL REFERENCES networks(id) ON DELETE CASCADE,
    source_node_id  UUID,
    source_cidr     CIDR,
    dest_node_id    UUID,
    dest_cidr       CIDR,
    action          TEXT NOT NULL CHECK (action IN ('allow', 'deny')),
    ports           JSONB,
    protocol        TEXT DEFAULT 'tcp',
    priority        INTEGER DEFAULT 100,
    created_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- 审计日志表（分区表，按月分区）
CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    network_id      UUID,
    node_id         UUID,
    action          TEXT NOT NULL,
    detail          JSONB,
    ip              INET,
    created_at      TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
) PARTITION BY RANGE (created_at);

-- 审计日志分区示例
CREATE TABLE audit_logs_2026_04 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

-- 索引
CREATE INDEX idx_nodes_network ON nodes(network_id);
CREATE INDEX idx_nodes_status ON nodes(status);
CREATE INDEX idx_nodes_last_seen ON nodes(last_seen) WHERE status = 'online';
CREATE INDEX idx_acl_network ON acl_rules(network_id);
CREATE INDEX idx_audit_network ON audit_logs(network_id);
CREATE INDEX idx_audit_created ON audit_logs(created_at);
CREATE INDEX idx_audit_network_created ON audit_logs(network_id, created_at);

-- 触发器：自动更新 updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_networks_updated_at BEFORE UPDATE ON networks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_nodes_updated_at BEFORE UPDATE ON nodes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### 4.2 Agent 模块设计

#### 4.2.1 Agent 架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Mycel Agent                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   Control Plane                        │  │
│  │                                                        │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌───────────────┐  │  │
│  │  │   Config    │  │  Connection │  │   Heartbeat   │  │  │
│  │  │   Manager   │  │   Manager   │  │   Service     │  │  │
│  │  │             │  │             │  │               │  │  │
│  │  │ - Load      │  │ - Dial      │  │ - Report      │  │  │
│  │  │ - Save      │  │ - Listen    │  │ - Keepalive   │  │  │
│  │  │ - Watch     │  │ - Close     │  │ - Reconnect   │  │  │
│  │  └─────────────┘  └─────────────┘  └───────────────┘  │  │
│  │                                                        │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌───────────────┐  │  │
│  │  │   NAT       │  │   Route     │  │    Daemon     │  │  │
│  │  │   Traversal │  │   Manager   │  │    Service    │  │  │
│  │  │             │  │             │  │               │  │  │
│  │  │ - STUN      │  │ - Add       │  │ - Start       │  │  │
│  │  │ - PortMap   │  │ - Delete    │  │ - Stop        │  │  │
│  │  │ - HolePunch │  │ - Update    │  │ - Restart     │  │  │
│  │  └─────────────┘  └─────────────┘  └───────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                          │                                  │
│                          ▼                                  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   Data Plane                           │  │
│  │                                                        │  │
│  │  ┌─────────────────────────────────────────────────┐   │  │
│  │  │              WireGuard Interface                │   │  │
│  │  │               (wg0 / utun)                      │   │  │
│  │  │                                                 │   │  │
│  │  │  ┌───────────┐  ┌───────────┐  ┌────────────┐   │   │  │
│  │  │  │ Encryption│  │ Routing   │  │  Traffic   │   │   │  │
│  │  │  │ ChaCha20  │  │ Table     │  │  Stats     │   │   │  │
│  │  │  │ Poly1305  │  │           │  │  RX/TX     │   │   │  │
│  │  │  └───────────┘  └───────────┘  └────────────┘   │   │  │
│  │  └─────────────────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### 4.2.2 Agent 状态机

```
                   ┌─────────────┐
                   │   STOPPED   │
                   └──────┬──────┘
                          │ init()
                          ▼
                   ┌─────────────┐
              ┌────│   INIT      │◄───┐
              │    └──────┬──────┘    │
              │           │ connect() │ reconnect()
              │           ▼           │
              │    ┌─────────────┐    │
              │    │ CONNECTING  │────┘
              │    └──────┬──────┘ (failure)
              │           │ connected()
              │           ▼
              │    ┌─────────────┐
              │    │   ONLINE    │◄──────┐
              │    └──────┬──────┘       │
              │           │              │ heartbeat()
              │     heartbeat()    ┌─────┘
              │     timeout/       │
              │     disconnect()   │
              │           │        │
              │           ▼        │
              │    ┌─────────────┐ │
              └───►│  RECONNECTING│─┘
                   └─────────────┘
```

#### 4.2.3 WireGuard 配置生成

```go
// WireGuard 配置结构
type WGConfig struct {
    Interface struct {
        PrivateKey string `json:"private_key"`
        Address    string `json:"address"`
        DNS        string `json:"dns"`
        MTU        int    `json:"mtu"`
    }
    Peers []struct {
        PublicKey          string   `json:"public_key"`
        AllowedIPs         []string `json:"allowed_ips"`
        Endpoint           string   `json:"endpoint"`
        PersistentKeepalive int     `json:"persistent_keepalive"`
    }
}

// 生成 wg-quick 配置格式
func (c *WGConfig) ToWgQuick() string {
    var buf strings.Builder

    buf.WriteString("[Interface]\n")
    buf.WriteString(fmt.Sprintf("PrivateKey = %s\n", c.Interface.PrivateKey))
    buf.WriteString(fmt.Sprintf("Address = %s\n", c.Interface.Address))
    buf.WriteString(fmt.Sprintf("DNS = %s\n", c.Interface.DNS))
    buf.WriteString(fmt.Sprintf("MTU = %d\n", c.Interface.MTU))

    for _, peer := range c.Peers {
        buf.WriteString("\n[Peer]\n")
        buf.WriteString(fmt.Sprintf("PublicKey = %s\n", peer.PublicKey))
        buf.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(peer.AllowedIPs, ", ")))
        if peer.Endpoint != "" {
            buf.WriteString(fmt.Sprintf("Endpoint = %s\n", peer.Endpoint))
        }
        if peer.PersistentKeepalive > 0 {
            buf.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", peer.PersistentKeepalive))
        }
    }

    return buf.String()
}
```

### 4.3 NAT 穿透模块设计

#### 4.3.1 穿透策略

```
┌─────────────────────────────────────────────────────────────┐
│                    NAT Traversal Strategy                     │
├─────────────────────────────────────────────────────────────┘

Step 1: 检查本地端点
  └─► 获取所有本地 IP (IPv4/IPv6)
  └─► 获取监听端口

Step 2: STUN 查询
  └─► 向公共 STUN 服务器发送 Binding Request
  └─► 获取公网 IP:Port (X:Y)
  └─► 如果失败，标记为"对称 NAT"，走 Step 5

Step 3: 端口预测 (针对 Port-Restricted NAT)
  └─► 发送多个 STUN 请求，分析端口分配模式
  └─► 预测下一个端口号

Step 4: 打洞
  └─► 节点 A 向节点 B 的公网 IP:Port 发送 UDP 包
  └─► 节点 B 向节点 A 的公网 IP:Port 发送 UDP 包
  └─► 如果成功，建立直连
  └─► 如果失败，走 Step 5

Step 5: 中继模式
  └─► 通过 Coordinator 转发流量
  └─► 同时继续尝试打洞
```

#### 4.3.2 STUN 客户端实现

```go
// NAT 类型检测
func DetectNATType(stunServers []string) NATType {
    results := make([]NATResult, 0)

    for _, server := range stunServers {
        result := querySTUN(server)
        results = append(results, result)
    }

    // 分析结果，判断 NAT 类型
    return analyzeNATType(results)
}

// STUN 查询
func querySTUN(server string) NATResult {
    conn, _ := net.DialUDP("udp4", nil, nil)
    request := stun.Build(stun.TransactionID, stun.BindingRequest)
    response, _ := conn.Do(request, server)

    var mappedAddr stun.XORMappedAddress
    response.Parse(&mappedAddr)

    return NATResult{
        LocalAddr:  conn.LocalAddr().(*net.UDPAddr),
        PublicAddr: &mappedAddr,
        Server:     server,
    }
}
```

### 4.4 CLI 模块设计

#### 4.4.1 命令结构

```
mycelctl
├── init                    # 初始化节点
│   └── --name <string>     # 节点名称
├── join                    # 加入网络
│   ├── --token <string>    # 加入令牌
│   └── --coordinator <string>
├── leave                   # 离开网络
├── list                    # 列出节点
├── status                  # 查看状态
├── config                  # 配置管理
│   ├── export              # 导出配置
│   └── import              # 导入配置
├── acl                     # 访问控制
│   ├── add                 # 添加规则
│   ├── remove              # 删除规则
│   └── list                # 列出规则
├── exit-node               # Exit Node 管理
│   ├── set                 # 设置出口节点
│   └── disable             # 禁用出口节点
└── daemon                  # 守护进程
    ├── start               # 启动服务
    ├── stop                # 停止服务
    └── restart             # 重启服务
```

#### 4.4.2 CLI 命令示例

```go
// cmd/init.go
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "初始化 Mycel 节点",
    Long:  "生成节点密钥对，创建本地配置文件",
    RunE: func(cmd *cobra.Command, args []string) error {
        name, _ := cmd.Flags().GetString("name")

        // 生成密钥对
        privateKey, publicKey, _ := wgcfg.GenerateKey()

        // 保存配置
        cfg := &Config{
            Name:       name,
            PrivateKey: privateKey,
            PublicKey:  publicKey,
        }

        if err := cfg.Save(); err != nil {
            return err
        }

        fmt.Printf("节点初始化成功!\n")
        fmt.Printf("公钥：%s\n", publicKey)
        return nil
    },
}
```

---

## 5. 接口设计

### 5.1 REST API 详细设计

#### 认证接口

```yaml
POST /api/v1/auth/login:
  summary: 管理员登录
  tags: [Auth]
  requestBody:
    content:
      application/json:
        schema:
          type: object
          properties:
            username:
              type: string
            password:
              type: string
  responses:
    200:
      content:
        application/json:
          schema:
            type: object
            properties:
              token:
                type: string
              expires_in:
                type: integer
              refresh_token:
                type: string

POST /api/v1/auth/token:
  summary: 创建加入令牌
  tags: [Auth]
  security:
    - bearerAuth: []
  requestBody:
    content:
      application/json:
        schema:
          type: object
          properties:
            network_id:
              type: string
            name:
              type: string
            expires_in:
              type: integer
            usage_limit:
              type: integer
  responses:
    200:
      content:
        application/json:
          schema:
            type: object
            properties:
              token:
                type: string
              expires_at:
                type: string
```

#### 节点接口

```yaml
POST /api/v1/nodes:
  summary: 注册节点
  tags: [Nodes]
  requestBody:
    content:
      application/json:
        schema:
          type: object
          required:
            - name
            - public_key
            - auth_token
          properties:
            name:
              type: string
            public_key:
              type: string
            auth_token:
              type: string
            info:
              $ref: '#/components/schemas/NodeInfo'
  responses:
    200:
      content:
        application/json:
          schema:
            type: object
            properties:
              node_id:
                type: string
              assigned_ip:
                type: string
              peers:
                type: array
                items:
                  $ref: '#/components/schemas/PeerConfig'

GET /api/v1/nodes:
  summary: 获取节点列表
  tags: [Nodes]
  parameters:
    - name: status
      in: query
      schema:
        type: string
        enum: [online, offline, all]
    - name: limit
      in: query
      schema:
        type: integer
        default: 50
  responses:
    200:
      content:
        application/json:
          schema:
            type: object
            properties:
              total:
                type: integer
              nodes:
                type: array
                items:
                  $ref: '#/components/schemas/Node'

DELETE /api/v1/nodes/{node_id}:
  summary: 删除节点
  tags: [Nodes]
  parameters:
    - name: node_id
      in: path
      required: true
      schema:
        type: string
  responses:
    204:
      description: Node deleted successfully
```

### 5.2 内部事件设计

#### 事件类型定义

```go
// 事件类型
const (
    EventTypeNodeRegistered   = "node.registered"
    EventTypeNodeUnregistered = "node.unregistered"
    EventTypeNodeHeartbeat    = "node.heartbeat"
    EventTypeACLUpdated       = "acl.updated"
    EventTypeNetworkUpdated   = "network.updated"
)

// 事件结构
type Event struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Timestamp time.Time              `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
}

// 事件处理器
type EventHandler func(event Event) error

// 事件总线
type EventBus struct {
    mu       sync.RWMutex
    handlers map[string][]EventHandler
}

func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
    eb.mu.Lock()
    defer eb.mu.Unlock()
    eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

func (eb *EventBus) Publish(event Event) {
    eb.mu.RLock()
    defer eb.mu.RUnlock()

    handlers := eb.handlers[event.Type]
    for _, h := range handlers {
        go h(event)
    }
}
```

---

## 6. 安全设计

### 6.1 身份认证

```
┌─────────────────────────────────────────────────────────────┐
│                    Authentication Flow                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. 管理员登录                                               │
│  ┌─────────┐         ┌─────────────┐         ┌─────────┐   │
│  │  Admin  │────────►│ Coordinator │         │   DB    │   │
│  │  Client │  登录   │   AuthSvc   │────────►│ (验证)  │   │
│  └─────────┘         └──────┬──────┘         └─────────┘   │
│                             │                               │
│                             │ JWT Token                     │
│                             ◄───────────────────────────────│
│                                                             │
│  2. 节点加入                                                 │
│  ┌─────────┐         ┌─────────────┐         ┌─────────┐   │
│  │  Node   │────────►│ Coordinator │         │  Token  │   │
│  │  Agent  │  注册   │   NodeSvc   │────────►│ (验证)  │   │
│  └─────────┘         └──────┬──────┘         └─────────┘   │
│                             │                               │
│                             │ IP + Peer List                │
│                             ◄───────────────────────────────│
│                                                             │
│  3. 节点间通信                                               │
│  ┌─────────┐                      ┌─────────┐               │
│  │  Node A │══════════════════════│  Node B │               │
│  └─────────┘   WireGuard Tunnel   └─────────┘               │
│      │                                    │                  │
│      │ 加密流量                            │ 解密流量         │
│      │ ChaCha20-Poly1305                  │                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 6.2 安全机制

| 层面 | 机制 | 实现 |
|------|------|------|
| **传输安全** | 加密算法 | WireGuard ChaCha20-Poly1305 |
| **密钥交换** | 密钥生成 | Curve25519 ECDH |
| **身份认证** | 节点认证 | 公钥指纹验证 + 加入令牌 |
| **访问控制** | API 鉴权 | JWT Bearer Token |
| **权限管理** | RBAC | 管理员/普通用户 |
| **审计日志** | 操作记录 | 所有敏感操作记录到数据库 |
| **速率限制** | 防暴力破解 | 登录失败 5 次锁定 15 分钟 |

---

## 7. 部署设计

### 7.1 Coordinator 部署架构

#### 单节点部署 (开发/测试)

```yaml
# docker-compose.yml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: mycel
      POSTGRES_PASSWORD: mycel_secret
      POSTGRES_DB: mycel
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U mycel"]
      interval: 5s
      timeout: 5s
      retries: 5

  coordinator:
    image: mycel/coordinator:latest
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "51820:51820/udp"  # WireGuard
      - "8080:8080"        # HTTP API
      - "9090:9090"        # gRPC
    environment:
      - MYCEL_DB_URL=postgres://mycel:mycel_secret@postgres:5432/mycel
      - MYCEL_NETWORK_NAME=default
      - MYCEL_SUBNET=10.0.0.0/24
      - MYCEL_AUTH_TOKEN=secret-token
    restart: unless-stopped

volumes:
  pgdata:
```

#### 高可用部署 (生产)

```
                     ┌─────────────────┐
                     │   Load Balancer │
                     │   (Nginx/ALB)   │
                     └────────┬────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
              ▼               ▼               ▼
        ┌──────────┐   ┌──────────┐   ┌──────────┐
        │Coordinator│   │Coordinator│   │Coordinator│
        │   Node 1  │   │   Node 2  │   │   Node 3  │
        └────┬─────┘   └────┬─────┘   └────┬─────┘
             │              │              │
             └──────────────┴──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │    Redis     │
                     │   (Cluster)  │
                     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │  PostgreSQL  │
                     │    (HA)      │
                     │  Patroni/PGB │
                     └──────────────┘
```

### 7.2 Agent 部署

#### Linux systemd 服务

```ini
# /etc/systemd/system/mycel-agent.service
[Unit]
Description=Mycel Mesh Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/mycel-agent daemon start
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

#### Windows 服务

```powershell
# 安装服务
sc.exe create MycelAgent binPath= "C:\Program Files\Mycel\mycel-agent.exe daemon start" start= auto
sc.exe failure MycelAgent reset= 0 actions= restart/5000/restart/5000/restart/5000
```

---

## 8. 监控与运维

### 8.1 Prometheus Metrics

```go
// 指标定义
var (
    // 节点相关
    metricNodesTotal = promauto.NewGauge(prometheus.GaugeOpts{
        Namespace: "mycel",
        Name:      "nodes_total",
        Help:      "Total number of registered nodes",
    })

    metricNodesOnline = promauto.NewGauge(prometheus.GaugeOpts{
        Namespace: "mycel",
        Name:      "nodes_online",
        Help:      "Number of online nodes",
    })

    // 连接相关
    metricConnectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
        Namespace: "mycel",
        Name:      "connections_total",
        Help:      "Total number of connections established",
    })

    // 流量相关
    metricTrafficRX = promauto.NewCounter(prometheus.CounterOpts{
        Namespace: "mycel",
        Name:      "traffic_rx_bytes_total",
        Help:      "Total bytes received",
    })

    metricTrafficTX = promauto.NewCounter(prometheus.CounterOpts{
        Namespace: "mycel",
        Name:      "traffic_tx_bytes_total",
        Help:      "Total bytes transmitted",
    })

    // NAT 穿透
    metricNATPunchSuccess = promauto.NewCounter(prometheus.CounterOpts{
        Namespace: "mycel",
        Name:      "nat_punch_success_total",
        Help:      "Total successful NAT punch attempts",
    })

    metricNATPunchFailure = promauto.NewCounter(prometheus.CounterOpts{
        Namespace: "mycel",
        Name:      "nat_punch_failure_total",
        Help:      "Total failed NAT punch attempts",
    })
)
```

### 8.2 Grafana Dashboard

```json
{
  "dashboard": {
    "title": "Mycel Mesh Overview",
    "panels": [
      {
        "title": "Nodes Online",
        "targets": [
          {
            "expr": "mycel_nodes_online",
            "legendFormat": "Online"
          }
        ]
      },
      {
        "title": "Traffic (RX/TX)",
        "targets": [
          {
            "expr": "rate(mycel_traffic_rx_bytes_total[5m])",
            "legendFormat": "RX"
          },
          {
            "expr": "rate(mycel_traffic_tx_bytes_total[5m])",
            "legendFormat": "TX"
          }
        ]
      },
      {
        "title": "NAT Punch Success Rate",
        "targets": [
          {
            "expr": "mycel_nat_punch_success_total / (mycel_nat_punch_success_total + mycel_nat_punch_failure_total)",
            "legendFormat": "Success Rate"
          }
        ]
      }
    ]
  }
}
```

---

## 9. 测试策略

### 9.1 单元测试覆盖

| 模块 | 目标覆盖率 | 关键测试点 |
|------|------------|------------|
| Coordinator | > 80% | 节点注册、IP 分配、ACL 校验 |
| Agent | > 80% | 配置生成、连接管理、心跳 |
| NAT 穿透 | > 70% | STUN 查询、端口预测 |
| CLI | > 60% | 命令解析、配置读写 |

### 9.2 集成测试场景

```go
// 测试场景定义
var testScenarios = []TestScenario{
    {
        Name:        "单节点加入",
        Description: "新节点成功加入网络并获取 IP",
        Steps: []string{
            "启动 Coordinator",
            "创建加入令牌",
            "节点执行 join",
            "验证获取到正确 IP",
            "验证 wg 接口创建成功",
        },
    },
    {
        Name:        "双节点直连",
        Description: "两个节点之间建立 WireGuard 直连",
        Steps: []string{
            "节点 A 加入网络",
            "节点 B 加入网络",
            "验证 A->B ping 通",
            "验证 B->A ping 通",
            "验证流量统计",
        },
    },
    {
        Name:        "NAT 穿透",
        Description: "验证不同 NAT 类型下的穿透成功率",
        Steps: []string{
            "节点 A 在 Full Cone NAT 后",
            "节点 B 在 Symmetric NAT 后",
            "验证 A->B 直连成功",
            "验证穿透失败时走中继",
        },
    },
    {
        Name:        "ACL 验证",
        Description: "验证访问控制规则生效",
        Steps: []string{
            "配置 A 不能访问 B",
            "验证 A->B 不通",
            "验证 B->A 通",
        },
    },
}
```

---

## 10. 性能优化

### 10.1 性能目标

| 指标 | 目标值 | 测量方法 |
|------|--------|----------|
| 连接建立时间 | < 3s | 节点加入至可 ping 通 |
| 单隧道吞吐量 | > 500 Mbps | iperf3 TCP |
| 单隧道 UDP 延迟 | < 10ms | 同地域节点间 |
| Coordinator QPS | > 1000 | 节点注册/心跳请求 |

### 10.2 优化措施

| 层面 | 优化项 | 预期收益 |
|------|--------|----------|
| **协议层** | WireGuard 内核态优先 | 吞吐量提升 30% |
| **协议层** | 启用 UDP GSO | 降低 CPU 占用 |
| **网络层** | 多 STUN 服务器 | 穿透率提升 10% |
| **应用层** | 连接池复用 | 降低延迟 |
| **数据层** | PostgreSQL 连接池 (pgbouncer) | 并发提升 10 倍 |
| **数据层** | 批量写入 + COPY | 减少 IO 次数 |

---

## 11. 风险与缓解

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| WireGuard 内核模块缺失 | 中 | 中 | 默认使用 wireguard-go 用户态 |
| 对称 NAT 无法穿透 | 中 | 低 | 自动降级到中继模式 |
| 高并发心跳风暴 | 高 | 低 | 心跳 jitter + 批量处理 |
| 单点故障 | 高 | 低 | 支持 Coordinator 集群部署 |
| 证书/密钥泄露 | 高 | 低 | 支持密钥轮换 + 设备指纹 |
| PostgreSQL 连接失败 | 高 | 低 | 连接池重试 + 健康检查 |
| 数据库迁移兼容 | 中 | 中 | 使用 golang-migrate 管理 schema 版本 |

---

## 12. 附录

### 12.1 项目目录结构

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
│   ├── docker/
│   ├── systemd/
│   └── k8s/
├── docs/                  # 文档
└── test/                  # 测试
    ├── unit/
    ├── integration/
    └── e2e/
```

### 12.2 依赖清单

```go
// go.mod 核心依赖
module github.com/mycel/mesh

go 1.21

require (
    // gRPC
    google.golang.org/grpc v1.60.0
    google.golang.org/protobuf v1.32.0
    github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.0

    // WireGuard
    golang.zx2c4.com/wireguard v0.0.20231106

    // TUN/TAP
    gvisor.dev/gvisor v0.0.20231205

    // CLI
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0

    // Database
    github.com/jackc/pgx/v5 v5.5.0
    github.com/jackc/pgbouncer v1.0.0
    github.com/redis/go-redis/v9 v9.3.0

    // Networking
    github.com/pion/stun/v2 v2.0.0
    golang.org/x/net v0.19.0

    // Logging
    go.uber.org/zap v1.26.0

    // Metrics
    github.com/prometheus/client_golang v1.18.0

    // Testing
    github.com/stretchr/testify v1.8.4
    github.com/testcontainers/testcontainers-go v0.27.0
)
```

---

**文档结束**

---

*本设计文档由 Mycel Mesh 技术团队维护，与 PRD 配套使用。*
