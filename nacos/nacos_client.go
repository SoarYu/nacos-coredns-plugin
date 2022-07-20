/*
 * Copyright 1999-2018 Alibaba Group Holding Ltd.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *      http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nacos

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cihub/seelog"
)

var (
	NacosClientLogger seelog.LoggerInterface
	LogConfig         string
	GrpcClient        *NacosGrpcClient
)

func init() {
	initLog()
}

type NacosClient struct {
	domainMap ConcurrentMap
	udpServer UDPServer
	//serverManager ServerManager
	//serverPort    int
}

type NacosClientError struct {
	Msg string
}

func (err NacosClientError) Error() string {
	return err.Msg
}

var Inited = false

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func initDir() {
	dir, err := filepath.Abs(Home())
	if err != nil {
		os.Exit(1)
	}

	if LogPath == "" {
		LogPath = dir + string(os.PathSeparator) + "logs"
	}

	if CachePath == "" {
		CachePath = dir + string(os.PathSeparator) + "nacos-go-client-cache"
	}

	mkdirIfNecessary(CachePath)
}

func mkdirIfNecessary(path string) {
	if ok, _ := exists(path); !ok {
		err := os.Mkdir(path, 0755)
		if err != nil {
			NacosClientLogger.Warn("can not cread dir: "+path, err)
		}
	}
}

func initLog() {
	if NacosClientLogger != nil {
		return
	}

	initDir()
	var err error
	var nacosLogger seelog.LoggerInterface
	if LogConfig == "" || !Exist(LogConfig) {
		home := Home()
		LogConfig = home
		nacosLogger, err = seelog.LoggerFromConfigAsString(`
    		<seelog minlevel="info" maxlevel="error">
        		<outputs formatid="main">
            		<buffered size="1000" flushperiod="1000">
                		<rollingfile type="size" filename="` + home + `/logs/nacos-go-client/nacos-go-client.log" maxsize="100000000" maxrolls="10"/>
        			</buffered>
    			</outputs>
    			<formats>
        			<format id="main" format="%Date(2006-01-02 15:04:05.000) %LEVEL %Msg %n"/>
    			</formats>
			</seelog>
			`)
	} else {
		nacosLogger, err = seelog.LoggerFromConfigAsFile(LogConfig)
	}

	fmt.Println("log directory: " + LogConfig + "/logs/nacos-go-client/")

	if err != nil {
		fmt.Println("Failed to init log, ", err)
	} else {
		NacosClientLogger = nacosLogger
	}
}

func (nacosClient *NacosClient) asyncGetAllDomNAmes() {
	for {
		time.Sleep(time.Duration(AllDoms.CacheSeconds) * time.Second) //30秒更新一次 AllDomNAmes
		nacosClient.getAllDomNames()
	}
}

//func (nacosClient *NacosClient) GetServerManager() (serverManager *ServerManager) {
//	return &nacosClient.serverManager
//}

//func (nacosClient *NacosClient) GetUdpServer() (us UDPServer) {
//	return nacosClient.udpServer
//}

func (nacosClient *NacosClient) getAllDomNames() {
	fmt.Println("getAllDomNames()")
	//ip := nacosClient.serverManager.NextServer()
	//s := Get("http://"+ip+":"+strconv.Itoa(nacosClient.serverPort)+"/nacos/v1/ns/api/allDomNames", nil)
	//s := Get("http://106.52.77.111:8000/allDomNames", nil)
	//

	doms := GrpcClient.GetAllServicesInfo()
	if doms == nil {
		return
	}

	var allName = AllDomNames{Doms: doms, CacheMillis: 0}

	//err := json.Unmarshal([]byte(s), &allName)

	//if err != nil {
	//	//compatible nacos over v1.2.x
	//	var newAllName NewAllDomNames
	//	err = json.Unmarshal([]byte(s), &newAllName)
	//	if err != nil {
	//		NacosClientLogger.Error("failed to unmarshal json: "+s, err)
	//		return
	//	}
	//	var doms []string
	//	for _, domMap := range newAllName.Doms {
	//		for _, dom := range domMap {
	//			doms = append(doms, dom)
	//			//subcribe
	//		}
	//	}
	//	allName.Doms = doms
	//	allName.CacheMillis = newAllName.CacheMillis
	//}
	tmpMap := make(map[string]bool)
	//
	for _, dom := range allName.Doms {
		tmpMap[dom] = true
		////subcribe
	}

	AllDoms.DLock.Lock()
	AllDoms.Data = tmpMap

	if allName.CacheMillis < 30*1000 {
		AllDoms.CacheSeconds = 30
	} else {
		AllDoms.CacheSeconds = allName.CacheMillis / 1000
	}
	AllDoms.DLock.Unlock()
}

//func (nacosClient *NacosClient) SetServers(servers []string) {
//	nacosClient.serverManager.SetServers(servers)
//}

func (vc *NacosClient) Registered(dom string) bool {
	defer AllDoms.DLock.RUnlock()
	AllDoms.DLock.RLock()
	_, ok1 := AllDoms.Data[dom]

	return ok1
}

func (vc *NacosClient) loadCache() {
	files, err := ioutil.ReadDir(CachePath)
	if err != nil {
		NacosClientLogger.Critical(err)
	}

	for _, f := range files {
		fileName := CachePath + string(os.PathSeparator) + f.Name()
		b, err := ioutil.ReadFile(fileName)
		if err != nil {
			NacosClientLogger.Error("failed to read cache file: "+fileName, err)
		}

		s := string(b)
		domain, err1 := ProcessDomainString(s)

		if err1 != nil {
			continue
		}

		vc.domainMap.Set(f.Name(), domain)
	}

	NacosClientLogger.Info("finish loading cache, total: " + strconv.Itoa(len(files)))
}

func ProcessDomainString(s string) (Domain, error) {
	var domain Domain
	err1 := json.Unmarshal([]byte(s), &domain)

	if err1 != nil {
		NacosClientLogger.Error("failed to unmarshal json string: "+s, err1)
		return Domain{}, err1
	}

	if len(domain.Hosts) == 0 {
		NacosClientLogger.Warn("get empty ip list, ignore it, dom: " + domain.Name)
		return domain, NacosClientError{"empty ip list"}
	}

	NacosClientLogger.Info("domain "+domain.Name+" is updated, current ips: ", domain.Hosts)
	return domain, nil
}

func NewNacosClient(namespaceId string, serverHosts []string, serverPort int) *NacosClient {
	fmt.Println("init nacos client.")
	initLog()
	vc := NacosClient{NewConcurrentMap(), UDPServer{}}
	vc.loadCache()
	vc.udpServer.vipClient = &vc
	//vc.SetServers(servers)
	//init grpcClient
	GrpcClient = NewNacosGrpcClient(namespaceId, serverHosts, &vc)

	if EnableReceivePush {
		go vc.udpServer.StartServer()
	}

	AllDoms = AllDomsMap{}
	AllDoms.Data = make(map[string]bool)
	AllDoms.DLock = sync.RWMutex{}

	vc.getAllDomNames()

	go vc.asyncGetAllDomNAmes() //定时更新所有域名

	go vc.asyncUpdateDomain() //定时更新服务名对应ip信息

	NacosClientLogger.Info("cache-path: " + CachePath)
	return &vc
}

func (vc *NacosClient) GetDomainCache() ConcurrentMap {
	return vc.domainMap
}

//func (vc *NacosClient) GetDomain(name string) (*Domain, error) {
//	item, _ := vc.domainMap.Get(name)
//
//	if item == nil {
//		domain := Domain{}
//		ss := strings.Split(name, SEPERATOR)
//
//		domain.Name = ss[0]
//		domain.CacheMillis = DefaultCacheMillis
//		domain.LastRefMillis = CurrentMillis()
//		vc.domainMap.Set(name, domain)
//		item = domain
//		return nil, NacosClientError{"domain not found: " + name}
//	}
//
//	domain := item.(Domain)
//
//	return &domain, nil
//}

func (vc *NacosClient) asyncUpdateDomain() {
	for {
		for k, v := range vc.domainMap.Items() {
			dom := v.(Domain)
			ss := strings.Split(k, SEPERATOR)

			domName := ss[0]
			var clientIP string
			if len(ss) > 1 && ss[1] != "" {
				clientIP = ss[1]
			}
			//如果超过缓存时间并且map里包含了该服务
			if CurrentMillis()-dom.LastRefMillis > dom.CacheMillis && vc.Registered(domName) {
				vc.getDomNow(domName, &vc.domainMap, clientIP)
			}
		}
		time.Sleep(1 * time.Second)
	}

}

func GetCacheKey(dom, clientIP string) string {
	return dom + SEPERATOR + clientIP
}

func (vc *NacosClient) getDomNow(serviceName string, cache *ConcurrentMap, clientIP string) Domain {
	fmt.Println("getDomNow:", serviceName)
	params := make(map[string]string)
	params["serviceName"] = serviceName

	domain := GrpcClient.GetService(serviceName)
	//if clientIP != "" {
	//	params["clientIP"] = clientIP
	//}

	//ip := vc.serverManager.NextServer()
	//s := Get("http://"+ip+":"+strconv.Itoa(vc.serverPort)+"/nacos/v1/ns/api/srvIPXT?", params)
	//s := Get("http://106.52.77.111:8000/srvIPXT?", params)

	//if s == "" {
	//	NacosClientLogger.Warn("empty result from server, dom:" + serviceName)
	//	return Domain{}
	//}
	//
	//domain, err1 := ProcessDomainString(s)
	//if err1 != nil {
	//	domain.Name = serviceName
	//	return domain
	//}

	cacheKey := GetCacheKey(serviceName, clientIP) //GetCacheKey return dom + SEPERATOR + clientIP

	oldDomain, ok := cache.Get(cacheKey)

	if !ok || ok && !reflect.DeepEqual(domain.Hosts, oldDomain.(Domain).Hosts) {
		if !ok {
			NacosClientLogger.Info("dom not found in cache " + cacheKey)
			oldDomain = Domain{}
		}

		NacosClientLogger.Info("dom "+cacheKey+" updated: ", domain)
	}

	domFileName := CachePath + string(os.PathSeparator) + cacheKey

	domainByte, _ := json.Marshal(domain)
	err := ioutil.WriteFile(domFileName, domainByte, 0666)
	if err != nil {
		NacosClientLogger.Error("faild to write cache "+cacheKey+", value: "+string(domainByte), err)
	}

	cache.Set(cacheKey, domain)
	return domain
}

func (vc *NacosClient) SrvInstance(domainName, clientIP string) *Instance {
	cacheKey := GetCacheKey(domainName, clientIP)
	item, hasDom := vc.domainMap.Get(cacheKey)
	var dom Domain
	if !hasDom {
		dom = Domain{}
		dom.LastRefMillis = CurrentMillis()
		dom.CacheMillis = DefaultCacheMillis
		vc.domainMap.Set(GetCacheKey(domainName, clientIP), dom)
		dom = vc.getDomNow(domainName, &vc.domainMap, clientIP)
	} else {
		dom = item.(Domain)
	}

	hosts := dom.SrvInstances()

	if len(hosts) == 0 {
		NacosClientLogger.Warn("no hosts for " + domainName)
		return nil
	}

	i, indexOk := indexMap.Get(domainName)
	var index int

	if !indexOk {
		index = rand.Intn(len(hosts))
	} else {
		index = i.(int)
		index += 1
		if index >= len(hosts) {
			index = index % len(hosts)
		}
	}

	indexMap.Set(domainName, index)

	return &hosts[index]
}

func (vc *NacosClient) SrvInstances(domainName, clientIP string) []Instance {
	cacheKey := GetCacheKey(domainName, clientIP)
	item, hasDom := vc.domainMap.Get(cacheKey)
	var dom Domain

	if !hasDom {
		dom = Domain{}
		dom.Name = domainName
		vc.domainMap.Set(cacheKey, dom)
		dom = vc.getDomNow(domainName, &vc.domainMap, clientIP)
	} else {
		dom = item.(Domain)
	}

	hosts := dom.SrvInstances()

	return hosts
}

func (vc *NacosClient) Contains(dom, clientIP string, host Instance) bool {
	hosts := vc.SrvInstances(dom, clientIP)

	for _, host1 := range hosts {
		if host1 == host {
			return true
		}
	}

	return false
}
