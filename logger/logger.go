package logger

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func init() {
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatal("Failed to create logs directory:", err)
	}

	// Open log file
	logFile, err := os.OpenFile(filepath.Join("logs", "raft.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}

	// Set output to both file and stdout
	log.SetOutput(os.Stdout)
	log.AddHook(&fileHook{
		Writer: logFile,
		LogLevels: []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
			logrus.InfoLevel,
			logrus.DebugLevel,
		},
	})
}

// GetLogger returns the global logger instance
func GetLogger() *logrus.Logger {
	return log
}

// fileHook is a hook that writes logs to a file
type fileHook struct {
	Writer    *os.File
	LogLevels []logrus.Level
}

func (hook *fileHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	_, err = hook.Writer.Write([]byte(line))
	return err
}

func (hook *fileHook) Levels() []logrus.Level {
	return hook.LogLevels
}
