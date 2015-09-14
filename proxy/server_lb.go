// Copyright 2015 Spring Rain Software Compnay LTD. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.
package proxy

import (
	"fmt"
	thrift "git.apache.org/thrift.git/lib/go/thrift"
	utils "github.com/wfxiang08/rpc_proxy/utils"
	"github.com/wfxiang08/rpc_proxy/utils/atomic2"
	"github.com/wfxiang08/rpc_proxy/utils/log"
	zk "github.com/wfxiang08/rpc_proxy/zk"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

//
// Thrift Server的参数
//
type ThriftLoadBalanceServer struct {
	productName     string
	serviceName     string
	frontendAddr    string // 绑定的端口
	backendAddr     string
	lbServiceName   string
	topo            *zk.Topology // ZK相关
	zkAddr          string
	verbose         bool
	backendService  *BackServiceLB
	exitEvt         chan bool
	lastRequestTime atomic2.Int64
	config          *utils.Config
}

func NewThriftLoadBalanceServer(config *utils.Config) *ThriftLoadBalanceServer {
	log.Printf("FrontAddr: %s\n", Magenta(config.FrontendAddr))

	// 前端对接rpc_proxy
	p := &ThriftLoadBalanceServer{
		config:       config,
		zkAddr:       config.ZkAddr,
		productName:  config.ProductName,
		serviceName:  config.Service,
		frontendAddr: config.FrontendAddr,
		backendAddr:  config.BackAddr,
		verbose:      config.Verbose,
		exitEvt:      make(chan bool),
	}

	p.topo = zk.NewTopology(p.productName, p.zkAddr)
	p.lbServiceName = GetServiceIdentity(p.frontendAddr)

	// 后端对接: 各种python的rpc server
	p.backendService = NewBackServiceLB(p.serviceName, p.backendAddr, p.verbose, p.exitEvt)
	return p

}

func (p *ThriftLoadBalanceServer) Run() {
	//	// 1. 创建到zk的连接

	// 127.0.0.1:5555 --> 127_0_0_1:5555

	exitSignal := make(chan os.Signal, 1)

	signal.Notify(exitSignal, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	// syscall.SIGKILL
	// kill -9 pid
	// kill -s SIGKILL pid 还是留给运维吧
	//

	// 注册服务
	evtExit := make(chan interface{})
	serviceEndpoint := RegisterService(p.serviceName, p.frontendAddr, p.lbServiceName,
		p.topo, evtExit, p.config.WorkDir, p.config.CodeUrlVersion)

	//	var suideTime time.Time

	//	isAlive := true

	// 3. 读取后端服务的配置
	var transport thrift.TServerTransport
	var err error

	isUnixDomain := false
	// 127.0.0.1:9999(以:区分不同的类型)
	if !strings.Contains(p.frontendAddr, ":") {
		if FileExist(p.frontendAddr) {
			os.Remove(p.frontendAddr)
		}
		transport, err = NewTServerUnixDomain(p.frontendAddr)
		isUnixDomain = true
	} else {
		transport, err = thrift.NewTServerSocket(p.frontendAddr)
	}

	if err != nil {
		log.ErrorErrorf(err, "Server Socket Create Failed: %v", err)
		panic(fmt.Sprintf("Invalid FrontendAddress: %s", p.frontendAddr))
	}

	err = transport.Listen()
	if err != nil {
		log.ErrorErrorf(err, "Server Socket Create Failed: %v", err)
		panic(fmt.Sprintf("Binding Error FrontendAddress: %s", p.frontendAddr))
	}

	ch := make(chan thrift.TTransport, 4096)
	defer close(ch)

	// 强制退出? TODO: Graceful退出
	go func() {
		<-exitSignal

		// 通知RegisterService终止循环
		evtExit <- true
		log.Info(Green("Receive Exit Signals...."))
		serviceEndpoint.DeleteServiceEndpoint(p.topo)

		start := time.Now().Unix()
		for true {
			// 如果5s内没有接受到新的请求了，则退出
			now := time.Now().Unix()
			if now-p.lastRequestTime.Get() > 5 {
				log.Printf(Red("[%s]Graceful Exit..."), p.serviceName)
				break
			} else {
				log.Printf(Cyan("[%s]Sleeping %d seconds before Exit...\n"),
					p.serviceName, now-start)
				time.Sleep(time.Second)
			}
		}

		transport.Interrupt()
		transport.Close()
	}()

	go func() {
		var address string
		for c := range ch {
			// 为每个Connection建立一个Session
			socket, ok := c.(SocketAddr)

			if ok {
				if isUnixDomain {
					address = p.frontendAddr
				} else {
					address = socket.Addr().String()
				}
			} else {
				address = "unknow"
			}
			x := NewNonBlockSession(c, address, p.verbose, &p.lastRequestTime)
			// Session独立处理自己的请求
			go x.Serve(p.backendService, 1000)
		}
	}()

	// Accept什么时候出错，出错之后如何处理呢?
	for {
		c, err := transport.Accept()
		if err != nil {
			close(ch)
			break
		} else {
			ch <- c
		}
	}
}
