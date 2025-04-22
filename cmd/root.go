package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "sshtalk",
	Short: "SSHTalk - Chat via terminal using SSH",
	Long:  `SSHTalk is a terminal application that allows you to chat via SSH.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 默认模式 (直接 TUI 模式)
		// 这个函数实现在 ui/tui.go 中
		startLocalUI()
	},
}

func init() {
	// 移除全局 PORT 检查，改为在子命令中检查

	// 添加子命令
	rootCmd.AddCommand(sshCmd)
	rootCmd.AddCommand(httpCmd)
}

// SSH 服务器子命令
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Run as SSH server",
	Run: func(cmd *cobra.Command, args []string) {
		// 检查 PORT 环境变量
		if os.Getenv("PORT") == "" {
			log.Fatal("PORT is not set for SSH server mode")
		}
		log.Println("Starting in SSH server mode...")
		startSSHServer()
	},
}

// HTTP 服务器子命令
var httpCmd = &cobra.Command{
	Use:   "http",
	Short: "Run as HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		// 检查 PORT 环境变量
		if os.Getenv("PORT") == "" {
			log.Fatal("PORT is not set for HTTP server mode")
		}
		log.Println("Starting in HTTP server mode...")
		startHttpServer()
	},
}
