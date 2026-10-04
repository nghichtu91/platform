package grpclog

import (
	"io"

	"google.golang.org/grpc/grpclog"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type logger struct {
	log func(tilogs.TiLogger, string, ...interface{})
}

func (l *logger) Write(data []byte) (n int, _ error) {
	if l == nil || l.log == nil {
		return
	}
	l.log(tilogs.L(), string(data))
	n = len(data)
	return
}

func newInfoLogger() io.Writer {
	return &logger{log: tilogs.TiLogger.Infof}
}

func newWarnLogger() io.Writer {
	return &logger{log: tilogs.TiLogger.Warnf}
}

func newErrorLogger() io.Writer {
	return &logger{log: tilogs.TiLogger.Errorf}
}

//SetGRPCLogger 将GRPC的日志输出重定向到zapLogger
//gRPC的连接重连日志是WARN级别的, 为了能在Sentry上记录重连的日志, 这里将其对接成我们的ERROR级别日志
func SetGRPCLogger() {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(newInfoLogger(), newErrorLogger(), newErrorLogger()))
}
