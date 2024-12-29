// loginfra/logger.go
package loginfra

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
)

type Logger struct {
	writer io.Writer
	prefix string
	level  LogLevel
}

type contextKey string

const loggerKey = contextKey("logger")

func NewLogger(writer io.Writer, prefix string, level LogLevel) *Logger {
	return &Logger{
		writer: writer,
		prefix: prefix,
		level:  level,
	}
}

func (l *Logger) log(level LogLevel, message string) {
	if level >= l.level {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(l.writer, "[%s] %s %s: %s\n",
			timestamp, l.prefix, getLevelString(level), message)
	}
}

func getLevelString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func (l *Logger) Debug(message string) {
	l.log(DEBUG, message)
}

func (l *Logger) Info(message string) {
	l.log(INFO, message)
}

func (l *Logger) Warning(message string) {
	l.log(WARNING, message)
}

func (l *Logger) Error(message string) {
	l.log(ERROR, message)
}

// WithLogger adds logger to context
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// GetLogger retrieves logger from context
func GetLogger(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		return logger
	}
	// Return a default logger if none found in context
	return NewLogger(os.Stdout, "DEFAULT", INFO)
}
