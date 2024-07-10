package utils

import (
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
	"time"
)

func getLogLevel(level string) log.Level {
	switch level {
	case "info":
		return log.InfoLevel
	case "debug":
		return log.DebugLevel
	case "error":
		return log.ErrorLevel
	default:
		return log.InfoLevel
	}
}

func InitLog(level string, path string) {
	log.SetLevel(getLogLevel(level))
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: "2006-01-02 15:04:05.000"})

	// file system logger setting
	if path != "" {
		localFilesystemLogger(path)
	}
}

func logWriter(logPath string) *rotatelogs.RotateLogs {
	logFullPath := logPath
	logwriter, err := rotatelogs.New(
		logFullPath+".%Y%m%d",
		rotatelogs.WithLinkName(logFullPath),
		rotatelogs.WithRotationSize(100*1024*1024), // 100MB
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		panic(err)
	}
	return logwriter
}

func localFilesystemLogger(logPath string) {
	lfHook := lfshook.NewHook(logWriter(logPath), &log.TextFormatter{FullTimestamp: true, TimestampFormat: "2006-01-02 15:04:05.000"})
	log.AddHook(lfHook)
}
