package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd 是 mycelctl 的根命令
var rootCmd = &cobra.Command{
	Use:   "mycelctl",
	Short: "Mycel Mesh CLI 工具",
	Long:  "Mycel Mesh 虚拟组网工具的命令行管理界面",
}

// Execute 执行根命令，处理错误并退出
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
