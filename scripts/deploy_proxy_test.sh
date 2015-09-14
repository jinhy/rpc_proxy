# Copyright 2015 Spring Rain Software Company LTD. All Rights Reserved.
# Licensed under the MIT (MIT-LICENSE.txt) license.

if [ "$#" -ne 1 ]; then
    echo "Please input hostname"
    exit -1
fi


host_name=$1

# 创建目录，拷贝rpc_proxy/rpc_lb
echo "ssh root@${host_name} mkdir -p /usr/local/rpc_proxy/bin/"
ssh root@${host_name} "mkdir -p /usr/local/rpc_proxy/bin/"
ssh root@${host_name} "mkdir -p /usr/local/rpc_proxy/log/"

# 拷贝: rpc_lb
echo "ssh root@${host_name} rm -f /usr/local/rpc_proxy/bin/rpc_lb"
ssh root@${host_name} "rm -f /usr/local/rpc_proxy/bin/rpc_lb"

echo "scp rpc_lb root@${host_name}:/usr/local/rpc_proxy/bin/rpc_lb"
scp rpc_lb root@${host_name}:/usr/local/rpc_proxy/bin/rpc_lb

# 拷贝: rpc_proxy
echo "ssh root@${host_name} rm -f /usr/local/rpc_proxy/bin/rpc_proxy"
ssh root@${host_name} "rm -f /usr/local/rpc_proxy/bin/rpc_proxy"

echo "scp rpc_proxy root@${host_name}:/usr/local/rpc_proxy/bin/rpc_proxy"
scp rpc_proxy root@${host_name}:/usr/local/rpc_proxy/bin/rpc_proxy

# 拷贝脚本
scp github.com/wfxiang08/rpc_proxy/scripts/control_lb.sh    root@${host_name}:/usr/local/rpc_proxy/
scp github.com/wfxiang08/rpc_proxy/scripts/control_proxy.sh root@${host_name}:/usr/local/rpc_proxy/
scp github.com/wfxiang08/rpc_proxy/scripts/config.test.ini  root@${host_name}:/usr/local/rpc_proxy/config.ini
scp github.com/wfxiang08/rpc_proxy/scripts/rpc_proxy.conf.upstart  root@${host_name}:/etc/init/rpc_proxy.conf
