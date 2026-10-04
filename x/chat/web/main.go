// chat web --- go run main.go allinone
package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/urfave/cli"
	"github.com/nghichtu91/platform/share/planx/version"
	"github.com/nghichtu91/platform/share/x/chat/web/cmds"
	_ "github.com/nghichtu91/platform/share/x/chat/web/cmds/allinone"
)

func main() {

	app := cli.NewApp()

	app.Version = version.GetVersion()
	app.Name = "Chat_Web"
	app.Usage = fmt.Sprintf("Chat web server")
	app.Author = "wangzhiyang"
	app.Email = "wangzhiyang@taiyouxi.cn"

	cmds.InitCommands(&app.Commands)

	app.Run(os.Args)
}
