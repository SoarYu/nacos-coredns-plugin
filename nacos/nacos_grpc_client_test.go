package nacos

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

var grpcClientTest = NewNacosGrpcClientTest()

func NewNacosGrpcClientTest() *NacosGrpcClient {
	grpcClient, _ := NewNacosGrpcClient("", []string{"console.nacos.io:8848"}, nacosClientTest)
	return grpcClient
}

func TestGetAllServicesInfo(t *testing.T) {
	doms := grpcClientTest.GetAllServicesInfo()
	if assert.NotNil(t, doms) {
		fmt.Println("doms size:", len(doms))
	}
}

func TestGetService(t *testing.T) {
	doms := grpcClientTest.GetAllServicesInfo()
	for _, dom := range doms {
		service := grpcClientTest.GetService(dom)
		assert.NotNil(t, service)
	}
}

func TestSubscribe(t *testing.T) {
	doms := grpcClientTest.GetAllServicesInfo()
	for _, dom := range doms {
		err := grpcClientTest.Subscribe(dom)
		assert.Nil(t, err)
	}
}

func TestCallback(t *testing.T) {
	services := model.Service{
		Name:        "DEFAULT_GROUP@@demo.go",
		CacheMillis: 1000,
		Hosts: []model.Instance{
			{
				InstanceId:  "10.10.10.10-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.10",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEFAULT_GROUP@@demo.go",
				Enable:      true,
				Healthy:     true,
			},
			{
				InstanceId:  "10.10.10.11-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.11",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEFAULT_GROUP@@demo.go",
				Enable:      true,
				Healthy:     true,
			},
			{
				InstanceId:  "10.10.10.12-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.12",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEFAULT_GROUP@@demo.go",
				Enable:      true,
				Healthy:     false,
			},
			{
				InstanceId:  "10.10.10.13-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.13",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEFAULT_GROUP@@demo.go",
				Enable:      false,
				Healthy:     true,
			},
			{
				InstanceId:  "10.10.10.14-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.14",
				Weight:      0,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEFAULT_GROUP@@demo.go",
				Enable:      true,
				Healthy:     true,
			},
		},
		Checksum:    "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
		LastRefTime: 1528787794594, Clusters: "a"}

	grpcClientTest.nacosClient.GetDomainCache().Set("demo.go", services)

	newServices := model.Service{
		Name:        "DEFAULT_GROUP@@demo.go",
		CacheMillis: 1000,
		Hosts: []model.Instance{
			{
				InstanceId:  "10.10.10.10-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.10",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEFAULT_GROUP@@demo.go",
				Enable:      true,
				Healthy:     true,
			},
			{
				InstanceId:  "10.10.10.11-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.11",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEFAULT_GROUP@@demo.go",
				Enable:      true,
				Healthy:     true,
			},
			{
				InstanceId:  "10.10.10.12-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.12",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEFAULT_GROUP@@demo.go",
				Enable:      true,
				Healthy:     false,
			},
			{
				InstanceId:  "10.10.10.13-80-a-DEMO",
				Port:        80,
				Ip:          "10.10.10.13",
				Weight:      1,
				Metadata:    map[string]string{},
				ClusterName: "a",
				ServiceName: "DEFAULT_GROUP@@demo.go",
				Enable:      false,
				Healthy:     true,
			},
		},
		Checksum:    "3bbcf6dd1175203a8afdade0e77a27cd1528787794594",
		LastRefTime: 1528787794594, Clusters: "a"}
	grpcClientTest.Callback(newServices.Hosts, nil)

	s, _ := grpcClientTest.nacosClient.GetDomainCache().Get("demo.go")

	updateServices := s.(model.Service)

	pass := len(newServices.Hosts) == len(updateServices.Hosts)
	assert.True(t, pass)

}
