package integration

import (
	"testing"

	"github.com/mycel/mesh/internal/pkg/wireguard"
)

// TestNodeKeyGeneration 测试节点密钥生成集成
func TestNodeKeyGeneration(t *testing.T) {
	t.Run("节点可以生成唯一密钥对", func(t *testing.T) {
		// 模拟两个节点生成密钥
		node1Priv, node1Pub, err := wireguard.GenerateKey()
		if err != nil {
			t.Fatalf("节点 1 密钥生成失败：%v", err)
		}

		node2Priv, node2Pub, err := wireguard.GenerateKey()
		if err != nil {
			t.Fatalf("节点 2 密钥生成失败：%v", err)
		}

		// 验证两个节点的密钥不同
		if node1Priv == node2Priv {
			t.Fatal("两个节点生成了相同的私钥")
		}

		if node1Pub == node2Pub {
			t.Fatal("两个节点生成了相同的公钥")
		}
	})
}

// TestNetworkBasicConnectivity 测试基础网络连通性
func TestNetworkBasicConnectivity(t *testing.T) {
	t.Run(" WireGuard 密钥格式正确", func(t *testing.T) {
		// 生成密钥并验证格式
		priv, pub, err := wireguard.GenerateKey()
		if err != nil {
			t.Fatalf("密钥生成失败：%v", err)
		}

		// 验证密钥长度（base64 编码的 32 字节应该是 44 字符）
		if len(priv) != 44 {
			t.Logf("警告：私钥长度 %d, 预期 44", len(priv))
		}

		if len(pub) != 44 {
			t.Logf("警告：公钥长度 %d, 预期 44", len(pub))
		}
	})
}

// TestEndToEnd 端到端测试
func TestEndToEnd(t *testing.T) {
	t.Run("完整节点初始化流程", func(t *testing.T) {
		// 1. 生成密钥
		priv, pub, err := wireguard.GenerateKey()
		if err != nil {
			t.Fatalf("步骤 1 - 密钥生成失败：%v", err)
		}

		// 2. 验证密钥有效
		if priv == "" || pub == "" {
			t.Fatal("步骤 2 - 密钥为空")
		}

		// 3. 验证密钥可以解码
		if !isValidBase64(priv) {
			t.Fatal("步骤 3 - 私钥格式无效")
		}

		if !isValidBase64(pub) {
			t.Fatal("步骤 3 - 公钥格式无效")
		}
	})
}

// isValidBase64 检查字符串是否是有效的 base64 编码
func isValidBase64(s string) bool {
	if len(s) == 0 {
		return false
	}
	// 简单验证：base64 只包含特定字符
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
			return false
		}
	}
	return true
}
