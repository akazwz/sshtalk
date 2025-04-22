package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

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
		startLocalUI()
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
	rootCmd.AddCommand(httpCmd)
}

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Run as SSH server",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getenv("PORT") == "" {
			log.Fatal("PORT is not set for SSH server mode")
		}
		log.Println("Starting in SSH server mode...")
		startSSHServer()
	},
}

var httpCmd = &cobra.Command{
	Use:   "http",
	Short: "Run as HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getenv("PORT") == "" {
			log.Fatal("PORT is not set for HTTP server mode")
		}
		log.Println("Starting in HTTP server mode...")
		startHttpServer()
	},
}
