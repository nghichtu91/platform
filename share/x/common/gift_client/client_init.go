package gift_client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var client *http.Client

var giftCodeUrl string // 礼包码的访问路径。

// 初始化礼包码服务器请求客户端。
//
// Param-caPath: CA证书路径; Param-clientCertPath: 客户端证书路径;
// Param-clientKeyPath: 客户端秘钥路径; Param-url: 礼包码服务器地址。
func InitClient(caPath, clientCertPath, clientKeyPath, url string, dua time.Duration) error {
	// 读入CA证书文件。
	caCrt, errRead := ioutil.ReadFile(caPath)
	if errRead != nil {
		return fmt.Errorf("InitClient ReadCACert fail %v", errRead)
	}

	// 加入CA验证池。
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCrt)

	// 加载客户端证书和秘钥。
	cliCrt, errLoad := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if errLoad != nil {
		return fmt.Errorf("InitClient LoadX509KeyPair fail %v", errLoad)
	}

	// 设置客户端Https访问请求。
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            pool,
			Certificates:       []tls.Certificate{cliCrt},
			InsecureSkipVerify: true,
		},
	}
	giftCodeUrl = url
	client = &http.Client{Transport: tr}

	// 向GiftServer的请求超时时间。
	client.Timeout = dua
	return nil
}
