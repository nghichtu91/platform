package zaplog

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/bufio.v1"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/nghichtu91/platform/share/planx/config"
	"github.com/nghichtu91/platform/share/planx/githubsrc/zapsentry"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type zapLogConfig struct {
	Level      string `toml:"level"`     // 记录log级别
	Production bool   `toml:"prod"`      // prod or dev
	Sampling   int    `toml:"sampling"`  // 当错误频繁发生，同一个错误1s内最多输出的数量；zap是根据message给每个level设置了4096个不同的错误类型，如果超了会有问题
	SentryDSN  string `toml:"sentryDSN"` // 为空则sentry无效
	// 以下为log落磁盘相关配置，只本地开发需要，远程服务器将有supervisor接管
	LogPathAndName string `toml:"log_path"`    // log文件位置和文件名
	MaxSize        int    `toml:"max_size"`    // 一个log文件最大大小，单位M
	MaxBackups     int    `toml:"max_backups"` // 最多保留多少个log文件
	MaxAge         int    `toml:"max_age"`     // 最多保留多少天前的log文件
	ConsoleEncode  bool   `toml:"console"`     // console
}

var (
	cfg           zapLogConfig
	zapLogLevel   zap.AtomicLevel
	zapSentryTags map[string]string
	zapLogger     *zapLogWrapper
)

func init() {
	// 测试cfg
	cfg = zapLogConfig{
		// LogPathAndName: "./log.log",
		Level:      "debug",
		Production: false,
		Sampling:   100,
		MaxSize:    20,
		MaxBackups: 20,
		MaxAge:     10,
	}
}

func InitDebugLog() {
	InitZapLog("", nil, "")
}

/*
初始化zaplog，配置文件中的level支持热更，其他不支持
logFileName: log文件路径和文件名前缀
sentryTags:需要带这些，如

	service: gamex
	subcmd: allinone
	sid:	[1001,1002]
	mode:	prod

serverType:服务类型，如：gamex
serverId:服务id
*/
func InitZapLog(logFileName string, sentryTags map[string]string, serverType string) *zap.Logger {
	zapSentryTags = sentryTags
	if logFileName != "" {
		if ret := config.NewConfig(logFileName, true, func(lcfgname string, cmd config.LoadCmd) error {
			return loadZapLogConfig(lcfgname, cmd)
		}); ret == nil {
			log.Printf("InitZapLog NewConfig failed!\n")
			zapLogger.Close()
			os.Exit(1)
		}
	} else {
		zapLogger = newZapLog()
	}

	zapLogger.SugaredLogger = zapLogger.SugaredLogger.With(tilogs.TagServerType, serverType)
	tilogs.SetTiLogger(zapLogger)

	// 如果未传入serverType，日志级别热更不开启
	if serverType != "" {
		lis_Internal := hotLevel()
		if lis_Internal == nil {
			tilogs.L().Errorf("hotLevel start failed!")
			zapLogger.Close()
			os.Exit(1)
		}
		go func() {
			if err := http.Serve(lis_Internal, zapLogLevel); err != nil {
				tilogs.L().Errorf("hotLevel Serve start err %s", err.Error())
				zapLogger.Close()
				os.Exit(1)
			}
		}()
	}

	return zapLogger.Desugar()
}

type zapLogWrapper struct {
	*zap.SugaredLogger
	*zap.AtomicLevel
}

func (l *zapLogWrapper) Close() {
	l.Info("close zap log...")
	if l.SugaredLogger != nil {
		l.Info("sync zap log...")
		l.Sync()
	}
}
func (l *zapLogWrapper) LevelEnable(level string) bool {
	var lvl zapcore.Level
	lvl.UnmarshalText([]byte(level))
	return l.AtomicLevel.Enabled(lvl)
}

func (l *zapLogWrapper) With(args ...interface{}) tilogs.TiLogger {
	return &zapLogWrapper{
		SugaredLogger: l.SugaredLogger.With(args...),
		AtomicLevel:   l.AtomicLevel,
	}
}

func (l *zapLogWrapper) Alarm(format string, args ...interface{}) {
	l.Errorf("[alarm]"+format, args...)
}

func (l *zapLogWrapper) WithUser(acid string, args ...interface{}) tilogs.TiLogger {
	return &zapLogWrapper{
		SugaredLogger: l.SugaredLogger.With(tilogs.TagUserID, acid).With(args...),
		AtomicLevel:   l.AtomicLevel,
	}
}

func (l *zapLogWrapper) Error(msg string) {
	l.Errorf(msg)
}

func loadZapLogConfig(cfgFile string, cmd config.LoadCmd) error {
	var _cfg zapLogConfig
	if _, err := toml.DecodeFile(cfgFile, &_cfg); err != nil {
		log.Printf("App config load failed. %s, %s\n", cfgFile, err.Error())
		return err
	} else {
		log.Printf("Config loaded: %s\n", cfgFile)
	}
	switch cmd {
	case config.Load:
		cfg = _cfg
		log.Printf("load ZapLog config %v\n", cfg)
		zapLogger = newZapLog()
	case config.Reload:
		l := zap.DebugLevel
		if err := l.UnmarshalText([]byte(_cfg.Level)); err != nil {
			zapLogger.Errorf("Reload ZapLog level err by %s", _cfg.Level)
			return err
		}
		zapLogLevel.SetLevel(l)
		zapLogger.Infof("Reload ZapLog level %s successful", _cfg.Level)
	}
	return nil
}

