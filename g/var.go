// Copyright 2018 RosenLo

// Copyright 2017 Xiaomi, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/**
 * This code was originally worte by Xiaomi, Inc. modified by RosenLo.
**/

package g

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/RosenLo/toolkits/file"
	"github.com/RosenLo/toolkits/gnu"
	"github.com/RosenLo/toolkits/str"
	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/toolkits/slice"
)

var Root string

func InitRootDir() {
	var err error
	Root, err = os.Getwd()
	if err != nil {
		log.Fatalln("getwd fail:", err)
	}
}

var LocalIp string

func InitLocalIp() {
	if Config().Heartbeat.Enabled {
		conn, err := net.DialTimeout("udp", "114.114.114.114:53", time.Second*10)
		if err != nil {
			log.Println("get local addr failed, due to: %s", err)
		} else {
			defer conn.Close()
			LocalIp = strings.Split(conn.LocalAddr().String(), ":")[0]
		}
	} else {
		log.Println("hearbeat is not enabled, can't get localip")
	}
}

var (
	HbsClient *SingleConnRpcClient
)

func InitRpcClients() {
	if Config().Heartbeat.Enabled {
		HbsClient = &SingleConnRpcClient{
			RpcServer: Config().Heartbeat.Addr,
			Timeout:   time.Duration(Config().Heartbeat.Timeout) * time.Millisecond,
		}
	}
}

func SendToTransfer(metrics []*model.MetricValue) {
	if len(metrics) == 0 {
		return
	}

	dt := Config().DefaultTags
	if len(dt) > 0 {
		var buf bytes.Buffer
		default_tags_list := []string{}
		for k, v := range dt {
			buf.Reset()
			buf.WriteString(k)
			buf.WriteString("=")
			buf.WriteString(v)
			default_tags_list = append(default_tags_list, buf.String())
		}
		default_tags := strings.Join(default_tags_list, ",")

		for i, x := range metrics {
			buf.Reset()
			if x.Tags == "" {
				metrics[i].Tags = default_tags
			} else {
				buf.WriteString(metrics[i].Tags)
				buf.WriteString(",")
				buf.WriteString(default_tags)
				metrics[i].Tags = buf.String()
			}
		}
	}

	debug := Config().Debug

	if debug {
		log.Printf("=> <Total=%d> %v\n", len(metrics), metrics[0])
	}

	var resp model.TransferResponse
	SendMetrics(metrics, &resp)

	if debug {
		log.Println("<=", &resp)
	}
}

var (
	reportUrls     map[string]string
	reportUrlsLock = new(sync.RWMutex)
)

func ReportUrls() map[string]string {
	reportUrlsLock.RLock()
	defer reportUrlsLock.RUnlock()
	return reportUrls
}

func SetReportUrls(urls map[string]string) {
	reportUrlsLock.RLock()
	defer reportUrlsLock.RUnlock()
	reportUrls = urls
}

var (
	reportPorts     []int64
	reportPortsLock = new(sync.RWMutex)
)

func ReportPorts() []int64 {
	reportPortsLock.RLock()
	defer reportPortsLock.RUnlock()
	return reportPorts
}

func SetReportPorts(ports []int64) {
	reportPortsLock.Lock()
	defer reportPortsLock.Unlock()
	reportPorts = ports
}

var (
	duPaths     []string
	duPathsLock = new(sync.RWMutex)
)

func DuPaths() []string {
	duPathsLock.RLock()
	defer duPathsLock.RUnlock()
	return duPaths
}

func SetDuPaths(paths []string) {
	duPathsLock.Lock()
	defer duPathsLock.Unlock()
	duPaths = paths
}

var (
	// tags => {1=>name, 2=>cmdline}
	// e.g. 'name=falcon-agent'=>{1=>falcon-agent}
	// e.g. 'cmdline=xx'=>{2=>xx}
	reportProcs     map[string]map[int]string
	reportProcsLock = new(sync.RWMutex)
)

func ReportProcs() map[string]map[int]string {
	reportProcsLock.RLock()
	defer reportProcsLock.RUnlock()
	return reportProcs
}

func SetReportProcs(procs map[string]map[int]string) {
	reportProcsLock.Lock()
	defer reportProcsLock.Unlock()
	reportProcs = procs
}

