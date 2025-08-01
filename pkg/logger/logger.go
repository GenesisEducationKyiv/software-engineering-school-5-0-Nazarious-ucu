package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	maxSize = 10
	maxBack = 5
	maxAge  = 30
)

func NewLogger(filePath, serviceName string) (zerolog.Logger, error) {
	// Initialize the logger with default settings
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}

	var writers []io.Writer
	writers = append(writers, consoleWriter)

	// Initialize file writer
	fileRotator := &lumberjack.Logger{
		Filename:   filePath, // log file location
		MaxSize:    maxSize,  // megabytes before rotation
		MaxBackups: maxBack,  // number of old files to retain
		MaxAge:     maxAge,   // days to retain rotated files
		Compress:   true,     // gzip old log files
	}

	writers = append(writers, fileRotator)

	// Create a multi-writer to write logs to both console and file
	multiWriter := zerolog.MultiLevelWriter(writers...)
	logger := zerolog.New(multiWriter).With().
		Timestamp().
		Caller().
		Str("service", serviceName).
		Logger().
		Level(zerolog.DebugLevel)

	logger.Info().
		Str("logsFilePath", filePath).
		Str("serviceName", serviceName).
		Msg("Logger initialized with file rotation")

	return logger, nil
}
