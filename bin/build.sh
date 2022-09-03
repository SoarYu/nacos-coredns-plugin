#!/usr/bin/env bash
# cd GOPATH
cd $GOPATH/src/

# remove codes
rm -rf coredns
rm -rf nacos-coredns-plugin

# clone current codes
git clone https://github.com/nacos-group/nacos-coredns-plugin.git
git clone https://github.com/coredns/coredns.git


# cd coredns directory
cd $GOPATH/src/coredns
git checkout -b v1.9.3 v1.9.3

# copy nacos plugin to coredns
cp -r ../nacos-coredns-plugin/nacos plugin/
cp -r ../nacos-coredns-plugin/forward/setup.go plugin/forward
cp -r ../nacos-coredns-plugin/conf conf

# insert nacos into plugin
sed -i '/hosts/a\\t"nacos",' core/dnsserver/zdirectives.go
sed -i '/coredns\/plugin\/hosts/a\\t_ "github.com/coredns/coredns/plugin/nacos"' core/plugin/zplugin.go
sed -i '/hosts:hosts/a\nacos:nacos' plugin.cfg

#go get github.com/cihub/seelog
#go get github.com/nacos-group/nacos-sdk-go/v2
#go get github.com/nacos-group/nacos-coredns-plugin/nacos
go mod tidy

# build
make