var (
	ips     []string
	ipsLock = new(sync.Mutex)
)

func TrustableIps() []string {
	ipsLock.Lock()
	defer ipsLock.Unlock()
	return ips
}

func SetTrustableIps(ipStr string) {
	arr := strings.Split(ipStr, ",")
	ipsLock.Lock()
	defer ipsLock.Unlock()
	ips = arr
}

func IsTrustable(remoteAddr string) bool {
	ip := remoteAddr
	idx := strings.LastIndex(remoteAddr, ":")
	if idx > 0 {
		ip = remoteAddr[0:idx]
	}

	if ip == "127.0.0.1" {
		return true
	}

	return slice.ContainsString(TrustableIps(), ip)
}

var (
	HostInfo map[string]interface{}
)

func OSBit() string {
	cmd := exec.Command("uname", "-i")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Println("get os bit failed, due to: ", err)
	}
	return out.String()
}

func getRedhatishVersion(contents []string) string {
	c := strings.ToLower(strings.Join(contents, ""))

	if strings.Contains(c, "rawhide") {
		return "rawhide"
	}
	if matches := regexp.MustCompile(`release (\d[\d.]*)`).FindStringSubmatch(c); matches != nil {
		return matches[1]
	}
	return ""
}

func getRedhatishPlatform(contents []string) string {
	c := strings.ToLower(strings.Join(contents, ""))

	if strings.Contains(c, "red hat") {
		return "redhat"
	}
	f := strings.Split(c, " ")

	return f[0]
}

func getLSB(content []string) (string, string) {
	var platformName, platformVersion string
	for _, line := range content {
		fileds := strings.Split(line, "=")
		if len(fileds) < 2 {
			continue
		}
		switch fileds[0] {
		case "DISTRIB_ID":
			platformName = fileds[1]
		case "DISTRIB_RELEASE":
			platformVersion = fileds[1]
		}
	}
	return platformName, platformVersion
}

func PlatformInfo() (string, string, string) {
	var platformName, platformVersion string
	platformType := "1" // Linux
	prefix := "/etc/"
	linuxRelease := str.Concatenate(prefix, "system-release")
	ubuntuRelease := str.Concatenate(prefix, "lsb-release")
	if file.IsExist(linuxRelease) {
		c, err := file.ReadLines(linuxRelease)
		if err == nil {
			platformVersion = getRedhatishVersion(c)
			platformName = getRedhatishPlatform(c)
		}
	} else if file.IsExist(ubuntuRelease) {
		c, err := file.ReadLines(ubuntuRelease)
		if err == nil {
			platformName, platformVersion = getLSB(c)
		}
	}
	return platformType, platformName, platformVersion
}

func CPUInfo() (int, int, string, string) {
	var num, mhz int
	var module string
	hostType := "1" // virtual
	cpu, err := gnu.CpuInfo()
	if err == nil {
		mhz = cpu.MHz
		num = cpu.Num
		module = cpu.Module
		if !cpu.Virtual {
			hostType = "2"
		}
	}
	return num, mhz, module, hostType
}

func MemTotal() uint64 {
	mem, err := gnu.MemInfo()
	if err != nil {
		return uint64(1)
	}
	return mem.MemTotal / 1000 / 1000
}

func DiskTotal() uint64 {
	mem, err := gnu.MemInfo()
	if err != nil {
		return uint64(1)
	}
	return mem.MemTotal
}

func InitHostInfo() {
	hostname, err := Hostname()
	if err != nil {
		hostname = fmt.Sprintf("error:%s", err.Error())
	}
	platformType, platformName, platformVersion := PlatformInfo()
	cpuNum, cpuMhz, cpuModule, hostType := CPUInfo()
	HostInfo = map[string]interface{}{
		"bk_host_name":    hostname,
		"bk_host_innerip": IP(), "import_from": "2", // from agent
		"bk_cpu":        cpuNum,
		"bk_cpu_mhz":    cpuMhz,
		"bk_cpu_module": cpuModule,
		"host_type":     hostType,
		"bk_os_bit":     OSBit(),
		"bk_os_type":    platformType,
		"bk_os_name":    platformName,
		"bk_os_version": platformVersion,
		"bk_mem":        MemTotal(),
	}
}
