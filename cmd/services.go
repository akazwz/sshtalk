package cmd

import (
	httpServer "sshtalk/server/http"
	sshServer "sshtalk/server/ssh"
	"sshtalk/ui"
)

// 启动本地 UI 的适配器
func startLocalUI() {
	ui.StartLocalUI()
}

// SSH 服务器适配器
func startSSHServer() {
	sshServer.Start()
}

// HTTP 服务器适配器
func startHttpServer() {
	httpServer.Start()
}
