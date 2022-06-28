#!/usr/bin/env bash
# cd GOPATH
cd $GOPATH/src/

# remove codes
rm -rf coredns
rm -rf nacos-coredns-plugin

# clone current codes
git clone https://github.com/coredns/coredns.git
git clone https://github.com/nacos-group/nacos-coredns-plugin.git

# cd nacos-coredns-plugin directory
cd $GOPATHgit /src/nacos-coredns-plugin
git checkout -b v1.6.7 origin/v1.6.7
# cd coredns directory
cd $GOPATH/src/coredns
git checkout -b v1.6.7 v1.6.7
go get github.com/cihub/seelog

# copy nacos plugin to coredns
cp -r ../nacos-coredns-plugin/nacos plugin/
cp -r ../nacos-coredns-plugin/forward/setup.go plugin/forward

# insert nacos into plugin
sed -i '/hosts/a\\t"nacos",' core/dnsserver/zdirectives.go
sed -i '/coredns\/plugin\/hosts/a\"coredns/plugin/nacos"' core/plugin/zplugin.go
sed -i '/hosts:hosts/a\nacos:nacos' plugin.cfg

# build
make
