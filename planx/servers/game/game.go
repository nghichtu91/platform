package game

import "github.com/nghichtu91/platform/share/planx/servers"

type PreparePlayer func(accountid string, clientInfo string) (Player2, bool)

type Server interface {
	Stop()
	Start(mux *servers.Mux)
}
