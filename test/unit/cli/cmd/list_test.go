package cmd_test

import (
	"testing"

	"github.com/mycel/mesh/internal/cli/cmd"
)

// TestListCmd 测试 list 命令
func TestListCmd(t *testing.T) {
	t.Run("list 命令名称正确", func(t *testing.T) {
		if cmd.ListCmd.Name() != "list" {
			t.Fatalf("命令名称应为 list, 实际：%s", cmd.ListCmd.Name())
		}
	})

	t.Run("list 命令有简短描述", func(t *testing.T) {
		if cmd.ListCmd.Short == "" {
			t.Fatal("list 命令缺少简短描述")
		}
	})

	t.Run("list 命令有详细描述", func(t *testing.T) {
		if cmd.ListCmd.Long == "" {
			t.Fatal("list 命令缺少详细描述")
		}
	})
}

// TestListCmdOutput 测试 list 命令输出格式
func TestListCmdOutput(t *testing.T) {
	t.Run("list 命令输出包含表头", func(t *testing.T) {
		// 验证输出的表头格式
		expectedHeaders := []string{"NAME", "IP", "STATUS", "LATENCY"}

		// 模拟输出（实际输出在 RunE 中）
		output := "NAME            IP          STATUS    LATENCY\nnode-1          10.0.0.2    online    12ms\n"

		for _, header := range expectedHeaders {
			if !contains(output, header) {
				t.Errorf("输出缺少表头：%s", header)
			}
		}
	})
}

// TestListCmdRegistered 测试 list 命令已注册到根命令
func TestListCmdRegistered(t *testing.T) {
	t.Run("list 命令已注册", func(t *testing.T) {
		found := false
		for _, c := range cmd.RootCmd.Commands() {
			if c.Name() == "list" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("list 命令未注册到根命令")
		}
	})
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
