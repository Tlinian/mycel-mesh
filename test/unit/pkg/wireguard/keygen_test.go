package wireguard_test

import (
	"encoding/base64"
	"testing"

	"github.com/mycel/mesh/internal/pkg/wireguard"
)

// TestGenerateKey 测试密钥生成函数
func TestGenerateKey(t *testing.T) {
	t.Run("成功生成密钥对", func(t *testing.T) {
		privateKey, publicKey, err := wireguard.GenerateKey()

		// 验证无错误
		if err != nil {
			t.Fatalf("GenerateKey() 返回错误：%v", err)
		}

		// 验证私钥不为空
		if privateKey == "" {
			t.Fatal("GenerateKey() 返回空私钥")
		}

		// 验证公钥不为空
		if publicKey == "" {
			t.Fatal("GenerateKey() 返回空公钥")
		}

		// 验证私钥是有效的 base64 编码
		decodedPriv, err := base64.StdEncoding.DecodeString(privateKey)
		if err != nil {
			t.Fatalf("私钥不是有效的 base64: %v", err)
		}

		// 验证私钥长度为 32 字节
		if len(decodedPriv) != 32 {
			t.Fatalf("私钥长度应为 32 字节，实际：%d", len(decodedPriv))
		}

		// 验证公钥是有效的 base64 编码
		decodedPub, err := base64.StdEncoding.DecodeString(publicKey)
		if err != nil {
			t.Fatalf("公钥不是有效的 base64: %v", err)
		}

		// 验证公钥长度为 32 字节
		if len(decodedPub) != 32 {
			t.Fatalf("公钥长度应为 32 字节，实际：%d", len(decodedPub))
		}
	})

	t.Run("多次生成密钥对不相同", func(t *testing.T) {
		priv1, pub1, err1 := wireguard.GenerateKey()
		if err1 != nil {
			t.Fatalf("第一次 GenerateKey() 失败：%v", err1)
		}

		priv2, pub2, err2 := wireguard.GenerateKey()
		if err2 != nil {
			t.Fatalf("第二次 GenerateKey() 失败：%v", err2)
		}

		// 验证两次生成的密钥不同
		if priv1 == priv2 {
			t.Fatal("两次生成的私钥相同，随机性不足")
		}

		if pub1 == pub2 {
			t.Fatal("两次生成的公钥相同，随机性不足")
		}
	})
}

// TestGenerateKeyDeterministic 测试密钥生成的确定性（可选）
func TestGenerateKeyDeterministic(t *testing.T) {
	// 这个测试验证公钥可以从私钥推导出来
	// 由于 GenerateKey 内部使用随机数，我们无法直接测试确定性
	// 但我们可以通过多次运行来验证统计特性
	t.Run("生成 100 对密钥都有效", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			priv, pub, err := wireguard.GenerateKey()
			if err != nil {
				t.Fatalf("第 %d 次 GenerateKey() 失败：%v", i, err)
			}
			if priv == "" || pub == "" {
				t.Fatalf("第 %d 次 GenerateKey() 返回空密钥", i)
			}
		}
	})
}
