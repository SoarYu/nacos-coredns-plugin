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
	"strings"

	"github.com/coredns/caddy"
	os "os"
	"testing"
)

func TestNacosParse(t *testing.T) {
	tests := []struct {
		input              string
		expectedPath       string
		expectedEndpoint   string
		expectedErrContent string // substring from the expected error. Empty for positive cases.
	}{
		{
			`nacos {
					nacos_namespaceId public
					nacos_server_host console.nacos.io:8848
				  }
`, "skydns", "localhost:300", "",
		},
	}

	os.Unsetenv("nacos_server_list")

	for _, test := range tests {
		c := caddy.NewTestController("dns", test.input)
		_, err := NacosParse(c)
		if err != nil {
			t.Error("Failed to get instance.")
		} else {
			if strings.Compare(GrpcClient.namespaceId, "") != 0 {
				t.Fatal("Failed")
			}
			var passed bool
			for _, item := range GrpcClient.serverConfigs {
				if strings.Compare(item.IpAddr, "console.nacos.io") == 0 && item.Port == 8848 {
					t.Log("Passed")
					passed = true
				}
			}
			if !passed {
				t.Fatal("Failed")
			}
		}
	}
}
