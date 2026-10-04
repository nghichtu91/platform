package metrics

import (
	"net"
	"time"

	graphite "github.com/cyberdelia/go-metrics-graphite"
	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func startGolangStats(stop <-chan struct{}) {
	tilogs.L().Infof("metric.golang stats started.")

	if !_cfg.GraphiteValid {
		tilogs.L().Infof("metric.golang stats exit, GraphiteValid is false .")
		return
	}

	if _cfg.GraphiteHost == "" {
		tilogs.L().Infof("metric.golang stats exit, GraphiteHost is null .")
		return
	}

	if _prefix == "" {
		_prefix = "YouForgotPrefix"
	}
	_gcStats := gm.NewRegistry()

	if _cfg.GraphiteGCStats || _cfg.GraphiteMemStats {
		if _cfg.GraphiteGCStats {
			gm.RegisterDebugGCStats(_gcStats)
		}
		if _cfg.GraphiteMemStats {
			gm.RegisterRuntimeMemStats(_gcStats)
		}

		debugTick := time.NewTicker(5e9) // 5e9 == 5 seconds
		go func() {
			tilogs.L().Debugf("debugTick Start")
			for {
				select {
				case <-stop:
					debugTick.Stop()
					tilogs.L().Debugf("debugTick stop")
					return
				case <-debugTick.C:
					//tilogs.Trace("debugTick comes")
					if _cfg.GraphiteGCStats {
						gm.CaptureDebugGCStatsOnce(_gcStats)
					}
					if _cfg.GraphiteMemStats {
						gm.CaptureRuntimeMemStatsOnce(_gcStats)
					}
				}
			}
		}()
	}

	//if _cfg.GraphiteOutPutStdErr {
	//gm.Log(_gcStats, 60e9, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
	//}
	if _cfg.GraphiteHost != "" {
		addr, _ := net.ResolveTCPAddr("tcp", _cfg.GraphiteHost)
		graphiteDebugTick := time.NewTicker(time.Duration(_cfg.GraphiteFlushInterval))
		graphiteDebugConfig := graphite.Config{
			Addr:          addr,
			Registry:      _gcStats,
			FlushInterval: time.Duration(_cfg.GraphiteFlushInterval),
			Prefix:        _prefix,
			DurationUnit:  time.Millisecond,
			Percentiles:   []float64{0.5, 0.75, 0.99, 0.999},
		}
		go func() {
			tilogs.L().Debugf("graphiteDebugTick Start")
			for {
				select {
				case <-stop:
					tilogs.L().Debugf("graphiteDebugTick Stop")
					graphiteDebugTick.Stop()
					return
				case <-graphiteDebugTick.C:
					// tilogs.Trace("graphiteDebugTick comes")
					if err := graphite.Once(graphiteDebugConfig); err != nil {
						tilogs.L().Warnf("GraphiteDebugTick error with %s", err.Error())
					}
				}
			}
		}()

		graphiteCCUTick := time.NewTicker(time.Second)
		graphiteCCUConfig := graphite.Config{
			Addr:          addr,
			Registry:      _goStatsPerSeond,
			FlushInterval: time.Second,
			Prefix:        _prefix,
			DurationUnit:  time.Millisecond,
			Percentiles:   []float64{0, 0.25, 0.5, 0.75, 1},
		}
		go func() {
			tilogs.L().Debugf("graphiteCCUTick Start")
			for {
				select {
				case <-stop:
					tilogs.L().Debugf("graphiteCCUTick Stop")
					graphiteCCUTick.Stop()
					return
				case <-graphiteCCUTick.C:
					// tilogs.Trace("graphiteCCUTick comes")
					if err := graphite.Once(graphiteCCUConfig); err != nil {
						tilogs.L().Warnf("graphiteCCUTick error with %s", err.Error())
					}
				}
			}
		}()
		graphiteCustomTick := time.NewTicker(time.Second)
		graphiteCustomConfig := graphite.Config{
			Addr:          addr,
			Registry:      _goCurstomStatsPerSecond,
			FlushInterval: time.Second,
			Prefix:        _prefix + ".custom",
			DurationUnit:  time.Millisecond,
			Percentiles:   []float64{0, 0.25, 0.5, 0.75, 1},
		}
		go func() {
			tilogs.L().Debugf("graphiteCCUTick Start")
			for {
				select {
				case <-stop:
					tilogs.L().Debugf("graphiteCCUTick Stop")
					graphiteCustomTick.Stop()
					return
				case <-graphiteCustomTick.C:
					if err := graphite.Once(graphiteCustomConfig); err != nil {
						tilogs.L().Warnf("graphiteCustomTick error with %s", err.Error())
					}
				}
			}
		}()
	}
}
