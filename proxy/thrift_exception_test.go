// Copyright 2015 Spring Rain Software Compnay LTD. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.
package proxy

import (
	//	"fmt"
	//	thrift "git.apache.org/thrift.git/lib/go/thrift"
	//	"github.com/wfxiang08/rpc_proxy/utils/assert"
	//	"strings"
	"testing"
)

//
// go test github.com/wfxiang08/rpc_proxy/proxy -v -run "TestGetThriftException"
//
func TestGetThriftException(t *testing.T) {

	//	serviceName := "accounts"
	//	data := GetServiceNotFoundData(serviceName, 0)
	//	fmt.Println("Exception Data: ", string(data))

	//	transport := NewTMemoryBufferWithBuf(data)
	//	exc := thrift.NewTApplicationException(-1, "")
	//	protocol := thrift.NewTBinaryProtocolTransport(transport)

	//	// 注意: Read函数返回的是一个新的对象
	//	protocol.ReadMessageBegin()
	//	exc, _ = exc.Read(protocol)
	//	protocol.ReadMessageEnd()

	//	fmt.Println("Exc: ", exc.TypeId(), "Error: ", exc.Error())

	//	var errMsg string = exc.Error()
	//	assert.Must(strings.Contains(errMsg, serviceName))
}
