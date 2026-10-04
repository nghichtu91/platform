package net

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strings"

	"github.com/nghichtu91/platform/share/planx/etcd"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

const (
	HealthPortStart = 6000
	HealthPortEnd   = 6949
	PProfPortStart  = 6950
	PProfPortEnd    = 7050
	PortStart       = 7100
	PortEnd         = 7999
)

/*
尝试可用的端口进行监听，目前只用于内网服务之间通信用，所以先不支持tls
*/
func TryListen() net.Listener {
	for port := PortStart; port <= PortEnd; port++ {
		url := fmt.Sprintf(":%d", port)
		lis, err := net.Listen("tcp", url)
		if err != nil {
			continue
		}
		return lis
	}
	return nil
}

func ListenedPort(lis net.Listener) int {
	addr := lis.Addr()
	switch t := addr.(type) {
	case *net.TCPAddr:
		return t.Port
	case *net.UDPAddr:
		return t.Port
	case *net.IPAddr:
		return 0
	case *net.UnixAddr:
		return 0
	default:
		tilogs.L().Errorf("ListenedPort unknown type %v", t)
		return 0
	}
}

// PProfStart 开启pprof服务，自动探测的端口
// pprofEtcdKey,internalIp参数用于GM拉取pprof性能样本，不用可传空串
func PProfStart(pprofEtcdKey string, internalIp string) func() {
	go func() {
		for port := PProfPortStart; port <= PProfPortEnd; port++ {
			url := fmt.Sprintf(":%d", port)
			tilogs.L().Infof("pprof try listen port %d ", port)
			mode := os.FileMode(0644)
			f, err := os.OpenFile("pprof", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				tilogs.L().Errorf("pprof start failed for can't open new file: %s", err)
				return
			}
			f.Write([]byte(url))
			f.Close()

			if pprofEtcdKey != "" {
				err := etcd.Put(pprofEtcdKey, internalIp+url)
				if err != nil {
					tilogs.L().Errorf("pprof port:%s put to etcd err:%v", url, err)
				}
			}
			if err := http.ListenAndServe(url, nil); err != nil {
				// tilogs.L().Warnf("pprof server try %s err %v", url, err)
			} else {
				// success

				return
			}
		}
		// 所有端口都用完了，仍然没找到可用的端口
		panic(fmt.Errorf("pprof can not find available port"))
	}()
	runtime.SetBlockProfileRate(2)
	return func() {
		if pprofEtcdKey != "" {
			err := etcd.Delete(pprofEtcdKey)
			tilogs.L().Infof("pprof port:%s delete from etcd err:%v", pprofEtcdKey, err)
		}
	}
}

/*
服务发现的grpc health cheaking用的server端监听端口
*/
func TryListenForServerHealth() net.Listener {
	for port := HealthPortStart; port <= HealthPortEnd; port++ {
		url := fmt.Sprintf(":%d", port)
		tilogs.L().Infof("ServerDiscovery try listen port %d ", port)
		mode := os.FileMode(0644)
		f, err := os.OpenFile("health", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			tilogs.L().Errorf("ServerDiscovery start failed for can't open new file: %s", err)
			return nil
		}
		f.Write([]byte(url))
		f.Close()

		lis, err := net.Listen("tcp", url)
		if err != nil {
			continue
		}
		return lis
	}
	return nil
}

// 由于内网端口是自动发现模式，服务运行起来后不知道确切的监听的那个端口，不太方便
// 这个方法将地址写到文件中，文件名"tryaddr"
func DebugForTryAddr(addr string) {
	mode := os.FileMode(0644)
	f, err := os.OpenFile("tryaddr", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		tilogs.L().Errorf("pprof start failed for can't open new file: %s", err)
		return
	}
	f.Write([]byte(addr))
	f.Close()
}

var netErrClosed = "use of closed network connection"

func NetIsClosed(err error) bool {
	if strings.HasSuffix(err.Error(), netErrClosed) {
		return true
	}
	return false
}

func GetExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

const JsonPostTyp = "application/json; charset=utf-8"

func HttpPost(url, typ string, data []byte) ([]byte, error) {
	body := bytes.NewBuffer([]byte(data))
	resp, err := http.Post(url, typ, body)
	if err != nil {
		return []byte{}, err
	}

	defer resp.Body.Close()
	body_res, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Println(err.Error())
		return []byte{}, err
	}

	return body_res, nil
}

func HttpPostWCode(url, typ string, data []byte) (int, []byte, error) {
	body := bytes.NewBuffer([]byte(data))
	resp, err := http.Post(url, typ, body)
	if err != nil {
		return 404, []byte{}, err
	}

	defer resp.Body.Close()
	body_res, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Println(err.Error())
		return 404, []byte{}, err
	}

	return resp.StatusCode, body_res, nil
}

// 整形转换成字节
func IntToBytes(n int) []byte {
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

// 字节转换成整形
func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)
	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return int(x)
}

// RemoveIPDotsAndColon 去掉ip:port形式地址中的点和冒号
func RemoveIPDotsAndColon(ip string) string {
	ip = strings.Replace(ip, ".", "_", -1)
	ip = strings.Replace(ip, ":", "_", -1)
	return ip
}
