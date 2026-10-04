package main

import (
	"os"

	"github.com/urfave/cli"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/version"
	"github.com/nghichtu91/platform/share/x/chat/cometx/cmds"
	_ "github.com/nghichtu91/platform/share/x/chat/cometx/cmds/chat"
)

func main() {
	app := cli.NewApp()
	app.Version = version.GetVersion()
	app.Name = "comet"
	app.Author = "yangzhenyu"
	app.Email = "yangzhenyu@taiyouxi.cn"

	cmds.InitCommands(&app.Commands)

	app.Run(os.Args)

	tilogs.L().Infof("comet close success")
}
