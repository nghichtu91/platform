package tilogs

import "fmt"

const (
	TagUserID     = "accountid"  // log的json中用来存放用户id或工会id等id用的字段名
	TagServerType = "ServerType" // 用来标明服务类型的字段名
	TagServerId   = "server_id"  // 用来标明服务id的字段名
	TagGid        = "gid"        // 用来表明大区id
)

var (
	_logger TiLogger
)

type TiLogger interface {
	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Alarm(format string, v ...interface{}) // 该接口调用 Errorf 并添加 alarm 关键词, 触发 sentry -> 飞书 的报警通知
	Error(msg string)
	LevelEnable(level string) bool
	With(args ...interface{}) TiLogger
	WithUser(acid string, args ...interface{}) TiLogger
	Close()
}

func L() TiLogger {
	return _logger
}

func SetTiLogger(l TiLogger) {
	_logger = l
}

func LevelEnable(level string) bool {
	if _logger != nil {
		return _logger.LevelEnable(level)
	}
	return false
}

func Close() {
	if _logger != nil {
		_logger.Close()
	}
}

//PanicCatcher use like: defer PanicCatcher()
func PanicCatcher(format string, v ...interface{}) {
	if err := recover(); err != nil {
		if _logger != nil {
			msg := fmt.Sprintf(format, v...)
			errmsg := fmt.Sprintf("%s, err:%v", msg, err)
			_logger.Errorf(errmsg)
		}
	}
}

//PanicCatcherWithFunc use like: defer PanicCatcherWithFunc()
func PanicCatcherWithFunc(f func(), format string, v ...interface{}) {
	if err := recover(); err != nil {
		if _logger != nil {
			msg := fmt.Sprintf(format, v...)
			errmsg := fmt.Sprintf("%s, err:%v", msg, err)
			_logger.Errorf(errmsg)
		}
		f()
	}
}

//PanicCatcherWithInfo use like: defer PanicCatcherWithInfo()
func PanicCatcherWithInfo(_loggerWithInfo TiLogger, format string, v ...interface{}) {
	if err := recover(); err != nil {
		if _loggerWithInfo != nil {
			msg := fmt.Sprintf(format, v...)
			errmsg := fmt.Sprintf("%s, err:%v", msg, err)
			_loggerWithInfo.Errorf(errmsg)
		}
	}
}
