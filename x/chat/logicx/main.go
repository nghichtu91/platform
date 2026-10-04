package main

import (
	"os"

	"github.com/urfave/cli"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/version"
	"github.com/nghichtu91/platform/share/x/chat/logicx/cmds"
	_ "github.com/nghichtu91/platform/share/x/chat/logicx/cmds/chat"
)

func main() {
	app := cli.NewApp()
	app.Version = version.GetVersion()
	app.Name = "logic"
	app.Author = "dAwn_"
	app.Email = "gaoxifeng@taiyouxi.cn"

	cmds.InitCommands(&app.Commands)

	app.Run(os.Args)

	tilogs.L().Infof("logic close success")
}
