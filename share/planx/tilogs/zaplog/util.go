package zaplog

import (
	"fmt"
	"net"
	"os"

	net2 "github.com/nghichtu91/platform/share/planx/net"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

const httpLvlTemplate = `
#!/bin/bash

if [ $1 == "get" ]
then
    curl "http://127.0.0.1:%s/"
fi

if [ $1 == "set" ]
then
    curl -X PUT -H "Content-Type: application/json" -d "{\"level\":\"$2\"}" "http://127.0.0.1:%s/"
fi
`

const fileName = "hotchangelevel.sh"

func hotLevel() net.Listener {
	lis_Internal := net2.TryListen()
	addr := lis_Internal.Addr()
	port := fmt.Sprintf("%d", addr.(*net.TCPAddr).Port)

	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		tilogs.L().Errorf("hotchangelevel start failed for can't open new file: %s", err)
		return nil
	}
	f.Write([]byte(fmt.Sprintf(httpLvlTemplate, port, port)))
	f.Close()

	return lis_Internal
}