/*
初始化并创建zapLogWrapper
*/
func newZapLog() *zapLogWrapper {
	l := zap.DebugLevel
	if err := l.UnmarshalText([]byte(cfg.Level)); err != nil {
		panic(err)
	}
	zapLogLevel = zap.NewAtomicLevelAt(l)

	opts := make([]zap.Option, 0, 4)
	opts = append(opts, zap.AddCaller())
	// level
	stackLevel := zap.ErrorLevel
	if !cfg.Production {
		stackLevel = zap.WarnLevel
		opts = append(opts, zap.Development())
	}
	// stacktrace
	opts = append(opts, zap.AddStacktrace(stackLevel))
	// sample
	if cfg.Sampling > 0 {
		opts = append(opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(core, time.Second, cfg.Sampling, cfg.Sampling)
		}))
	}
	// sentry
	if cfg.SentryDSN != "" {
		opts = append(opts, zapSentryOption(cfg.SentryDSN, zapSentryTags))
	}
	// encoder
	encoder_cfg := zapcore.EncoderConfig{
		TimeKey:        "TS",
		LevelKey:       "L",
		NameKey:        "_logger",
		CallerKey:      "CALLER",
		MessageKey:     "MSG",
		StacktraceKey:  "STACK",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapCallerEncoder,
	}
	var encoder zapcore.Encoder
	if cfg.ConsoleEncode {
		encoder_cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder_cfg.ConsoleSeparator = "  "
		encoder_cfg.EncodeCaller = zapcore.ShortCallerEncoder
		encoder = zapcore.NewConsoleEncoder(encoder_cfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoder_cfg)
	}
	// persist
	// 如果本地开发，设置LogPathAndName，log将输出到文件；如果远程服务器上，不用设置LogPathAndName，stdout的log将有supervisor接管
	stdOutW, _, _ := zap.Open("stdout")
	ws := stdOutW
	if cfg.LogPathAndName != "" {
		zapPersist := zap2Lumberjack()
		ws = zap.CombineWriteSyncers(stdOutW, zapPersist)
	}
	// 重定向log也写到zapPersist中
	log.SetOutput(ws)
	// gin重定向
	gin.DefaultWriter = ws
	gin.DefaultErrorWriter = ws
	// timer flush
	// go func() {
	//	interval := time.Duration(cfg.FlushInterval) * time.Millisecond
	//	timerChan := timeutil.TimerMS.After(interval)
	//	for {
	//		select {
	//		case <-timerChan:
	//			zapPersist.Sync()
	//			timerChan = timeutil.TimerMS.After(interval)
	//		}
	//	}
	// }()
	return &zapLogWrapper{
		SugaredLogger: zap.New(zapcore.NewCore(encoder, ws, zapLogLevel), opts...).Sugar(),
		AtomicLevel:   &zapLogLevel,
	}
}

func zapSentryOption(DSN string, tags map[string]string) zap.Option {
	cfg := zapsentry.Configuration{
		Level: zapcore.ErrorLevel, // when to send message to sentry
		Tags:  tags,
	}
	core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromDSN(DSN))
	// in case of err it will return noop core. so we can safely attach it
	if err != nil {
		panic(fmt.Errorf("failed to init zap sentry"))
	}

	extra := []zap.Field{
		zap.String("runtime.Version", runtime.Version()),
		// zap.Int("runtime.NumCPU", runtime.NumCPU()),
		// zap.Int("runtime.GOMAXPROCS", runtime.GOMAXPROCS(0)), // 0 just returns the current value
		// zap.Int("runtime.NumGoroutine", runtime.NumGoroutine()),
	}
	core = core.With(extra)

	return zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(c, core)
	})
}

func zapCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(strings.Join([]string{caller.TrimmedPath(), runtime.FuncForPC(caller.PC).Name()}, ":"))
}

func zapTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + t.Format("2006-01-02T15:04:05.000Z0700") + "]")
}

func zap2Lumberjack() zapcore.WriteSyncer {
	if cfg.Production {
		return &forZapBufWriter{bufio.NewWriter(&lumberjack.Logger{
			Filename:   cfg.LogPathAndName,
			MaxSize:    cfg.MaxSize, // megabytes
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge, // days
			LocalTime:  true,
		})}
	} else {
		return &forZapWriter{&lumberjack.Logger{
			Filename:   cfg.LogPathAndName,
			MaxSize:    cfg.MaxSize, // megabytes
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge, // days
			LocalTime:  true,
		}}
	}

}

type forZapBufWriter struct {
	*bufio.Writer
}

func (w *forZapBufWriter) Sync() error {
	return w.Flush()
}

type forZapWriter struct {
	io.Writer
}

func (w *forZapWriter) Sync() error {
	return nil
}

// DebugFilterLogs 关闭所有日志，仅打印fatal级别日志
// 一般用于性能测试过滤不必要的日志
func DebugFilterLogs() {
	zapLogLevel.SetLevel(zap.FatalLevel)
}

// SetOnlyErrorLogs 只打印Error级别错误
// 用于teamcity打表等需要过滤大部分Info信息的情况
func SetOnlyErrorLogs() {
	zapLogLevel.SetLevel(zap.ErrorLevel)
}

func DebugSetLogLevel(level zapcore.Level) {
	zapLogLevel.SetLevel(level)
}

// DebugConsoleEncode 控制台格式输出. 必须要在zap log初始化之前调用
func DebugConsoleEncode() {
	cfg.ConsoleEncode = true
}

// TestLogger 测试专用
func TestLogger() {
	DebugConsoleEncode()
	InitZapLog("", nil, "")
}
