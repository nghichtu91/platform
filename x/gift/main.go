package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/nghichtu91/platform/share/x/gift/cmds"

	"github.com/urfave/cli"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/version"

	_ "github.com/nghichtu91/platform/share/x/gift/cmds/allinone"
)

func main() {
	defer tilogs.Close()

	app := cli.NewApp()
	app.Version = version.GetVersion()
	app.Name = "gift"
	app.Usage = fmt.Sprintf("Gift Https Api Server. version: %s", version.GetVersion())
	app.Author = "ZhangZhen"
	app.Email = "zhangzhen@taiyouxi.cn"

	cmds.InitCommands(&app.Commands)

	app.Run(os.Args)

	tilogs.L().Infof("gift close success")
}
