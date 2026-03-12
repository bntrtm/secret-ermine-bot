// Package logging provides users of the Secret Ermine bot
// with organized structured logging that categorizes bot
// output on a per-event basis.
package logging

import (
	"context"
	"log/slog"
	"os"
)

// LogType defines a binary of log category managed by the bot:
// bot logs and event logs. Bot logs relate to top-level outputs
// directly, while event logs are generated on a per-event
// basis ("event" here meaning "Secret Santa event").
type LogType int

const (
	bot LogType = iota
	event
)

// logKey is a string used as a key in log-related contexts
// to determine whether a slog record pertains to an event or bot log file
type logKey string

const SIDKey logKey = "sID"

// Logger handles structured logging output related to the bot and the
// events that it manages.
type Logger struct {
	botLogFile *os.File // the file used for logging output not related to events
	slogger    *slog.Logger
	handler    *multiHandler // the handler associated with the slogger
}

// Init initializes a new multiHandler for logging.
// This should always be performed following instantiation
// of the [Logger] type.
func (l *Logger) Init() {
	l.botLogFile, _ = New(bot, "")

	l.handler = NewMultiHandler(l.botLogFile, slog.LevelInfo)
	l.slogger = slog.New(l.handler)
}

// Get returns the slog.Logger associated with the purpose-built [Logger] type.
func (l *Logger) Get() *slog.Logger {
	return l.slogger
}

// Quit closes all open files tracked by the logger
func (l *Logger) Quit() {
	_ = Close(l.botLogFile)
	if l.handler != l.slogger.Handler() {
		return
	}

	for _, file := range l.handler.eventLogFiles {
		_ = Close(file)
	}
}

// Log outputs to bot logs at the Info level
func (l *Logger) Log(msg string, args ...any) {
	l.slogger.Info(msg, args...)
}

// LogError outputs to bot logs at the Error level
func (l *Logger) LogError(msg string, args ...any) {
	l.slogger.Error(msg, args...)
}

// ELogDebug wraps "slog.DebugContext" functions for readability,
// to output to event logs at the Debug level
func (l *Logger) ELogDebug(sID, msg string, args ...any) {
	l.slogger.DebugContext(context.WithValue(context.Background(), SIDKey, sID), msg, args...)
}

// ELogInfo wraps "slog.InfoContext" functions for readability
// to output to event logs at the Info level
func (l *Logger) ELogInfo(sID, msg string, args ...any) {
	l.slogger.InfoContext(context.WithValue(context.Background(), SIDKey, sID), msg, args...)
}

// ELogWarn wraps "slog.WarnContext" functions for readability
// to output to event logs at the Warn level
func (l *Logger) ELogWarn(sID, msg string, args ...any) {
	l.slogger.WarnContext(context.WithValue(context.Background(), SIDKey, sID), msg, args...)
}

// ELogError wraps "slog.ErrorContext" functions for readability
// to output to event logs at the Error level
func (l *Logger) ELogError(sID, msg string, args ...any) {
	l.slogger.ErrorContext(context.WithValue(context.Background(), SIDKey, sID), msg, args...)
}
