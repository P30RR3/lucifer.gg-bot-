package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestConsoleHandlerKeepsLuxLogsReadable(t *testing.T) {
	var output bytes.Buffer
	handler := &consoleHandler{writer: &output, level: slog.LevelDebug}
	record := slog.NewRecord(time.Date(2026, 10, 31, 22, 0, 0, 0, time.UTC), slog.LevelInfo, "Lucifer llegó al Lux", 0)
	record.AddAttrs(slog.String("guild", "El club central"), slog.Int("members", 12))

	if err := handler.Handle(context.Background(), record); err != nil {
		t.Fatalf("write log: %v", err)
	}
	got := output.String()
	for _, expected := range []string{"22:00:00", "INFO", "Lucifer llegó al Lux", `guild="El club central"`, "members=12"} {
		if !strings.Contains(got, expected) {
			t.Errorf("log output %q does not contain %q", got, expected)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("non-terminal logs should not contain ANSI escapes: %q", got)
	}
}

func TestConsoleHandlerColorsOnlyWhenEnabled(t *testing.T) {
	var output bytes.Buffer
	handler := &consoleHandler{writer: &output, level: slog.LevelDebug, color: true}
	record := slog.NewRecord(time.Now(), slog.LevelError, "falló la conexión", 0)
	if err := handler.Handle(context.Background(), record); err != nil {
		t.Fatalf("write log: %v", err)
	}
	if !strings.Contains(output.String(), red) || !strings.Contains(output.String(), reset) {
		t.Fatalf("error output should include its terminal color: %q", output.String())
	}
}

func TestConsoleSafeRemovesDecorationButKeepsSpanishText(t *testing.T) {
	got := consoleSafe("🔥・Lucifer abrió el Lux ╭────╮")
	if strings.ContainsAny(got, "🔥・╭╮─") {
		t.Fatalf("console decoration should be removed: %q", got)
	}
	if !strings.Contains(got, "Lucifer abrió el Lux") {
		t.Fatalf("Spanish text should remain readable: %q", got)
	}
}

func TestConsoleHandlerPreservesGroupsAndWithAttrs(t *testing.T) {
	var output bytes.Buffer
	handler := (&consoleHandler{writer: &output, level: slog.LevelDebug}).
		WithGroup("discord").
		WithAttrs([]slog.Attr{slog.String("component", "router")})
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "panel abierto", 0)
	record.AddAttrs(slog.String("guild", "lux"))
	if err := handler.Handle(context.Background(), record); err != nil {
		t.Fatalf("write grouped log: %v", err)
	}
	got := output.String()
	for _, expected := range []string{"component=router", "discord.guild=lux"} {
		if !strings.Contains(got, expected) {
			t.Errorf("grouped log output %q does not contain %q", got, expected)
		}
	}
}
