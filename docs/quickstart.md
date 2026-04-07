# Mycel Mesh 快速开始指南

**版本**: v1.0.0
**更新时间**: 2026-04-07
**预计时间**: 10 分钟

---

## 一、环境准备

### 1.1 系统要求

| 组件 | 最低要求 | 推荐配置 |
|------|---------|---------|
| **Coordinator** | 1 核 CPU, 512MB 内存 | 2 核 CPU, 1GB 内存 |
| **Agent** | 单核 CPU, 256MB 内存 | 单核 CPU, 512MB 内存 |
| **操作系统** | Linux/macOS/Windows | Linux (Ubuntu 20.04+) |

### 1.2 依赖检查

```bash
# 检查 Go 版本 (编译需要)
go version  # 需要 1.21+

# 检查 WireGuard (运行时自动安装)
wg --version
```

---

## 二、快速部署 (5 分钟)

### 2.1 方式一：源码编译

```bash
# 1. 克隆仓库
git clone https://github.com/mycel/mesh.git
cd mesh

# 2. 编译
make build

# 3. 验证
./bin/mycelctl version
```

### 2.2 方式二：下载预编译二进制

```bash
# Linux
wget https://github.com/mycel/mesh/releases/download/v1.0.0/mycel-linux-amd64.tar.gz
tar xzf mycel-linux-amd64.tar.gz
sudo mv mycelctl /usr/local/bin/

# macOS
brew install mycel-mesh

# Windows
# 从 GitHub Releases 下载 exe 文件
```

---

## 三、启动 Coordinator (2 分钟)

### 3.1 初始化 Coordinator

```bash
# 创建 Coordinator 配置目录
mkdir -p /etc/mycel

# 初始化 Coordinator
./bin/coordinator --init
```

**输出示例**:
```
[INFO] Coordinator initialized
[INFO] Network ID: default-network
[INFO] Network CIDR: 10.0.0.0/16
[INFO] Join Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
[INFO] HTTP API: http://localhost:8080
[INFO] gRPC API: localhost:51820
```

### 3.2 启动 Coordinator 服务

```bash
# 前台运行 (调试用)
./bin/coordinator --config /etc/mycel/config.yaml

# 后台运行 (systemd)
sudo systemctl start mycel-coordinator
sudo systemctl enable mycel-coordinator
```

### 3.3 验证 Coordinator 状态

```bash
curl http://localhost:8080/health
# 返回: {"status":"healthy"}
```

---

## 四、加入节点 (3 分钟)

### 4.1 在节点机器上初始化 Agent

```bash
# 初始化 Agent (首次运行)
mycelctl init --name node-1
```

**输出示例**:
```
[INFO] Generated WireGuard keypair
[INFO] Public Key: xJ8K9mN2pL5qR7sT1vW3yZ4aB6cD8eF0gH2iJ4kL6m=
[INFO] Configuration saved to ~/.mycel/config.yaml
```

### 4.2 加入网络

```bash
# 使用 Coordinator 生成的 token 加入
mycelctl join --token <join-token> --coordinator localhost:51820
```

**输出示例**:
```
[INFO] Connecting to Coordinator...
[INFO] Registered as node-1
[INFO] Assigned IP: 10.0.0.2/16
[INFO] WireGuard interface created: mycel0
[INFO] Connection established!
```

### 4.3 验证连接

```bash
# 查看节点状态
mycelctl status

# 查看网络中的其他节点
mycelctl list

# 测试连接 (ping 其他节点)
ping 10.0.0.3
```

---

## 五、配置网络

### 5.1 查看网络拓扑

```bash
# 查看所有节点
mycelctl list --output table

# 查看节点详情
mycelctl node get node-1
```

**输出示例**:
```
ID       NAME     STATUS    IP          LATENCY    UPTIME
----------------------------------------------------------------
node-1   node-1   online    10.0.0.2    5ms        1h 23m
node-2   node-2   online    10.0.0.3    12ms       45m
node-3   node-3   offline   10.0.0.4    -          -
```

### 5.2 配置 ACL 规则

