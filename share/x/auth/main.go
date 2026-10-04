package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"

	"github.com/urfave/cli"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/version"
	"github.com/nghichtu91/platform/share/x/auth/cmds"

	//_ "github.com/nghichtu91/platform/share/x/auth/cmds/allinone"
	_ "github.com/nghichtu91/platform/share/x/auth/cmds/allinone"
	_ "github.com/nghichtu91/platform/share/x/auth/cmds/auth"
	_ "github.com/nghichtu91/platform/share/x/auth/cmds/login"
)

func main() {
	defer tilogs.Close()

	app := cli.NewApp()

	app.Version = version.GetVersion()
	app.Name = "AuthApix"
	app.Usage = fmt.Sprintf("Auth Http Api Server. version: %s", version.GetVersion())
	app.Author = "ZhangZhen"
	app.Email = "zhangzhen@taiyouxi.cn"

	cmds.InitCommands(&app.Commands)

	app.Run(os.Args)

	tilogs.L().Infof("auth close success")

}
