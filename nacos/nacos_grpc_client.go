package nacos

import (
	"encoding/json"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"strconv"
	"strings"
)

type NacosGrpcClient struct {
	namespaceId   string
	clientConfig  constant.ClientConfig       //nacos-coredns客户端配置
	serverConfigs []constant.ServerConfig     //nacos服务器集群配置
	grpcClient    naming_client.INamingClient //nacos-coredns与nacos服务器的grpc连接
	subsrcibeMap  map[string]bool
	nacosClient   *NacosClient
}

func NewNacosGrpcClient(namespaceId string, serverHosts []string, vc *NacosClient) *NacosGrpcClient {
	var nacosGrpcClient NacosGrpcClient
	if namespaceId == "public" {
		namespaceId = ""
	}
	nacosGrpcClient.namespaceId = namespaceId //When namespace is public, fill in the blank string here.

	var serverConfigs []constant.ServerConfig
	for _, serverHost := range serverHosts {
		fmt.Println("nacos_server_host:", serverHost)
		serverIp := strings.Split(serverHost, ":")[0]
		serverPort, _ := strconv.Atoi(strings.Split(serverHost, ":")[1])
		fmt.Println("serverIP, serverPost:", serverIp, serverPort)
		serverConfigs = append(serverConfigs, *constant.NewServerConfig(
			serverIp,
			uint64(serverPort),
			constant.WithScheme("http"),
			constant.WithContextPath("/nacos"),
		))
	}
	fmt.Println("serverConfigs:", serverConfigs)

	nacosGrpcClient.clientConfig = *constant.NewClientConfig(
		constant.WithNamespaceId(namespaceId),
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("debug"),
	)

	nacosGrpcClient.serverConfigs = serverConfigs

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

	nacosGrpcClient.subsrcibeMap = make(map[string]bool)

	nacosGrpcClient.nacosClient = vc

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
	service, _ := ngc.grpcClient.GetService(vo.GetServiceParam{ //从服务端返回model.Service
		ServiceName: serviceName,
	})
	if service.Hosts == nil {
		NacosClientLogger.Warn("empty result from server, dom:" + serviceName)
		return Domain{}
	}
	var domain Domain
	//s, _ := json.Marshal(service)      //model.Service转换成json
	s := util.ToJsonString(service)
	err1 := json.Unmarshal([]byte(s), &domain) //json转换成Domain
	if err1 != nil {
		NacosClientLogger.Error("failed to unmarshal json string: ", err1)
		return Domain{Name: serviceName}
	}
	return domain
}

func (ngc *NacosGrpcClient) Subscribe(serviceName string) {
	//if(ngc.subsrcibeMap[serviceName]){
	//
	//}
	param := &vo.SubscribeParam{
		ServiceName:       "demo.go",
		GroupName:         "",
		SubscribeCallback: ngc.Callback,
	}
	ngc.grpcClient.Subscribe(param)
	//ngc.grpcClient.Unsubscribe(param)
}

func (ngc *NacosGrpcClient) Callback(services []model.Instance, err error) {
	//fmt.Println("callback return")
	fmt.Printf("callback return services:%s \n\n", util.ToJsonString(services))

	//s := util.ToJsonString(services)
	//err1 := json.Unmarshal([]byte(s), &domain) //json转换成Domain

}