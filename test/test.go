package main

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
	subscribeMap  map[string]bool
}

func NewNacosGrpcClient() *NacosGrpcClient {
	var nacosGrpcClient NacosGrpcClient
	serverHosts := []string{"106.52.77.111:8848", "106.52.77.111:8849"}
	var serverConfigs = []constant.ServerConfig{}
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
	fmt.Println(serverConfigs)
	//
	nacosGrpcClient.clientConfig = *constant.NewClientConfig(
		constant.WithNamespaceId(""),
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

	nacosGrpcClient.subscribeMap = make(map[string]bool)

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

	for _, name := range doms {
		ngc.Subscribe(name)
		ngc.subscribeMap[name] = true
	}

	return doms
}

func (ngc *NacosGrpcClient) GetService(serviceName string) model.Service {
	service, _ := ngc.grpcClient.GetService(vo.GetServiceParam{ //从服务端返回model.Service
		ServiceName: serviceName,
	})
	if service.Hosts == nil {
		//NacosClientLogger.Warn("empty result from server, dom:" + serviceName)
		return model.Service{}
	}
	var domain model.Service
	s, _ := json.Marshal(service)      //model.Service转换成json
	err1 := json.Unmarshal(s, &domain) //json转换成Domain
	if err1 != nil {
		//NacosClientLogger.Error("failed to unmarshal json string: ", err1)
		return model.Service{Name: serviceName}
	}
	return domain
}

func (ngc *NacosGrpcClient) Subscribe(serviceName string) {
	if ngc.subscribeMap[serviceName] {
		fmt.Println(serviceName, " isSubscribe")
		return
	}
	param := &vo.SubscribeParam{
		ServiceName:       "demo.go",
		GroupName:         "",
		SubscribeCallback: Callback,
	}
	ngc.grpcClient.Subscribe(param)
	//ngc.grpcClient.Unsubscribe(param)
}

var GrpcClient *NacosGrpcClient

func main() {
	GrpcClient = NewNacosGrpcClient()
	serviceNames := GrpcClient.GetAllServicesInfo()
	fmt.Println("serviceNames:", serviceNames)
}

func Callback(services []model.Instance, err error) {
	//fmt.Println("callback return")
	fmt.Printf("callback return services:%s \n\n", util.ToJsonString(services))

	//s := util.ToJsonString(services)
	//err1 := json.Unmarshal([]byte(s), &domain) //json转换成Domain

}
