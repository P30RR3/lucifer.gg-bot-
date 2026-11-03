package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	reset   = "\x1b[0m"
	muted   = "\x1b[90m"
	red     = "\x1b[91m"
	yellow  = "\x1b[93m"
	green   = "\x1b[92m"
	cyan    = "\x1b[96m"
	magenta = "\x1b[95m"
)

type consoleHandler struct {
	writer io.Writer
	level  slog.Leveler
	attrs  []slog.Attr
	groups []string
	color  bool
}

func (h *consoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *consoleHandler) Handle(_ context.Context, record slog.Record) error {
	var line strings.Builder
	timestamp := record.Time
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	line.WriteString(h.paint(muted, timestamp.Format("15:04:05")))
	line.WriteString("  ")

	levelColor, levelName := levelStyle(record.Level)
	line.WriteString(h.paint(levelColor, fmt.Sprintf("%-5s", levelName)))
	line.WriteString("  ")
	line.WriteString(h.paint(cyan, record.Message))

	attrs := append([]slog.Attr(nil), h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	for _, attr := range attrs {
		appendAttr(&line, attr, h.groups, h.color)
	}
	line.WriteByte('\n')
	_, err := io.WriteString(h.writer, consoleSafe(line.String()))
	return err
}

func consoleSafe(value string) string {
	var safe strings.Builder
	for _, character := range value {
		switch {
		case character == '\u200d' || character == '\ufe0f' || character == '\u20e3' || character == '\u30fb':
			continue
		case character >= 0x1f000 && character <= 0x1faff:
			continue
		case character >= 0x2600 && character <= 0x27bf:
			continue
		case character >= 0x2500 && character <= 0x257f:
			continue
		default:
			safe.WriteRune(character)
		}
	}
	return safe.String()
}

func (h *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &clone
}

func (h *consoleHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	clone.groups = append(append([]string(nil), h.groups...), name)
	return &clone
}

func (h *consoleHandler) paint(color, value string) string {
	if !h.color || color == "" {
		return value
	}
	return color + value + reset
}

func levelStyle(level slog.Level) (string, string) {
	switch {
	case level >= slog.LevelError:
		return red, "ERROR"
	case level >= slog.LevelWarn:
		return yellow, "WARN"
	case level <= slog.LevelDebug:
		return magenta, "DEBUG"
	default:
		return green, "INFO"
	}
}

func appendAttr(line *strings.Builder, attr slog.Attr, groups []string, color bool) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}
	if attr.Value.Kind() == slog.KindGroup {
		nextGroups := append(append([]string(nil), groups...), attr.Key)
		for _, child := range attr.Value.Group() {
			appendAttr(line, child, nextGroups, color)
		}
		return
	}
	keyParts := append(append([]string(nil), groups...), attr.Key)
	key := strings.Join(keyParts, ".")
	value := fmt.Sprint(attr.Value.Any())
	if attr.Value.Kind() == slog.KindString {
		value = attr.Value.String()
	}
	if strings.ContainsAny(value, " \t\r\n") {
		value = strconv.Quote(value)
	}
	line.WriteString("  ")
	if color {
		line.WriteString(muted)
	}
	line.WriteString(key)
	if color {
		line.WriteString(reset)
	}
	line.WriteByte('=')
	line.WriteString(value)
}

func Init(level slog.Level, format string) {
	options := &slog.HandlerOptions{Level: level}
	if format == "json" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, options)))
		return
	}
	slog.SetDefault(slog.New(&consoleHandler{
		writer: os.Stdout,
		level:  level,
		color:  terminalColorsEnabled(),
	}))
}

func terminalColorsEnabled() bool {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled || os.Getenv("TERM") == "dumb" {
		return false
	}
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
