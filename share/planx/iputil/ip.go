package iputil

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/astaxie/beego/httplib"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/cenk/backoff"
)

// GetPublicIP 这个函数实际上是返回的 公网 ip+端口
func GetPublicIP(pip, listen string) (publicAddr string) {
	if _, port, err := net.SplitHostPort(listen); err != nil {
		panic(fmt.Sprintf("[GateServer] config listen %s has problem(aws). %s", listen, err.Error()))
	} else {
		if _, _, err := net.SplitHostPort(pip); err == nil {
			return pip
		}
		defer func() {
			// ip:port 组成的 addr, 理论上使用ResolveTCPAddr/ResolveUDPAddr都可以解析成功
			tilogs.L().Debugf("GetPublicIP pip %s publicAddr %s", pip, publicAddr)
			_, err := net.ResolveTCPAddr("", publicAddr)
			if err != nil {
				errMsg := fmt.Sprintf("GetPublicIP %s has problem. %s. %s", pip, publicAddr, err.Error())
				tilogs.L().Alarm(errMsg)
				panic(fmt.Sprintf(errMsg))
			}
		}()
		switch pip {
		case planx.CloudServerName_AWS, "AWS":
			if awsip, e := AwsGetPublicIP(); e != nil {
				panic(fmt.Sprintf("[GateServer] config listen has problem(aws). %s", e.Error()))
			} else {
				return net.JoinHostPort(awsip, port)
			}
		case planx.CloudServerName_QCCloud, "QCLOUD":
			if awsip, e := QcloudGetPublicIP(); e != nil {
				panic(fmt.Sprintf("[GateServer] config listen has problem(aws). %s", e.Error()))
			} else {
				return net.JoinHostPort(awsip, port)
			}
		case planx.CloudServerName_Docker, "Docker", "DOCKER":
			if envListenIP := os.Getenv("HostPublicIP"); "" != envListenIP {
				if envPort8667 := os.Getenv("PORT_8667"); "" != envPort8667 {
					return net.JoinHostPort(envListenIP, envPort8667)
				}
				panic(fmt.Sprintf("[GateServer] config listen has problem(docker). no PORT_8667 environment"))
			}
			panic(fmt.Sprintf("[GateServer] config listen has problem(docker). no HostPublicIP environment"))
		case planx.CloudServerName_Tencent:
			if ip, e := TencentGetPublicIP(); e != nil {
				panic(fmt.Sprintf("get public ip. %s", e.Error()))
			} else {
				return net.JoinHostPort(ip, port)
			}
		case planx.CloudServerName_Aliyun:
			if ip, e := AliyunGetPublicIP(); e != nil {
				panic(fmt.Sprintf("get public ip. %s", e.Error()))
			} else {
				return net.JoinHostPort(ip, port)
			}
		case planx.CloudServerName_Azure:
			if ip, e := AzureGetPublicIP(); e != nil {
				panic(fmt.Sprintf("get public ip. %s", e.Error()))
			} else {
				return net.JoinHostPort(ip, port)
			}
		case planx.CloudServerName_HuaWei:
			if ip, e := HuaweiGetPublicIP(); e != nil {
				panic(fmt.Sprintf("get public ip. %s", e.Error()))
			} else {
				return net.JoinHostPort(ip, port)
			}
		case planx.CloudServerName_Google:
			if ip, e := GoogleGetPublicIP(); e != nil {
				panic(fmt.Sprintf("get public ip. %s", e.Error()))
			} else {
				return net.JoinHostPort(ip, port)
			}
		default:
			host, err := ExternalIP()
			if err != nil {
				panic(fmt.Sprintf("[GateServer] config listen has problem(externalIP). %s", err.Error()))
			} else {
				return net.JoinHostPort(host, port)
			}
		}
	}
}

func GetPrivateIP(pip string) (privateIp string) {
	var skip bool

	defer func() {
		if skip {
			return
		}
		tilogs.L().Debugf("GetPrivateIP pip %s privateIp %s", pip, privateIp)
		if ip := net.ParseIP(privateIp); ip == nil {
			errMsg := fmt.Sprintf("GetPrivateIP %s has problem. %s", pip, privateIp)
			tilogs.L().Alarm(errMsg)
			panic(errMsg)
		}
	}()

	switch strings.ToLower(pip) {
	case planx.CloudServerName_AWS:
		if awsip, e := AwsGetPrivateIP(); e != nil {
			panic(fmt.Sprintf("GetPrivateIP AwsGetPublicIP has problem(aws). %s", e.Error()))
		} else {
			return awsip
		}
	case planx.CloudServerName_Tencent:
		if ip, err := TencentGetPrivateIP(); err != nil {
			panic(fmt.Sprintf("get tencent ip error %v", err))
		} else {
			return ip
		}
	case planx.CloudServerName_Aliyun:
		if ip, err := AliyunGetPrivateIP(); err != nil {
			panic(fmt.Sprintf("get aliyun ip error %v", err))
		} else {
			return ip
		}
	case planx.CloudServerName_Azure:
		if ip, err := AzureGetPrivateIP(); err != nil {
			panic(fmt.Sprintf("get azure ip error %v", err))
		} else {
			return ip
		}
	case planx.CloudServerName_HuaWei:
		if ip, err := HuaweiGetPrivateIP(); err != nil {
			panic(fmt.Sprintf("get huawei ip error %v", err))
		} else {
			return ip
		}
	case planx.CloudServerName_Google:
		if ip, err := GoogleGetPrivateIP(); err != nil {
			panic(fmt.Sprintf("get google ip error %v", err))
		} else {
			return ip
		}
	case "local":
		host, err := ExternalIP()
		if err != nil {
			panic(fmt.Sprintf("GetPrivateIP has problem(externalIP). %s", err.Error()))
		} else {
			return host
		}
	default:
		skip = true
		if nil == net.ParseIP(pip) {
			return ""
		}
		return pip
	}
}

