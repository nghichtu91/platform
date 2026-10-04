package main

import (
	"os"

	"github.com/urfave/cli"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
	"github.com/nghichtu91/platform/share/planx/version"
	"github.com/nghichtu91/platform/share/x/chat/test/cmds"
	_ "github.com/nghichtu91/platform/share/x/chat/test/cmds/chat_client"
)

func main() {
	app := cli.NewApp()
	app.Version = version.GetVersion()
	app.Name = "client"
	app.Author = "dAwn_"
	app.Email = "fuck!"

	zaplog.InitZapLog("log.toml", map[string]string{
		"service": "test",
		"subcmd":  "test",
	}, "test")

	cmds.InitCommands(&app.Commands)

	app.Run(os.Args)

	tilogs.L().Infof("chat client close success")
}
