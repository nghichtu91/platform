package logic

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
)

var client *http.Client

// 初始化礼包码服务器请求客户端。
func InitClient() error {
	// 读入CA证书文件。
	caCrt, errRead := ioutil.ReadFile("./ca.crt")
	if errRead != nil {
		return fmt.Errorf("InitClient ReadCACert fail %v", errRead)
	}

	// 加入CA验证池。
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCrt)

	// 加载客户端证书和秘钥。
	cliCrt, errLoad := tls.LoadX509KeyPair("./client.crt", "./client.key")
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
	client = &http.Client{Transport: tr}
	return nil
}