func QcloudGetPublicIP() (string, error) {
	var ret string
	operation := func() error {
		response, err := http.Get("http://metadata.tencentyun.com/meta-data/public-ipv4")
		if err != nil {
			tilogs.L().Errorf("QcloudGetPublicIP get ipv4 failed. %s", err.Error())
			return err
		}
		defer response.Body.Close()
		body, err := ioutil.ReadAll(response.Body)
		ret = string(body)
		if err != nil {
			tilogs.L().Errorf("QcloudGetPublicIP get ipv4 failed. %s", err.Error())
			return err
		}
		return nil
	}
	ebo := backoff.NewExponentialBackOff()
	ebo.MaxElapsedTime = 10 * time.Second
	err := backoff.Retry(operation, ebo)
	if err != nil {
		// Handle error.
		tilogs.L().Errorf("QcloudGetPublicIP get ipv4 finally failed. %s", err.Error())
		return "", err
	}
	return ret, nil
}

func AwsGetPublicIP() (string, error) {
	sessDefaults := session.New()
	meta := ec2metadata.New(sessDefaults)
	ip, err := meta.GetMetadata("public-ipv4")
	if err != nil {
		return "", err
	}
	return ip, nil
}

func AwsGetPrivateIP() (string, error) {
	sessDefaults := session.New()
	meta := ec2metadata.New(sessDefaults)
	ip, err := meta.GetMetadata("local-ipv4")
	if err != nil {
		return "", err
	}
	return ip, nil
}

func TencentGetPrivateIP() (string, error) {
	req := httplib.Get("http://metadata.tencentyun.com/meta-data/local-ipv4")
	ip, err := req.String()
	if err != nil {
		return "", err
	} else {
		return ip, nil
	}
}

func TencentGetPublicIP() (string, error) {
	req := httplib.Get("http://metadata.tencentyun.com/meta-data/public-ipv4")
	ip, err := req.String()
	if err != nil {
		return "", err
	} else {
		return ip, nil
	}
}

func AliyunGetPrivateIP() (string, error) {
	req := httplib.Get("http://100.100.100.200/latest/meta-data/private-ipv4")
	ip, err := req.String()
	if err != nil {
		return "", err
	} else {
		return ip, nil
	}
}

func AliyunGetPublicIP() (string, error) {
	req := httplib.Get("http://100.100.100.200/latest/meta-data/eipv4")
	ip, err := req.String()
	if err != nil {
		return "", err
	} else {
		return ip, nil
	}
}

func AzureGetPrivateIP() (string, error) {
	req := httplib.Get("http://169.254.169.254/metadata/instance/network/interface/0/ipv4/ipAddress/0/privateIpAddress?api-version=2017-08-01&format=text")
	req.Header("Metadata", "true")
	ip, err := req.String()
	if err != nil {
		return "", err
	} else {
		return ip, nil
	}
}

func AzureGetPublicIP() (string, error) {
	req := httplib.Get("http://169.254.169.254/metadata/instance/network/interface/0/ipv4/ipAddress/0/publicIpAddress?api-version=2017-08-01&format=text")
	req.Header("Metadata", "true")
	ip, err := req.String()
	if err != nil {
		return "", err
	} else {
		return ip, nil
	}
}

func HuaweiGetPrivateIP() (string, error) {
	req := httplib.Get("http://169.254.169.254/latest/meta-data/local-ipv4")
	ip, err := req.String()
	if err != nil {
		return "", err
	} else {
		return ip, nil
	}
}

func HuaweiGetPublicIP() (string, error) {
	req := httplib.Get("http://169.254.169.254/latest/meta-data/public-ipv4")
	ip, err := req.String()
	if err != nil {
		return "", err
	} else {
		return ip, nil
	}
}

func GoogleGetPrivateIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("GoogleGetPrivateIP not found ip")
}

func GoogleGetPublicIP() (string, error) {
	req := httplib.Get("http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip")
	req.Header("Metadata-Flavor", "Google")
	ip, err := req.String()
	if err != nil {
		return "", err
	} else {
		return ip, nil
	}
}

func ExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	resIP := make([]string, 0, 1)
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
			resIP = append(resIP, ip.String())
		}
	}
	if len(resIP) == 1 {
		return resIP[0], nil
	} else if len(resIP) > 1 {
		conn, err := net.Dial("udp", "8.8.8.8:80")
		if err != nil {
			return "", err
		}
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		return localAddr.IP.String(), nil
	}
	return "", errors.New("are you connected to the network?")
}

func ParseHostByUrl(ul string) (string, error) {
	u, err := url.Parse(ul)
	if err != nil {
		return "", err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return "", err
	}
	return host, nil
}
