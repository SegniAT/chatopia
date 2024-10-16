package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"

	"github.com/fatih/color"
)

type PrettyHandlerOptions struct {
	SlogOpts slog.HandlerOptions
}

type PrettyHandler struct {
	slog.Handler
	l *log.Logger
}

func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	// color the level
	level := r.Level.String() + ":"

	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	}

	// format the attributes
	fields := make(map[string]interface{}, r.NumAttrs())

	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.Any()
		return true
	})

	b, err := json.MarshalIndent(fields, "", " ")
	if err != nil {
		return err
	}

	// otherwise we get {}, empty attribute list
	if len(fields) == 0 {
		b = nil
	}

	// format time & msg
	y, m, d := r.Time.Date()
	dateStr := fmt.Sprintf("%d/%d/%d", y, m, d)
	timeStr := r.Time.Format("15:05:05.000")
	msg := color.CyanString(r.Message)

	// print
	h.l.Println(fmt.Sprintf("[%s %s]", dateStr, timeStr), level, msg, color.WhiteString(string(b)))

	return nil
}

func NewPrettyHandler(out io.Writer, opts PrettyHandlerOptions) *PrettyHandler {
	h := &PrettyHandler{
		Handler: slog.NewJSONHandler(out, &opts.SlogOpts),
		l:       log.New(out, "", 0),
	}

	return h
}
