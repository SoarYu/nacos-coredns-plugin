package nacos

import (
	"encoding/json"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type NacosGrpcClient struct {
	namespaceId   string
	clientConfig  constant.ClientConfig       //nacos-coredns客户端配置
	serverConfigs []constant.ServerConfig     //nacos服务器集群配置
	grpcClient    naming_client.INamingClient //nacos-coredns与nacos服务器的grpc连接
}

func NewNacosGrpcClient(namespaceId string, servers []string) *NacosGrpcClient {
	var nacosGrpcClient NacosGrpcClient
	if namespaceId == "public" {
		namespaceId = ""
	}
	nacosGrpcClient.namespaceId = namespaceId
	nacosGrpcClient.clientConfig = *constant.NewClientConfig(
		constant.WithNamespaceId(namespaceId), //When namespace is public, fill in the blank string here.
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("debug"),
	)
	//Another way of create serverConfigs
	nacosGrpcClient.serverConfigs = []constant.ServerConfig{
		*constant.NewServerConfig(
			servers[0],
			8848,
			constant.WithScheme("http"),
			constant.WithContextPath("/nacos"),
		),
		*constant.NewServerConfig(
			servers[0],
			8849,
			constant.WithScheme("http"),
			constant.WithContextPath("/nacos"),
		),
	}
	// Another way of create naming client for service discovery (recommend)
	var err error
	nacosGrpcClient.grpcClient, err = clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &nacosGrpcClient.clientConfig,
			ServerConfigs: nacosGrpcClient.serverConfigs,
		},
	)
	if err != nil {
		fmt.Println("init nacos-client error")
	}

	return &nacosGrpcClient
}

//Doms        []string `json:"doms"`
func (ngc *NacosGrpcClient) GetAllServicesInfo() []string {
	var pageNo = uint32(1)
	var pageSize = uint32(10)
	var doms []string
	serviceList, _ := ngc.grpcClient.GetAllServicesInfo(vo.GetAllServiceInfoParam{
		NameSpace: ngc.namespaceId,
		PageNo:    pageNo,
		PageSize:  pageSize,
	})

	if serviceList.Count == 0 {
		return doms
	}

	doms = append(doms, serviceList.Doms...)

	for pageNo = 2; serviceList.Count >= int64(pageSize); pageNo++ {
		serviceList, _ = ngc.grpcClient.GetAllServicesInfo(vo.GetAllServiceInfoParam{
			NameSpace: ngc.namespaceId,
			PageNo:    pageNo,
			PageSize:  pageSize,
		})
		if serviceList.Count != 0 {
			doms = append(doms, serviceList.Doms...)
		}
	}

	return doms
}

func (ngc *NacosGrpcClient) GetService(serviceName string) Domain {
	service, _ := ngc.grpcClient.GetService(vo.GetServiceParam{
		ServiceName: serviceName,
	})
	if service.Hosts == nil {
		NacosClientLogger.Warn("empty result from server, dom:" + serviceName)
		return Domain{}
	}
	s, _ := json.Marshal(service) //model.Service转换成[]byte

	var domain Domain
	err1 := json.Unmarshal(s, &domain) //[]byte转换成Domain
	if err1 != nil {
		NacosClientLogger.Error("failed to unmarshal json string: ", err1)
		return Domain{Name: serviceName}
	}
	return domain
}
