import React, { useState, useEffect } from 'react';
import { Table, Tag, Space, Button, Card, Statistic } from 'antd';
import { 
  CheckCircleOutlined, 
  CloseCircleOutlined,
  ReloadOutlined,
  DeleteOutlined 
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';

// 节点数据类型
interface NodeType {
  key: string;
  id: string;
  name: string;
  private_ip: string;
  public_key: string;
  status: 'online' | 'offline';
  latency_ms?: number;
  last_seen: string;
  rx_bytes: number;
  tx_bytes: number;
}

// 模拟数据 (实际应从 API 获取)
const mockNodes: NodeType[] = [
  {
    key: '1',
    id: 'node-001',
    name: 'dev-server-01',
    private_ip: '10.0.0.2',
    public_key: 'xK7...3Q=',
    status: 'online',
    latency_ms: 12,
    last_seen: new Date().toISOString(),
    rx_bytes: 1024000,
    tx_bytes: 512000,
  },
  {
    key: '2',
    id: 'node-002',
    name: 'prod-server',
    private_ip: '10.0.0.3',
    public_key: 'yL8...4R=',
    status: 'online',
    latency_ms: 45,
    last_seen: new Date().toISOString(),
    rx_bytes: 2048000,
    tx_bytes: 1024000,
  },
  {
    key: '3',
    id: 'node-003',
    name: 'home-pc',
    private_ip: '10.0.0.4',
    public_key: 'zM9...5S=',
    status: 'offline',
    latency_ms: undefined,
    last_seen: new Date(Date.now() - 3600000).toISOString(),
    rx_bytes: 512000,
    tx_bytes: 256000,
  },
];

const Nodes: React.FC = () => {
  const [nodes, setNodes] = useState<NodeType[]>(mockNodes);
  const [loading, setLoading] = useState(false);

  // 格式化流量数据
  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  // 格式化时间
  const formatTime = (isoString: string) => {
    const date = new Date(isoString);
    return date.toLocaleString('zh-CN');
  };

  // 刷新节点列表
  const refreshNodes = async () => {
    setLoading(true);
    // TODO: 调用 API 获取真实数据
    // const response = await fetch('/api/v1/nodes');
    // const data = await response.json();
    // setNodes(data.nodes);
    setLoading(false);
  };

  // 删除节点
  const deleteNode = (id: string) => {
    // TODO: 调用 API 删除节点
    setNodes(nodes.filter(node => node.id !== id));
  };

  // 状态标签
  const statusTag = (status: 'online' | 'offline') => {
    if (status === 'online') {
      return (
        <Tag icon={<CheckCircleOutlined />} color="success">
          在线
        </Tag>
      );
    }
    return (
      <Tag icon={<CloseCircleOutlined />} color="default">
        离线
      </Tag>
    );
  };

  // 延迟显示
  const latencyDisplay = (latency?: number) => {
    if (latency === undefined) return '-';
    if (latency < 20) {
      return <span style={{ color: '#52c41a' }}>{latency} ms</span>;
    } else if (latency < 50) {
      return <span style={{ color: '#faad14' }}>{latency} ms</span>;
    } else {
      return <span style={{ color: '#ff4d4f' }}>{latency} ms</span>;
    }
  };

  // 表格列定义
  const columns: ColumnsType<NodeType> = [
    {
      title: '节点名称',
      dataIndex: 'name',
      key: 'name',
      render: (name, record) => (
        <Space>
          <span>{name}</span>
          <small style={{ color: '#999' }}>{record.private_ip}</small>
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: statusTag,
    },
    {
      title: '延迟',
      dataIndex: 'latency_ms',
      key: 'latency_ms',
      render: latencyDisplay,
    },
    {
      title: '接收流量',
      dataIndex: 'rx_bytes',
      key: 'rx_bytes',
      render: formatBytes,
    },
    {
      title: '发送流量',
      dataIndex: 'tx_bytes',
      key: 'tx_bytes',
      render: formatBytes,
    },
    {
      title: '最后可见',
      dataIndex: 'last_seen',
      key: 'last_seen',
      render: formatTime,
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small">详情</Button>
          <Button 
            type="link" 
            size="small" 
            danger
            icon={<DeleteOutlined />}
            onClick={() => deleteNode(record.id)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  // 统计数据
  const onlineCount = nodes.filter(n => n.status === 'online').length;
  const offlineCount = nodes.filter(n => n.status === 'offline').length;

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <Space size="large">
          <Statistic 
            title="总节点数" 
            value={nodes.length} 
            suffix="个"
          />
          <Statistic 
            title="在线节点" 
            value={onlineCount} 
            suffix="个"
            valueStyle={{ color: '#52c41a' }}
          />
          <Statistic 
            title="离线节点" 
            value={offlineCount} 
            suffix="个"
            valueStyle={{ color: '#ff4d4f' }}
          />
          <Button 
            icon={<ReloadOutlined />} 
            onClick={refreshNodes}
            loading={loading}
          >
            刷新
          </Button>
        </Space>
      </Card>

      <Card title="节点列表">
        <Table 
          columns={columns} 
          dataSource={nodes}
          loading={loading}
          pagination={{ pageSize: 10 }}
        />
      </Card>
    </div>
  );
};

export default Nodes;
