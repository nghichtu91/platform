package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	_ "net/http/pprof"

	"github.com/nghichtu91/platform/share/planx/version"
	"github.com/nghichtu91/platform/share/x/gatex/cmds"
	_ "github.com/nghichtu91/platform/share/x/gatex/cmds/gate"
)

func main() {

	defer tilogs.Close()

	app := cli.NewApp()
	app.Version = version.GetVersion()
	app.Name = "gate"
	app.Usage = fmt.Sprintf("Ticore game company gate server. gate ver(%s) \n",
		version.GetVersion())
	app.Author = "ZhangZhen"
	app.Email = "zhangzhen@taiyouxi.cn"

	cmds.InitCommands(&app.Commands)

	//tilogs.L().Infof("GOMAXPROCS is %d", runtime.GOMAXPROCS(0))
	//tilogs.L().Infof("Version is %s", app.Version)
	//runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	app.Run(os.Args)

	tilogs.L().Infof("gate close success")
}
