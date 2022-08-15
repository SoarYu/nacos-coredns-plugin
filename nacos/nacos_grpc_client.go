package nacos

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"strconv"
	"strings"
)

type NacosGrpcClient struct {
	namespaceId   string
	clientConfig  constant.ClientConfig       //nacos-coredns客户端配置
	serverConfigs []constant.ServerConfig     //nacos服务器集群配置
	grpcClient    naming_client.INamingClient //nacos-coredns与nacos服务器的grpc连接
	nacosClient   *NacosClient
}

func NewNacosGrpcClient(namespaceId string, serverHosts []string, vc *NacosClient) (*NacosGrpcClient, error) {
	var nacosGrpcClient NacosGrpcClient
	nacosGrpcClient.nacosClient = vc
	if namespaceId == "public" {
		namespaceId = ""
	}
	nacosGrpcClient.namespaceId = namespaceId //When namespace is public, fill in the blank string here.

	var serverConfigs []constant.ServerConfig
	for _, serverHost := range serverHosts {
		serverIp := strings.Split(serverHost, ":")[0]
		serverPort, err := strconv.Atoi(strings.Split(serverHost, ":")[1])
		if err != nil {
			NacosClientLogger.Error("nacos server host config error!", err)
		}
		serverConfigs = append(serverConfigs, *constant.NewServerConfig(
			serverIp,
			uint64(serverPort),
			constant.WithScheme("http"),
			constant.WithContextPath("/nacos"),
		))
	}
	nacosGrpcClient.serverConfigs = serverConfigs

	nacosGrpcClient.clientConfig = *constant.NewClientConfig(
		constant.WithNamespaceId(namespaceId),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithUpdateCacheWhenEmpty(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir(CachePath),
		constant.WithLogLevel("info"),
	)

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

	return &nacosGrpcClient, err
}

func (ngc *NacosGrpcClient) GetAllServicesInfo() []string {
	var pageNo = uint32(1)
	var pageSize = uint32(100)
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

func (ngc *NacosGrpcClient) GetService(serviceName string) model.Service {
	service, _ := ngc.grpcClient.GetService(vo.GetServiceParam{
		ServiceName: serviceName,
	})
	if service.Hosts == nil {
		NacosClientLogger.Warn("empty result from server, dom:" + serviceName)
	}

	return service
}

func (ngc *NacosGrpcClient) Subscribe(serviceName string) error {
	if AllDoms.Data[serviceName] {
		NacosClientLogger.Info("service " + serviceName + " already subsrcibed.")
		return nil
	}
	param := &vo.SubscribeParam{
		ServiceName:       serviceName,
		GroupName:         "",
		SubscribeCallback: ngc.Callback,
	}
	if err := ngc.grpcClient.Subscribe(param); err != nil {
		NacosClientLogger.Error("service subscribe error " + serviceName)
		return err
	}
	AllDoms.Data[serviceName] = true
	return nil
}

func (ngc *NacosGrpcClient) Callback(instances []model.Instance, err error) {
	//更新实例数量为0
	if len(instances) == 0 {
		for dom, _ := range AllDoms.Data {
			if service := ngc.GetService(dom); len(service.Hosts) == 0 {
				ngc.nacosClient.GetDomainCache().Set(dom, service)
				AllDoms.Data[dom] = false
			}
		}
		return
	}

	serviceName := strings.Split(instances[0].ServiceName, SEPERATOR)[1]
	oldService, ok := ngc.nacosClient.GetDomainCache().Get(serviceName)
	if !ok {
		NacosClientLogger.Info("service not found in cache " + serviceName)
		service := ngc.GetService(serviceName)
		ngc.nacosClient.GetDomainCache().Set(serviceName, service)
	} else {
		service := oldService.(model.Service)
		service.Hosts = instances
		service.LastRefTime = uint64(CurrentMillis())
		ngc.nacosClient.GetDomainCache().Set(serviceName, service)
	}
	NacosClientLogger.Info("serviceName: "+serviceName+" was updated to: ", instances)

}
