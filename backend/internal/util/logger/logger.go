package logger

import (
	"log/slog"
	"os"
	"sync"
	"time"
)

var loggerInstance *slog.Logger

var initLogger = sync.OnceFunc(func() {
	stdoutHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(time.Now().Format("2006/01/02 15:04:05"))
			}
			if a.Key == slog.LevelKey {
				return slog.Attr{}
			}
			return a
		},
	})

	loggerInstance = slog.New(stdoutHandler)
	loggerInstance.Info("Text structured logger initialized")
})

func GetLogger() *slog.Logger {
	initLogger()
	return loggerInstance
}
