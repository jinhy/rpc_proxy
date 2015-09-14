// Copyright 2015 Spring Rain Software Compnay LTD. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.
package proxy

import (
	"encoding/json"
	thrift "git.apache.org/thrift.git/lib/go/thrift"
	"github.com/wfxiang08/rpc_proxy/utils/atomic2"
	"github.com/wfxiang08/rpc_proxy/utils/errors"
	"github.com/wfxiang08/rpc_proxy/utils/log"
	"time"
)

//
// 用于rpc proxy或者load balance用来管理Client的
//
type Session struct {
	*TBufferedFramedTransport

	RemoteAddress string
	Ops           int64
	LastOpUnix    int64
	CreateUnix    int64

	quit    bool
	closed  atomic2.Bool
	verbose bool
}

// 返回当前Session的状态
func (s *Session) String() string {
	o := &struct {
		Ops        int64  `json:"ops"`
		LastOpUnix int64  `json:"lastop"`
		CreateUnix int64  `json:"create"`
		RemoteAddr string `json:"remote"`
	}{
		s.Ops, s.LastOpUnix, s.CreateUnix,
		s.RemoteAddress,
	}
	b, _ := json.Marshal(o)
	return string(b)
}

// c： client <---> proxy之间的连接
func NewSession(c thrift.TTransport, address string, verbose bool) *Session {
	return NewSessionSize(c, address, verbose, 1024*32, 5000)
}

func NewSessionSize(c thrift.TTransport, address string, verbose bool,
	bufsize int, timeout int) *Session {

	s := &Session{
		CreateUnix:               time.Now().Unix(),
		RemoteAddress:            address,
		verbose:                  verbose,
		TBufferedFramedTransport: NewTBufferedFramedTransport(c, time.Microsecond*100, 20),
	}

	// Reader 处理Client发送过来的消息
	// Writer 将后端服务的数据返回给Client
	log.Infof(Green("NewSession To: %s"), s.RemoteAddress)
	return s
}

func (s *Session) Close() error {
	s.closed.Set(true)
	log.Printf(Red("Close Proxy Session"))
	return s.TBufferedFramedTransport.Close()
}

func (s *Session) IsClosed() bool {
	return s.closed.Get()
}

func (s *Session) Serve(d Dispatcher, maxPipeline int) {

	var errlist errors.ErrorList
	defer func() {
		log.Infof(Red("==> Session Over: %s, Print Error List: %d Errors"),
			s.RemoteAddress, errlist.Len())

		// 只打印第一个Error
		if err := errlist.First(); err != nil {
			log.Infof("==> Session [%p] closed, Error = %v", s, err)
		} else {
			log.Infof("==> Session [%p] closed, Quit", s)
		}
	}()

	// 来自connection的各种请求
	tasks := make(chan *Request, maxPipeline)
	go func() {
		defer func() {
			// 出现错误了，直接关闭Session
			s.Close()

			// 扔掉所有的Tasks
			log.Warnf(Red("Session Closed, Abandon %d Tasks"), len(tasks))
			for task := range tasks {
				task.Recycle()
			}
		}()
		if err := s.loopWriter(tasks); err != nil {
			errlist.PushBack(err)
		}
	}()

	defer close(tasks)

	// 从Client读取用户的请求，然后再交给Dispatcher来处理
	if err := s.loopReader(tasks, d); err != nil {
		errlist.PushBack(err)
	}
	log.Info(Cyan("LoopReader Over, Session#Serve Over"))
}

// 从Client读取数据
func (s *Session) loopReader(tasks chan<- *Request, d Dispatcher) error {
	if d == nil {
		return errors.New("nil dispatcher")
	}

	for !s.quit {
		// client <--> rpc
		// 从client读取frames
		request, err := s.ReadFrame()
		if err != nil {
			err1, ok := err.(thrift.TTransportException)
			if !ok || err1.TypeId() != thrift.END_OF_FILE {
				// 遇到EOF等错误，就直接结束loopReader
				// 结束之前需要和后端的back_conn之间处理好关系?
				log.ErrorErrorf(err, Red("ReadFrame Error: %v"), err)
			}
			return err
		}

		r, err := s.handleRequest(request, d)
		if err != nil {
			return err
		} else {
			if s.verbose {
				log.Info("Succeed Get Result")
			}

			// 将请求交给: tasks, 同一个Session中的请求是
			tasks <- r
		}
	}
	return nil
}

func (s *Session) loopWriter(tasks <-chan *Request) error {
	// Proxy: Session ---> Client
	for r := range tasks {
		// 1. 等待Request对应的Response
		//    出错了如何处理呢?
		s.handleResponse(r)

		// 2. 将结果写回给Client
		if s.verbose {
			log.Printf("[%s]Session#loopWriter --> client FrameSize: %d",
				r.Service, len(r.Response.Data))
		}

		// r.Response.Data ---> Client
		_, err := s.TBufferedFramedTransport.Write(r.Response.Data)
		if err != nil {
			log.ErrorErrorf(err, "Write back Data Error: %v", err)
			return err
		}

		// 3. Flush
		err = s.TBufferedFramedTransport.FlushBuffer(true) // len(tasks) == 0
		if err != nil {
			log.ErrorErrorf(err, "Write back Data Error: %v", err)
			return err
		}
		r.Recycle()
	}
	return nil
}

//
//
// 等待Request请求的返回: Session最终被Block住
//
func (s *Session) handleResponse(r *Request) {
	// 等待结果的出现
	r.Wait.Wait()

	// 将Err转换成为Exception
	if r.Response.Err != nil {

		r.Response.Data = GetThriftException(r, "proxy_session")
		log.Printf(Magenta("---->Convert Error Back to Exception"))
	}

	// 如何处理Data和Err呢?
	incrOpStats(r.OpStr, microseconds()-r.Start)
}

// 处理来自Client的请求
func (s *Session) handleRequest(request []byte, d Dispatcher) (*Request, error) {
	// 构建Request
	if s.verbose {
		log.Printf("HandleRequest: %s", string(request))
	}
	r := NewRequest(request, true)

	// 增加统计
	s.LastOpUnix = time.Now().Unix()
	s.Ops++

	// 交给Dispatch
	// Router
	return r, d.Dispatch(r)
}

func microseconds() int64 {
	return time.Now().UnixNano() / int64(time.Microsecond)
}
