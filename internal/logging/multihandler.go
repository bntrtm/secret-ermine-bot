package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// multiHandler manages a top-level handler for the bot itself,
// as well as multiple handlers pertaining to each active
// Secret Santa event.
type multiHandler struct {
	botHandler       slog.Handler
	eventLogHandlers map[string]slog.Handler // map of server IDs to slog handlers for event-based logging
	eventLogFiles    []*os.File              // a slice of files opened by event log handlers
}

// NewMultiHandler returns a multiHandler for logging, purpose-built
// for the Secret Ermine bot.
//
// Its top-level bot handler is provided the given slog.Handler values.
// Its event handler map will be empty, but initialized.
func NewMultiHandler(w io.Writer, level slog.Level) *multiHandler {
	return &multiHandler{
		botHandler: slog.NewJSONHandler(w,
			&slog.HandlerOptions{Level: level}),
		eventLogHandlers: map[string]slog.Handler{},
	}
}

func (m *multiHandler) AddEventLogHandler(sID string, level slog.Level) error {
	file, err := New(event, sID)
	if err != nil {
		return err
	}

	m.eventLogFiles = append(m.eventLogFiles, file)

	m.eventLogHandlers[getCurrentLogName(event, sID)] = slog.NewJSONHandler(file, &slog.HandlerOptions{Level: level})
	return nil
}

// Enabled wraps a call to the bot handler's "Enabled" method.
func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return m.botHandler.Enabled(ctx, level)
}

// Handle checks the given context for a LogKey whose value exists
// as a key within the multiHandler's map of server IDs to slog
// handlers. If the context key or event log handler is not found,
// Handle calls the botHandler's "Handle" method.
func (m *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	if sID, ok := ctx.Value(SIDKey).(string); ok {
		handler, ok := m.eventLogHandlers[getCurrentLogName(event, sID)]
		if ok {
			return handler.Handle(ctx, record)
		}
		err := m.AddEventLogHandler(sID, slog.LevelInfo)
		if err == nil {
			return m.eventLogHandlers[getCurrentLogName(event, sID)].Handle(ctx, record)
		}
	}

	return m.botHandler.Handle(ctx, record)
}

// WithAttrs wraps a call to the bot handler's "WithAttrs" method.
func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return m.botHandler.WithAttrs(attrs)
}

// WithGroup wraps a call to the bot handler's "WithGroup" method.
func (m *multiHandler) WithGroup(name string) slog.Handler {
	return m.botHandler.WithGroup(name)
}
