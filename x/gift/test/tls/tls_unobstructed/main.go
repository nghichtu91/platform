package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"io/ioutil"
)

func main() {
	// 启动gin服务。
	g := gin.Default()
	g.GET("/Hello", func(g *gin.Context) {
		g.String(200, "Aloha")
	})

	pool := x509.NewCertPool()
	caCertPath := "./ca.crt"

	caCrt, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		fmt.Println("ReadFile err:", err)
		return
	}
	pool.AppendCertsFromPEM(caCrt)

	server := endless.NewServer("localhost:8081", g)
	server.TLSConfig = &tls.Config{
		ClientCAs:  pool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	err2 := server.ListenAndServeTLS("./server.crt", "./server.key")

	if err2 != nil {
		fmt.Println(err2)
	}
}
