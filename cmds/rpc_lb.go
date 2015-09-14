// Copyright 2015 Spring Rain Software Company LTD. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.
package main

import (
	proxy "github.com/wfxiang08/rpc_proxy/proxy"
	utils "github.com/wfxiang08/rpc_proxy/utils"
)

const (
	BINARY_NAME  = "rpc_lb"
	SERVICE_DESC = "Chunyu RPC Load Balance Service"
)

var (
	buildDate  string
	gitVersion string
)

func main() {
	proxy.RpcMain(BINARY_NAME, SERVICE_DESC,
		// 验证LB的配置
		proxy.ConfigCheckRpcLB,
		// 根据配置创建一个Server
		func(config *utils.Config) proxy.Server {
			return proxy.NewThriftLoadBalanceServer(config)
		}, buildDate, gitVersion)

}
