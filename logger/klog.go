package logger

import (
	"io"
	"os"

	"github.com/cloudwego/kitex/pkg/klog"
	kitexlogrus "github.com/kitex-contrib/obs-opentelemetry/logging/logrus"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitKlogWithLogrus(cfg Config) {
	// klog
	klogrus := kitexlogrus.NewLogger()

	// format
	if cfg.Format == Text {
		klogrus.Logger().SetFormatter(&logrus.TextFormatter{
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
		klogrus.Logger().AddHook(hook)
	}
	klog.SetLogger(klogrus)
	klog.SetLevel(klog.Level(cfg.Level))
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
		klog.SetOutput(os.Stdout)
	case AsyncFile:
		klog.SetOutput(asyncWriter)
		onShutDown(func() {
			asyncWriter.Sync()
		})
	case StdOut | AsyncFile:
		klog.SetOutput(io.MultiWriter(asyncWriter, os.Stdout))
		onShutDown(func() {
			asyncWriter.Sync()
		})
	}
}
