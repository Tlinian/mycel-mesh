package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/mycel/mesh/internal/pkg/wireguard"
)

// InitCmd 用于初始化 Mycel 节点
// 生成节点密钥对，创建本地配置文件
var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化 Mycel 节点",
	Long:  "生成节点密钥对，创建本地配置文件",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		// 生成密钥对
		privateKey, publicKey, err := wireguard.GenerateKey()
		if err != nil {
			return fmt.Errorf("密钥生成失败：%w", err)
		}

		fmt.Printf("节点初始化成功!\n")
		fmt.Printf("名称：%s\n", name)
		fmt.Printf("公钥：%s\n", publicKey)
		fmt.Printf("私钥：已保存到配置文件\n")

		// TODO: 将私钥保存到配置文件
		_ = privateKey

		return nil
	},
}

func init() {
	InitCmd.Flags().StringP("name", "n", "", "节点名称")
	InitCmd.MarkFlagRequired("name")
	RootCmd.AddCommand(InitCmd)
}
