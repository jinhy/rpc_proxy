// Copyright 2015 Spring Rain Software Company LTD. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.
package proxy

import (
	thrift "git.apache.org/thrift.git/lib/go/thrift"
	"github.com/wfxiang08/rpc_proxy/utils/log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

//
// Proxy中用来和后端服务通信的模块
//
type BackServiceLB struct {
	serviceName string
	backendAddr string

	// Mutex的使用：
	// 	避免使用匿名的Mutex, 需要指定一个语义明确的变量，限定它的使用范围(另可多定义几个Mutex, 不能滥用)
	//
	// 同时保护: activeConns 和 currentConnIndex
	activeConnsLock  sync.Mutex
	activeConns      []*BackendConnLB // 每一个BackendConn应该有一定的高可用保障
	currentConnIndex int

	verbose bool
	exitEvt chan bool
	ch      chan thrift.TTransport
}

// 创建一个BackService
func NewBackServiceLB(serviceName string, backendAddr string, verbose bool,
	exitEvt chan bool) *BackServiceLB {

	service := &BackServiceLB{
		serviceName:      serviceName,
		backendAddr:      backendAddr,
		activeConns:      make([]*BackendConnLB, 0, 10),
		verbose:          verbose,
		exitEvt:          exitEvt,
		currentConnIndex: 0,
		ch:               make(chan thrift.TTransport, 4096),
	}

	service.run()
	return service

}

//
// 后端如何处理一个Request, 处理完毕之后直接返回，因为Caller已经做好异步处理了
//
func (s *BackServiceLB) Dispatch(r *Request) error {
	backendConn := s.nextBackendConn()

	r.Service = s.serviceName

	if backendConn == nil {
		// 没有后端服务
		if s.verbose {
			log.Printf(Red("[%s]No BackSocket Found: %s"),
				s.serviceName, r.Request.Name)
		}
		// 从errMsg来构建异常
		errMsg := GetWorkerNotFoundData(r, "BackServiceLB")
		//		log.Printf(Magenta("---->Convert Error Back to Exception:[%d] %s\n"), len(errMsg), string(errMsg))
		r.Response.Data = errMsg

		return nil
	} else {
		//		if s.verbose {
		//			log.Println("SendMessage With: ", backendConn.Addr4Log(), "For Service: ", s.serviceName)
		//		}
		backendConn.PushBack(r)

		r.Wait.Wait()

		return nil
	}
}

func (s *BackServiceLB) run() {
	go func() {
		// 定时汇报当前的状态
		for true {
			log.Printf(Green("[Report]: %s --> %d workers, coroutine: %d"),
				s.serviceName, s.Active(), runtime.NumGoroutine())
			time.Sleep(time.Second * 10)
		}
	}()

	var transport thrift.TServerTransport
	var err error

	// 3. 读取后端服务的配置
	isUnixDomain := false
	// 127.0.0.1:9999(以:区分不同的类型)
	if !strings.Contains(s.backendAddr, ":") {
		if FileExist(s.backendAddr) {
			os.Remove(s.backendAddr)
		}
		transport, err = NewTServerUnixDomain(s.backendAddr)
		isUnixDomain = true
	} else {
		transport, err = thrift.NewTServerSocket(s.backendAddr)
	}

	if err != nil {
		log.ErrorErrorf(err, "[%s]Server Socket Create Failed: %v", s.serviceName, err)
		panic("BackendAddr Invalid")
	}

	err = transport.Listen()
	if err != nil {
		log.ErrorErrorf(err, "[%s]Server Socket Open Failed: %v", s.serviceName, err)
		panic("Server Socket Open Failed")
	}

	// 和transport.open做的事情一样，如果Open没错，则Listen也不会有问题

	log.Printf(Green("[%s]LB Backend Services listens at: %s"), s.serviceName, s.backendAddr)

	s.ch = make(chan thrift.TTransport, 4096)

	// 强制退出? TODO: Graceful退出
	go func() {
		<-s.exitEvt
		log.Info(Red("Receive Exit Signals...."))
		transport.Interrupt()
		transport.Close()
	}()

	go func() {
		var backendAddr string
		for trans := range s.ch {
			// 为每个Connection建立一个Session
			socket, ok := trans.(SocketAddr)
			if ok {
				if isUnixDomain {
					backendAddr = s.backendAddr
				} else {
					backendAddr = socket.Addr().String()
				}

				conn := NewBackendConnLB(trans, s.serviceName, backendAddr, s, s.verbose)

				// 因为连接刚刚建立，可靠性还是挺高的，因此直接加入到列表中
				s.activeConnsLock.Lock()
				conn.Index = len(s.activeConns)
				s.activeConns = append(s.activeConns, conn)
				s.activeConnsLock.Unlock()

				log.Printf(Green("%s --> %d workers"), s.serviceName, conn.Index)
			} else {
				panic("Invalid Socket Type")
			}

		}
	}()

	// Accept什么时候出错，出错之后如何处理呢?
	go func() {
		for {
			c, err := transport.Accept()
			if err != nil {
				return
			} else {
				s.ch <- c
			}
		}
	}()
}

func (s *BackServiceLB) Active() int {
	s.activeConnsLock.Lock()
	defer s.activeConnsLock.Unlock()
	return len(s.activeConns)
}

// 获取下一个active状态的BackendConn
func (s *BackServiceLB) nextBackendConn() *BackendConnLB {
	s.activeConnsLock.Lock()
	defer s.activeConnsLock.Unlock()

	// TODO: 暂时采用RoundRobin的方法，可以采用其他具有优先级排列的方法
	var backSocket *BackendConnLB

	if len(s.activeConns) == 0 {
		if s.verbose {
			log.Printf(Cyan("[%s]ActiveConns Len 0"), s.serviceName)
		}
		backSocket = nil
	} else {
		if s.currentConnIndex >= len(s.activeConns) {
			s.currentConnIndex = 0
		}
		backSocket = s.activeConns[s.currentConnIndex]
		s.currentConnIndex++
		if s.verbose {
			log.Printf(Cyan("[%s]ActiveConns Len %d, CurrentIndex: %d"), s.serviceName,
				len(s.activeConns), s.currentConnIndex)
		}
	}
	return backSocket
}

// 只有在conn出现错误时才会调用
func (s *BackServiceLB) StateChanged(conn *BackendConnLB) {
	s.activeConnsLock.Lock()
	defer s.activeConnsLock.Unlock()

	log.Printf(Green("[%s]StateChanged: %s, Index: %d, Count: %d"), conn.serviceName, conn.addr4Log, conn.Index, len(s.activeConns))
	if conn.IsConnActive.Get() {
		// BackServiceLB 只有一个状态转移: Active --> Not Active
		log.Printf(Magenta("Unexpected BackendConnLB State"))
		if s.verbose {
			panic("Unexpected BackendConnLB State")
		}
	} else {
		log.Printf(Red("Remove BackendConn From activeConns: %s, Index: %d, Count: %d"),
			conn.Addr4Log(), conn.Index, len(s.activeConns))

		// 从数组中删除一个元素(O(1)的操作)
		if conn.Index != INVALID_ARRAY_INDEX {
			// 1. 和最后一个元素进行交换
			lastIndex := len(s.activeConns) - 1
			if lastIndex != conn.Index {
				lastConn := s.activeConns[lastIndex]

				// 将最后一个元素和当前的元素交换位置
				s.activeConns[conn.Index] = lastConn
				lastConn.Index = conn.Index

				// 删除引用
				s.activeConns[lastIndex] = nil
				conn.Index = INVALID_ARRAY_INDEX

			}
			log.Printf(Red("Remove BackendConn From activeConns: %s"), conn.Addr4Log())

			// 2. slice
			s.activeConns = s.activeConns[0:lastIndex]

		}
	}
}
