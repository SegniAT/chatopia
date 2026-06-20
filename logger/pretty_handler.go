package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"path/filepath"
	"runtime"

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
	fields := make(map[string]any, r.NumAttrs())

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

	sourceStr := ""
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if f.File != "" {
			shortFile := filepath.Base(f.File)
			sourceStr = fmt.Sprintf(" %s:%d ", shortFile, f.Line)
		}
	}

	// print
	h.l.Println(
		fmt.Sprintf("[%s %s]", dateStr, timeStr),
		level,
		msg,
		color.WhiteString(string(b)),
		color.New(color.FgYellow, color.BgBlack, color.Italic).Sprintf(sourceStr),
	)

	return nil
}

func NewPrettyHandler(out io.Writer, opts PrettyHandlerOptions) *PrettyHandler {
	h := &PrettyHandler{
		Handler: slog.NewJSONHandler(out, &opts.SlogOpts),
		l:       log.New(out, "", 0),
	}

	return h
}