```bash
# 添加允许规则 (node-1 可以访问 node-2 的 80 端口)
mycelctl acl add \
  --source node-1 \
  --dest node-2 \
  --action allow \
  --ports 80,443 \
  --protocol tcp

# 查看 ACL 规则
mycelctl acl list

# 添加拒绝规则
mycelctl acl add \
  --source node-3 \
  --dest node-1 \
  --action deny

# 删除 ACL 规则
mycelctl acl remove --id <rule-id>
```

### 5.3 创建子网 (可选)

```bash
# 创建开发子网
mycelctl subnet create \
  --name dev-subnet \
  --cidr 10.1.0.0/24 \
  --description "Development subnet"

# 查看子网
mycelctl subnet list

# 将节点加入子网
mycelctl node update node-1 --subnet dev-subnet
```

---

## 六、使用 Web UI

### 6.1 访问 Web UI

打开浏览器访问：http://localhost:8080

### 6.2 登录

- **用户名**: admin
- **密码**: (在 Coordinator 配置中设置)

### 6.3 主要功能

| 页面 | 功能 |
|------|------|
| **节点管理** | 查看所有节点状态、延迟、流量 |
| **ACL 配置** | 可视化配置访问控制规则 |
| **子网管理** | 创建和管理子网 |
| **监控面板** | 查看流量图表、告警信息 |

---

## 七、配置监控告警

### 7.1 启动 Prometheus

```bash
# 使用 Docker 启动
docker run -d \
  -p 9090:9090 \
  -v $(pwd)/deploy/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus:v2.45.0
```

### 7.2 启动 Grafana

```bash
# 使用 Docker 启动
docker run -d \
  -p 3000:3000 \
  -v $(pwd)/deploy/grafana/dashboards:/etc/grafana/provisioning/dashboards \
  grafana/grafana:10.0.0
```

### 7.3 导入 Dashboard

1. 访问 http://localhost:3000
2. 登录 (admin/admin)
3. 导入 `deploy/grafana/dashboards/` 下的 JSON 文件

---

## 八、故障排查

### 8.1 节点无法加入网络

```bash
# 检查 Coordinator 是否运行
systemctl status mycel-coordinator

# 检查 token 是否有效
mycelctl token verify <token>

# 检查防火墙
sudo ufw status
```

### 8.2 节点间无法通信

```bash
# 检查 WireGuard 接口
ip link show mycel0

# 检查路由
ip route show

# 检查 NAT 穿透状态
mycelctl nat status
```

### 8.3 查看日志

```bash
# Coordinator 日志
journalctl -u mycel-coordinator -f

# Agent 日志
journalctl -u mycel-agent -f

# CLI 调试模式
mycelctl --debug list
```

---

## 九、下一步

- [ ] 阅读 [产品白皮书](./products/whitepaper.md) 了解更多功能
- [ ] 查看 [API 文档](./api.md) 了解完整 API
- [ ] 访问 [FAQ](./faq.md) 解决常见问题
- [ ] 加入社区 https://github.com/mycel/mesh/discussions

---

## 十、常用命令速查

```bash
# Coordinator 管理
mycelctl coordinator init          # 初始化 Coordinator
mycelctl coordinator status        # 查看状态
mycelctl coordinator config        # 查看配置

# 节点管理
mycelctl init --name <name>        # 初始化节点
mycelctl join --token <token>      # 加入网络
mycelctl leave                     # 离开网络
mycelctl list                      # 列出节点
mycelctl status                    # 查看状态

# ACL 管理
mycelctl acl add                   # 添加规则
mycelctl acl list                  # 列出规则
mycelctl acl remove --id <id>      # 删除规则

# 子网管理
mycelctl subnet create             # 创建子网
mycelctl subnet list               # 列出子网
mycelctl subnet delete --name <n>  # 删除子网

# 诊断
mycelctl nat status                # NAT 状态
mycelctl ping <node>               # 测试连接
mycelctl logs                      # 查看日志
```

---

**快速开始完成!** 🎉

现在你已经成功部署和使用 Mycel Mesh。如有问题，请查阅文档或联系社区。
