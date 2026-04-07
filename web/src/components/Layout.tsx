import React from 'react';
import { Layout as AntLayout, Menu } from 'antd';
import { 
  DashboardOutlined, 
  TeamOutlined, 
  SettingOutlined 
} from '@ant-design/icons';

const { Header, Content, Sider } = AntLayout;

const Layout: React.FC = () => {
  return (
    <AntLayout style={{ minHeight: '100vh' }}>
      <Header style={{ background: '#fff', padding: '0 24px' }}>
        <h1 style={{ margin: 0, fontSize: 20 }}>Mycel Mesh</h1>
      </Header>
      <AntLayout>
        <Sider width={200} theme="light">
          <Menu mode="inline">
            <Menu.Item key="dashboard" icon={<DashboardOutlined />}>
              仪表盘
            </Menu.Item>
            <Menu.Item key="nodes" icon={<TeamOutlined />}>
              节点管理
            </Menu.Item>
            <Menu.Item key="settings" icon={<SettingOutlined />}>
              设置
            </Menu.Item>
          </Menu>
        </Sider>
        <Content style={{ padding: 24 }}>
          {/* 内容区域占位 */}
          <p>欢迎使用 Mycel Mesh 控制台</p>
        </Content>
      </AntLayout>
    </AntLayout>
  );
};

export default Layout;
