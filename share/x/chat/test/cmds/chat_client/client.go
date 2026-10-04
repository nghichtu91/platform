package chat_client

import (
	"time"

	"github.com/urfave/cli"
	"github.com/nghichtu91/platform/share/planx/nats_cli"
	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/chat/test/client"
	"github.com/nghichtu91/platform/share/x/chat/test/cmds"
	"github.com/nghichtu91/platform/share/x/chat/test/user_info"
)

func init() {
	cmds.Register(&cli.Command{
		Name:   "start",
		Usage:  "启动测试client",
		Action: Start,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config, c",
				Value: "config.toml",
				Usage: "Onland Configuration toml config, in {CWD}/conf/ or {AppPath}/conf",
			},
			cli.StringFlag{
				Name:  "user, u",
				Value: "gaoxifeng",
				Usage: "player id",
			},
			cli.StringFlag{
				Name:  "addr, a",
				Value: "localhost:10001",
				Usage: "commet server ip addr",
			},
			cli.StringFlag{
				Name:  "multi, m",
				Value: "30000",
				Usage: "multi connect with start id 10000",
			},
			cli.StringFlag{
				Name:  "number, n",
				Value: "0",
				Usage: "multi connect count",
			},
			cli.StringFlag{
				Name:  "gid, g",
				Value: "12",
				Usage: "gid",
			},
		},
	})
}

//Start 启动chat 测试 client
func Start(c *cli.Context) {
	defer nats_cli.CloseNats()
	defer signalhandler.OnClose()

	// 初始化一些全局信息
	user_info.InitUser(c.String("user"))
	user_info.InitConnect(c.String("addr"))
	user_info.SetMultiStartId(c.Int("multi"))
	user_info.SetMultiCount(c.Int("number"))
	user_info.SetGid(c.Int("gid"))

	if err := nats_cli.InitNats("0", "nats://127.0.0.1:4222", false); err != nil {
		tilogs.L().Errorf("InitNats err %s, natsUrl %s", err.Error(), "nats://127.0.0.1:4222")
		return
	}
	signalhandler.SignalKillFunc(func() { nats_cli.CloseNats() })

	client.Do()

	tilogs.L().Infof("Start with local %s", time.Local.String())
	tilogs.L().Infof(".........chat client exit........")
}
