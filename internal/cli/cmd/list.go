package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// listCmd 用于列出网络节点
// 显示当前网络中所有节点的信息
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出网络节点",
	Long:  "显示当前网络中所有节点的信息",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: 实现列表功能
		// 1. 从协调服务器获取节点列表
		// 2. 查询每个节点的状态
		// 3. 格式化输出

		fmt.Printf("NAME            IP          STATUS    LATENCY\n")
		fmt.Printf("node-1          10.0.0.2    online    12ms\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
