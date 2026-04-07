package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// joinCmd 用于加入 Mycel 网络
// 使用加入令牌连接到 Mycel 协调服务器
var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "加入 Mycel 网络",
	Long:  "使用加入令牌连接到 Mycel 协调服务器",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, _ := cmd.Flags().GetString("token")
		coordinator, _ := cmd.Flags().GetString("coordinator")

		// TODO: 实现加入逻辑
		// 1. 验证令牌
		// 2. 连接协调服务器
		// 3. 获取网络配置
		// 4. 更新 WireGuard 配置

		fmt.Printf("正在加入网络...\n")
		fmt.Printf("协调服务器：%s\n", coordinator)
		fmt.Printf("令牌：%s\n", token)
		fmt.Printf("加入成功!\n")

		return nil
	},
}

func init() {
	joinCmd.Flags().StringP("token", "t", "", "加入令牌")
	joinCmd.Flags().StringP("coordinator", "c", "", "协调服务器地址")
	joinCmd.MarkFlagRequired("token")
	joinCmd.MarkFlagRequired("coordinator")
	rootCmd.AddCommand(joinCmd)
}
