package cmd_test

import (
	"encoding/base64"
	"testing"

	"github.com/mycel/mesh/internal/cli/cmd"
	"github.com/mycel/mesh/internal/pkg/wireguard"
)

// TestInitCmd 测试 init 命令
func TestInitCmd(t *testing.T) {
	t.Run("init 命令名称正确", func(t *testing.T) {
		if cmd.InitCmd.Name() != "init" {
			t.Fatalf("命令名称应为 init, 实际：%s", cmd.InitCmd.Name())
		}
	})

	t.Run("init 命令有简短描述", func(t *testing.T) {
		if cmd.InitCmd.Short == "" {
			t.Fatal("init 命令缺少简短描述")
		}
	})

	t.Run("init 命令有 name 标志", func(t *testing.T) {
		nameFlag := cmd.InitCmd.Flags().Lookup("name")
		if nameFlag == nil {
			t.Fatal("init 命令缺少 name 标志")
		}
	})
}

// TestInitCmdWithValidName 测试带有效名称的 init 命令
func TestInitCmdWithValidName(t *testing.T) {
	t.Run("密钥生成有效", func(t *testing.T) {
		// 验证 wireguard 密钥生成有效
		priv, pub, err := wireguard.GenerateKey()
		if err != nil {
			t.Fatalf("密钥生成失败：%v", err)
		}

		if priv == "" {
			t.Fatal("私钥为空")
		}

		if pub == "" {
			t.Fatal("公钥为空")
		}

		// 验证公钥是有效的 base64
		if !isValidBase64(pub) {
			t.Fatal("公钥不是有效的 base64 编码")
		}
	})
}

// TestInitCmdFlags 测试 init 命令的标志
func TestInitCmdFlags(t *testing.T) {
	t.Run("name 标志存在", func(t *testing.T) {
		nameFlag := cmd.InitCmd.Flags().Lookup("name")
		if nameFlag == nil {
			t.Fatal("name 标志不存在")
		}
		if nameFlag.Usage == "" {
			t.Fatal("name 标志缺少使用说明")
		}
	})
}

// isValidBase64 检查字符串是否是有效的 base64 编码
func isValidBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}
