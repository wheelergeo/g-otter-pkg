package logger

import (
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	hertzlogrus "github.com/hertz-contrib/obs-opentelemetry/logging/logrus"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type shutdownFunc func()

type LogMode uint8

type FormatType uint8

type Level int

type FileConfig struct {
	FileName      string
	MaxSize       int
	MaxBackups    int
	MaxAge        int
	FlushInterval time.Duration
}

type NetConfig struct {
	Endpoints []string
	CloudId   string
	ApiKey    string
	Host      string
	Level     logrus.Level
	Type      TransType
	Index     string
}

type Config struct {
	Mode    LogMode
	Format  FormatType
	Level   Level
	FileCfg FileConfig
	NetCfg  NetConfig
}

const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelNotice
	LevelWarn
	LevelError
	LevelFatal
)

const (
	StdOut    LogMode = 0x01
	AsyncFile LogMode = 0x02
	Net       LogMode = 0x04
)

const (
	Json FormatType = 0
	Text FormatType = 1
)

var onShutdownFunc []shutdownFunc
var signalChan chan os.Signal

func init() {
	signalChan = make(chan os.Signal, 1)
	go func() {
		//阻塞程序运行，直到收到终止的信号
		<-signalChan
		for _, v := range onShutdownFunc {
			v()
		}
	}()
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
}

func InitHlogWithLogrus(cfg Config) {
	// hlog
	hlogrus := hertzlogrus.NewLogger()

	// format
	if cfg.Format == Text {
		hlogrus.Logger().SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
	// netlog
	if (cfg.Mode & Net) == Net {
		hook, err := NewElasticHook(
			ElasticConfig{
				endpoints: cfg.NetCfg.Endpoints,
				cloudId:   cfg.NetCfg.CloudId,
				apiKey:    cfg.NetCfg.ApiKey,
			},
			cfg.NetCfg.Host,
			cfg.NetCfg.Type,
			cfg.NetCfg.Level,
			cfg.NetCfg.Index,
		)
		if err != nil {
			panic(err)
		}
		hlogrus.Logger().AddHook(hook)
	}
	hlog.SetLogger(hlogrus)
	hlog.SetLevel(hlog.Level(cfg.Level))
	// filelog
	asyncWriter := &zapcore.BufferedWriteSyncer{
		WS: zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.FileCfg.FileName,
			MaxSize:    cfg.FileCfg.MaxSize,
			MaxBackups: cfg.FileCfg.MaxBackups,
			MaxAge:     cfg.FileCfg.MaxAge,
		}),
		FlushInterval: cfg.FileCfg.FlushInterval,
	}

	switch cfg.Mode & (StdOut | AsyncFile) {
	case StdOut:
		hlog.SetOutput(os.Stdout)
	case AsyncFile:
		hlog.SetOutput(asyncWriter)
		onShutDown(func() {
			asyncWriter.Sync()
		})
	case StdOut | AsyncFile:
		hlog.SetOutput(io.MultiWriter(asyncWriter, os.Stdout))
		onShutDown(func() {
			asyncWriter.Sync()
		})
	}
}

func onShutDown(f shutdownFunc) {
	onShutdownFunc = append(onShutdownFunc, f)
}
