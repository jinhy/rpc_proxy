# Thrift的安装
参考: https://thrift.apache.org/docs/install/os_x

* git clone https://github.com/wfxiang08/thrift
* 安装Boost
	* http://sourceforge.net/projects/boost/files/boost/1.59.0/
	* ./bootstrap.sh sudo ./b2 threading=multi address-model=64 variant=release stage install
* 安装libevent
	* ./configure --prefix=/usr/local make sudo make install
* 安装thrift
    * export CXXFLAGS="-std=c++11"
	* ./configure --prefix=/usr/local/ --with-boost=/usr/local --with-libevent=/usr/local --without-ruby --without-haskell --without-erlang --without-perl --without-php --without-nodejs --without-cpp --without-json --without-as3 --without-csharp --without-erl --without-cocoa --without-ocaml --without-hs --without-xsd --without-html --without-delphi --without-gv --without-lua
	* make CXXFLAGS=-stdlib=libstdc++
	* sudo make install