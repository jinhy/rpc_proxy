// Copyright 2015 Spring Rain Software Compnay LTD. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.
package proxy

import (
	"fmt"
	thrift "git.apache.org/thrift.git/lib/go/thrift"
	"github.com/stretchr/testify/assert"
	"github.com/wfxiang08/rpc_proxy/utils/log"
	"io"
	"testing"
	"time"
)

func init() {
	log.SetLevel(log.LEVEL_INFO)
	log.SetFlags(log.Flags() | log.Lshortfile)
}

type fakeServer struct {
}

func (p *fakeServer) Dispatch(r *Request) error {
	log.Printf("Request SeqId: %d, MethodName: %s\n", r.Request.SeqId, r.Request.Name)
	r.Wait.Add(1)
	go func() {
		time.Sleep(time.Millisecond)
		r.Response.Data = []byte(string(r.Request.Data))

		typeId, seqId, _ := DecodeThriftTypIdSeqId(r.Response.Data)
		log.Printf(Green("TypeId: %d, SeqId: %d\n"), typeId, seqId)
		r.Wait.Done()
	}()
	//	r.RestoreSeqId()
	//	r.Wait.Done()
	return nil
}

//
// go test github.com/wfxiang08/rpc_proxy/proxy -v -run "TestError"
//
func TestError(t *testing.T) {
	err1 := io.EOF
	err2 := io.EOF
	fmt.Printf("Err1: %p, Err2: %p, Equal: %t\n", err1, err2, err1 == err2)
}

//
// go test github.com/wfxiang08/rpc_proxy/proxy -v -run "TestSession"
//
func TestSession(t *testing.T) {
	// 作为一个Server
	transport, err := thrift.NewTServerSocket("127.0.0.1:0")
	assert.NoError(t, err)
	err = transport.Open() // 打开Transport
	assert.NoError(t, err)

	defer transport.Close()

	err = transport.Listen() // 开始监听
	assert.NoError(t, err)

	addr := transport.Addr().String()

	fmt.Println("Addr: ", addr)

	var requestNum int32 = 10
	requests := make([]*Request, 0, requestNum)

	var i int32
	for i = 0; i < requestNum; i++ {
		buf := make([]byte, 100, 100)
		l := fakeData("Hello", thrift.CALL, i+1, buf[0:0])
		buf = buf[0:l]

		req := NewRequest(buf, true)

		req.Wait.Add(1) // 因为go routine可能还没有执行，代码就跑到最后面进行校验了

		assert.Equal(t, i+1, req.Request.SeqId, "Request SeqId是否靠谱")

		requests = append(requests, req)
	}
	go func() {
		// 模拟请求:
		// 客户端代码
		bc := NewBackendConn(addr, nil, "test", true)
		bc.currentSeqId = 10

		// 准备发送数据
		var i int32
		for i = 0; i < requestNum; i++ {
			fmt.Println("Sending Request to Backend Conn", i)
			bc.PushBack(requests[i])

			requests[i].Wait.Done()
		}

		// 需要等待数据返回?
		time.Sleep(time.Second * 2)
	}()

	server := &fakeServer{}
	go func() {
		// 服务器端代码
		tran, err := transport.Accept()
		defer tran.Close()
		if err != nil {
			log.ErrorErrorf(err, "Error: %v\n", err)
		}
		assert.NoError(t, err)

		session := NewSession(tran, "", true)

		session.Serve(server, 6)

		time.Sleep(time.Second * 2)
	}()

	for i = 0; i < requestNum; i++ {
		fmt.Println("===== Before Wait")
		requests[i].Wait.Wait()
		fmt.Println("===== Before After Wait")

		log.Printf("Request: %d, .....", i)
		assert.Equal(t, len(requests[i].Response.Data), len(requests[i].Request.Data))
	}
}
