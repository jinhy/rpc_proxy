# Copyright 2015 Spring Rain Software Company LTD. All Rights Reserved.
# Licensed under the MIT (MIT-LICENSE.txt) license.

go build -ldflags "-X main.buildDate=`date +%Y%m%d%H%M%S` -X main.gitVersion=`git -C github.com/wfxiang08/rpc_proxy rev-parse HEAD`" github.com/wfxiang08/rpc_proxy/cmds/rpc_lb.go 
#&& cp rpc_lb /usr/local/rpc_proxy/bin/
