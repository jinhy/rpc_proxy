// Copyright 2015 Spring Rain Software Compnay LTD. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.
package proxy

import (
	"fmt"
	thrift "git.apache.org/thrift.git/lib/go/thrift"
	utils "github.com/wfxiang08/rpc_proxy/utils"
	"github.com/wfxiang08/rpc_proxy/utils/log"
	zk "github.com/wfxiang08/rpc_proxy/zk"
	"os"
	"strings"
	"time"
)

const (
	HEARTBEAT_INTERVAL = 1000 * time.Millisecond
)

type ProxyServer struct {
	productName string
	proxyAddr   string
	zkAdresses  string
	topo        *zk.Topology
	verbose     bool
	profile     bool
	router      *Router
}

func NewProxyServer(config *utils.Config) *ProxyServer {
	p := &ProxyServer{
		productName: config.ProductName,
		proxyAddr:   config.ProxyAddr,
		zkAdresses:  config.ZkAddr,
		verbose:     config.Verbose,
		profile:     config.Profile,
	}
	p.topo = zk.NewTopology(p.productName, p.zkAdresses)
	p.router = NewRouter(p.productName, p.topo, p.verbose)
	return p
}

//
// 两参数是必须的:  ProductName, zkAddress, frontAddr可以用来测试
//
func (p *ProxyServer) Run() {

	var transport thrift.TServerTransport
	var err error

	log.Printf(Magenta("Start Proxy at Address: %s"), p.proxyAddr)
	// 读取后端服务的配置
	isUnixDomain := false
	if !strings.Contains(p.proxyAddr, ":") {
		if FileExist(p.proxyAddr) {
			os.Remove(p.proxyAddr)
		}
		transport, err = NewTServerUnixDomain(p.proxyAddr)
		isUnixDomain = true
	} else {
		transport, err = thrift.NewTServerSocket(p.proxyAddr)
	}
	if err != nil {
		log.ErrorErrorf(err, "Server Socket Create Failed: %v, Front: %s", err, p.proxyAddr)
	}

	// 开始监听
	//	transport.Open()
	transport.Listen()

	ch := make(chan thrift.TTransport, 4096)
	defer close(ch)

	go func() {
		var address string
		for c := range ch {
			// 为每个Connection建立一个Session
			socket, ok := c.(SocketAddr)
			if isUnixDomain {
				address = p.proxyAddr
			} else if ok {
				address = socket.Addr().String()
			} else {
				address = "unknow"
			}
			x := NewSession(c, address, p.verbose)
			// Session独立处理自己的请求
			go x.Serve(p.router, 1000)
		}
	}()

	// Accept什么时候出错，出错之后如何处理呢?
	for {
		c, err := transport.Accept()
		if err != nil {
			log.ErrorErrorf(err, "Accept Error: %v", err)
			break
		} else {
			ch <- c
		}
	}
}

func printList(msgs []string) string {
	results := make([]string, 0, len(msgs))
	results = append(results, fmt.Sprintf("Msgs Len: %d, ", len(msgs)-1))
	for i := 0; i < len(msgs)-1; i++ {
		results = append(results, fmt.Sprintf("[%s]", msgs[i]))
	}
	return strings.Join(results, ",")
}
