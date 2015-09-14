# Copyright 2015 Spring Rain Software Company LTD. All Rights Reserved.
# Licensed under the MIT (MIT-LICENSE.txt) license.

thrift -r --gen py rpc_thrift.services.thrift
mv gen-py/rpc_thrift/services/* lib/python/rpc_thrift/services/
