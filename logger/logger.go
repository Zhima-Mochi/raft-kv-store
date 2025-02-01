package logger

import (
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func init() {
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})
}

// GetLogger returns the global logger instance
func GetLogger() *logrus.Logger {
	return log
}
