package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/urfave/cli"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/version"
	"github.com/nghichtu91/platform/share/x/notice/cmds"
	_ "github.com/nghichtu91/platform/share/x/notice/cmds/notice"
)

func main() {
	defer tilogs.Close()

	app := cli.NewApp()

	app.Version = version.GetVersion()
	app.Name = "notice"
	app.Usage = fmt.Sprintf("Notice Api Server. version: %s", version.GetVersion())
	app.Author = "ZhangZhen"
	app.Email = "zhangzhen@taiyouxi.cn"

	cmds.InitCommands(&app.Commands)

	app.Run(os.Args)

	tilogs.L().Infof("notice close success")

}
