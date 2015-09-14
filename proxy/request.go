// Copyright 2015 Spring Rain Software Compnay LTD. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.
package proxy

import (
	thrift "git.apache.org/thrift.git/lib/go/thrift"
	"github.com/wfxiang08/rpc_proxy/utils/atomic2"
	"strings"
	"sync"
)

type Dispatcher interface {
	Dispatch(r *Request) error
}

const (
	// 不能使用负数
	MESSAGE_TYPE_HEART_BEAT thrift.TMessageType = 20
	MESSAGE_TYPE_STOP       thrift.TMessageType = 21
)

type Request struct {
	Service      string // 服务
	ProxyRequest bool   // Service是否出现在Request.Data中，默认为true, 但是心跳等信号中没有service

	// 原始的数据(虽然拷贝有点点效率低，但是和zeromq相比也差不多)
	Request struct {
		Name     string
		TypeId   thrift.TMessageType
		SeqId    int32
		Data     []byte
		DataOrig []byte
	}

	OpStr string
	Start int64

	// 返回的数据类型
	Response struct {
		Data   []byte
		Err    error
		SeqId  int32 // -1保留，表示没有对应的SeqNum
		TypeId thrift.TMessageType
	}

	Wait *sync.WaitGroup

	Failed *atomic2.Bool
}

//
// 给定一个thrift message，构建一个Request对象
//
func NewRequest(data []byte, serviceInReq bool) *Request {
	request := &Request{
		Wait:         &sync.WaitGroup{},
		ProxyRequest: serviceInReq,
	}
	request.Request.Data = data
	request.DecodeRequest()

	return request

}

//
// 从Request.Data中读取出 Request的Name, TypeId, SeqId
// RequestName可能和thrift package中的name不一致，Service部分从Name中剔除
//
func (r *Request) DecodeRequest() {
	transport := NewTMemoryBufferWithBuf(r.Request.Data)
	protocol := thrift.NewTBinaryProtocolTransport(transport)

	r.Request.Name, r.Request.TypeId, r.Request.SeqId, _ = protocol.ReadMessageBegin()

	// 参考 ： TMultiplexedProtocol
	nameFields := strings.SplitN(r.Request.Name, thrift.MULTIPLEXED_SEPARATOR, 2)
	if len(nameFields) != 2 {
		r.Service = ""
	} else {
		r.Service = nameFields[0]
		r.Request.Name = nameFields[1]
	}
}

//
// 将Request中的SeqNum进行替换（修改Request部分的数据)
//
func (r *Request) ReplaceSeqId(newSeq int32) {
	if r.Request.Data != nil {
		//		log.Printf(Green("Replace SeqNum: %d --> %d"), r.Request.SeqId, newSeq)
		r.Response.SeqId = newSeq

		start := 0

		if r.ProxyRequest {
			start = len(r.Service)
		}
		if start > 0 {
			start += 1 // ":"
			//			log.Printf("Service: %s, Name: %s\n", r.Service, r.Request.Name)
		}
		transport := NewTMemoryBufferWithBuf(r.Request.Data[start:start])
		protocol := thrift.NewTBinaryProtocolTransport(transport)
		protocol.WriteMessageBegin(r.Request.Name, r.Request.TypeId, newSeq)

		if start > 0 {
			r.Request.DataOrig = r.Request.Data
		}
		// 将service从name中剥离出去
		r.Request.Data = r.Request.Data[start:len(r.Request.Data)]

	}
}

func (r *Request) Recycle() {
	var sliceId uintptr = 0

	if r.Request.DataOrig != nil {
		returnSlice(r.Request.DataOrig)
		sliceId = getSliceId(r.Request.DataOrig)

		r.Request.DataOrig = nil
		r.Request.Data = nil
	} else if r.Request.Data != nil {
		sliceId = getSliceId(r.Request.Data)
		returnSlice(r.Request.Data)
		r.Request.Data = nil
	}
	if r.Response.Data != nil {
		if sliceId != getSliceId(r.Response.Data) {
			returnSlice(r.Response.Data)
		}
		r.Response.Data = nil
	}
}

func (r *Request) RestoreSeqId() {
	if r.Response.Data != nil {
		transport := NewTMemoryBufferWithBuf(r.Response.Data[0:0])
		protocol := thrift.NewTBinaryProtocolTransport(transport)

		// 切换回原始的SeqId
		// r.Response.TypeId 和 r.Request.TypeId可能不一样，要以Response为准
		protocol.WriteMessageBegin(r.Request.Name, r.Response.TypeId, r.Request.SeqId)
	}
}

//
// 给定thrift Message, 解码出: typeId, seqId
//
func DecodeThriftTypIdSeqId(data []byte) (typeId thrift.TMessageType, seqId int32, err error) {
	transport := NewTMemoryBufferWithBuf(data)
	protocol := thrift.NewTBinaryProtocolTransport(transport)

	_, typeId, seqId, err = protocol.ReadMessageBegin()
	return
}
