// Copyright 2015 Spring Rain Software Company LTD. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.
package main

import (
	proxy "github.com/wfxiang08/rpc_proxy/proxy"
	utils "github.com/wfxiang08/rpc_proxy/utils"
)

const (
	BINARY_NAME  = "rpc_proxy"
	SERVICE_DESC = "Chunyu RPC Local Proxy v0.1"
)

var (
	gitVersion string
	buildDate  string
)

func main() {
	proxy.RpcMain(BINARY_NAME, SERVICE_DESC,
		// 验证LB的配置
		proxy.ConfigCheckRpcProxy,
		// 根据配置创建一个Server
		func(config *utils.Config) proxy.Server {
			// 正式的服务
			return proxy.NewProxyServer(config)
		}, buildDate, gitVersion)

}
