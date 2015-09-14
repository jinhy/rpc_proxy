// Copyright 2015 Spring Rain Software Company LTD. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.
package proxy

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"git.apache.org/thrift.git/lib/go/thrift"
)

// 判断给定的文件是否存在
func FileExist(file string) bool {
	var err error
	_, err = os.Stat(file)
	return !os.IsNotExist(err)
}

type SocketAddr interface {
	Addr() net.Addr
}

type TServerUnixDomain struct {
	listener      net.Listener
	addr          net.Addr
	clientTimeout time.Duration

	// Protects the interrupted value to make it thread safe.
	mu          sync.RWMutex
	interrupted bool
}

func NewTServerUnixDomain(listenAddr string) (*TServerUnixDomain, error) {
	return NewTServerUnixDomainTimeout(listenAddr, 0)
}

func NewTServerUnixDomainTimeout(listenAddr string,
	clientTimeout time.Duration) (*TServerUnixDomain, error) {

	addr, err := net.ResolveUnixAddr("unix", listenAddr)
	if err != nil {
		return nil, err
	}
	return &TServerUnixDomain{addr: addr, clientTimeout: clientTimeout}, nil
}

func (p *TServerUnixDomain) Listen() error {
	if p.IsListening() {
		return nil
	}
	l, err := net.Listen(p.addr.Network(), p.addr.String())
	if err != nil {
		return err
	}
	p.listener = l

	fmt.Println(p.addr.Network(), p.addr.String())

	// 注意: 该Socket需要给所有需要访问该接口的人以读写的权限
	// 因此最终的 sock文件的权限为: 0777
	// 例如: aa.sock root/root 07777
	//      换一个用户，rm aa.sock 似乎无效
	filePath := p.addr.String()
	os.Chmod(filePath, os.ModePerm)

	return nil
}

func (p *TServerUnixDomain) Accept() (thrift.TTransport, error) {
	p.mu.RLock()
	interrupted := p.interrupted
	p.mu.RUnlock()

	if interrupted {
		return nil, errors.New("Transport Interrupted")
	}
	if p.listener == nil {
		return nil, thrift.NewTTransportException(thrift.NOT_OPEN,
			"No underlying server socket")
	}
	conn, err := p.listener.Accept()
	if err != nil {
		return nil, thrift.NewTTransportExceptionFromError(err)
	}
	return thrift.NewTSocketFromConnTimeout(conn, p.clientTimeout), nil
}

// Checks whether the socket is listening.
func (p *TServerUnixDomain) IsListening() bool {
	return p.listener != nil
}

// Connects the socket, creating a new socket object if necessary.
func (p *TServerUnixDomain) Open() error {
	if p.IsListening() {
		return thrift.NewTTransportException(thrift.ALREADY_OPEN,
			"Server socket already open")
	}
	if l, err := net.Listen(p.addr.Network(), p.addr.String()); err != nil {
		return err
	} else {
		p.listener = l
	}
	return nil
}

func (p *TServerUnixDomain) Addr() net.Addr {
	return p.addr
}

func (p *TServerUnixDomain) Close() error {
	defer func() {
		p.listener = nil
	}()
	if p.IsListening() {
		os.Remove(p.addr.String())
		return p.listener.Close()
	}
	return nil
}

func (p *TServerUnixDomain) Interrupt() error {
	p.mu.Lock()
	p.interrupted = true
	p.mu.Unlock()

	return nil
}
