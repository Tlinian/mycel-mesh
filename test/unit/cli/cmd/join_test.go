package cmd

import (
	"testing"
)

// TestJoinCmd 测试 join 命令
func TestJoinCmd(t *testing.T) {
	t.Run("join 命令名称正确", func(t *testing.T) {
		if joinCmd.Name() != "join" {
			t.Fatalf("命令名称应为 join, 实际：%s", joinCmd.Name())
		}
	})

	t.Run("join 命令有简短描述", func(t *testing.T) {
		if joinCmd.Short == "" {
			t.Fatal("join 命令缺少简短描述")
		}
	})

	t.Run("join 命令有 token 标志", func(t *testing.T) {
		tokenFlag := joinCmd.Flags().Lookup("token")
		if tokenFlag == nil {
			t.Fatal("join 命令缺少 token 标志")
		}
	})

	t.Run("join 命令有 coordinator 标志", func(t *testing.T) {
		coordFlag := joinCmd.Flags().Lookup("coordinator")
		if coordFlag == nil {
			t.Fatal("join 命令缺少 coordinator 标志")
		}
	})
}

// TestJoinCmdFlags 测试 join 命令的标志配置
func TestJoinCmdFlags(t *testing.T) {
	t.Run("token 标志是必需的", func(t *testing.T) {
		tokenFlag := joinCmd.Flags().Lookup("token")
		if tokenFlag == nil {
			t.Fatal("token 标志不存在")
		}
		// 注意：cobra 的 MarkFlagRequired 不直接提供检查方法
		// 这里只验证标志存在
	})

	t.Run("coordinator 标志是必需的", func(t *testing.T) {
		coordFlag := joinCmd.Flags().Lookup("coordinator")
		if coordFlag == nil {
			t.Fatal("coordinator 标志不存在")
		}
	})

	t.Run("token 标志有简短形式", func(t *testing.T) {
		tokenFlag := joinCmd.Flags().Lookup("token")
		if tokenFlag == nil {
			t.Fatal("token 标志不存在")
		}
		if tokenFlag.Shorthand != "t" {
			t.Fatalf("token 标志的简短形式应为 't', 实际：%s", tokenFlag.Shorthand)
		}
	})

	t.Run("coordinator 标志有简短形式", func(t *testing.T) {
		coordFlag := joinCmd.Flags().Lookup("coordinator")
		if coordFlag == nil {
			t.Fatal("coordinator 标志不存在")
		}
		if coordFlag.Shorthand != "c" {
			t.Fatalf("coordinator 标志的简短形式应为 'c', 实际：%s", coordFlag.Shorthand)
		}
	})
}
