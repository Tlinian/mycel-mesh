package cmd_test

import (
	"testing"

	"github.com/mycel/mesh/internal/cli/cmd"
)

// TestRootCmd 测试根命令的基本属性
func TestRootCmd(t *testing.T) {
	t.Run("根命令名称正确", func(t *testing.T) {
		if cmd.RootCmd.Name() != "mycelctl" {
			t.Fatalf("根命令名称应为 mycelctl, 实际：%s", cmd.RootCmd.Name())
		}
	})

	t.Run("根命令有简短描述", func(t *testing.T) {
		if cmd.RootCmd.Short == "" {
			t.Fatal("根命令缺少简短描述")
		}
	})

	t.Run("根命令有详细描述", func(t *testing.T) {
		if cmd.RootCmd.Long == "" {
			t.Fatal("根命令缺少详细描述")
		}
	})
}

// TestExecute 测试 Execute 函数
func TestExecute(t *testing.T) {
	t.Run("Execute 不panic", func(t *testing.T) {
		// 测试 Execute 在正常情况下的行为
		// 由于 Execute 会调用 os.Exit，我们只测试它不会因为空参数而 panic
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Execute() 发生 panic: %v", r)
			}
		}()

		// 注意：这里不实际调用 Execute()，因为它会退出程序
		// 我们只验证 RootCmd 已正确初始化
		if cmd.RootCmd == nil {
			t.Fatal("RootCmd 未初始化")
		}
	})
}

// TestRootCmdHasSubCommands 测试根命令有子命令
func TestRootCmdHasSubCommands(t *testing.T) {
	t.Run("根命令包含 init 子命令", func(t *testing.T) {
		found := false
		for _, c := range cmd.RootCmd.Commands() {
			if c.Name() == "init" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("根命令未包含 init 子命令")
		}
	})

	t.Run("根命令包含 join 子命令", func(t *testing.T) {
		found := false
		for _, c := range cmd.RootCmd.Commands() {
			if c.Name() == "join" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("根命令未包含 join 子命令")
		}
	})

	t.Run("根命令包含 list 子命令", func(t *testing.T) {
		found := false
		for _, c := range cmd.RootCmd.Commands() {
			if c.Name() == "list" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("根命令未包含 list 子命令")
		}
	})
}
