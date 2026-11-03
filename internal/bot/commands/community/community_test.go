package community

import (
	"testing"

	"krypton.dev/internal/bot"
)

func TestCogRegistersIdeaWorkflow(t *testing.T) {
	registry := bot.NewRegistry()
	if err := New().Register(registry); err != nil {
		t.Fatalf("register community cog: %v", err)
	}
	command, ok := registry.Get("ideas")
	if !ok {
		t.Fatal("ideas command was not registered")
	}
	if got, want := len(command.Options), 2; got != want {
		t.Fatalf("idea actions = %d, want %d", got, want)
	}
	if command.Options[0].Name != "panel" || command.Options[1].Name != "enviar" {
		t.Fatalf("idea actions = %q, %q, want panel and enviar", command.Options[0].Name, command.Options[1].Name)
	}
}

func TestSafeTextPreventsMentions(t *testing.T) {
	got := safeText("@everyone, ven al canal")
	if got == "@everyone, ven al canal" {
		t.Fatal("safeText left a broadcast mention active")
	}
}

func TestShortenPreservesShortTextAndBoundsLongText(t *testing.T) {
	if got := shorten("corto", 10); got != "corto" {
		t.Fatalf("shorten short text = %q", got)
	}
	if got := shorten("abcdefghijklmnop", 8); len([]rune(got)) != 8 {
		t.Fatalf("shortened text has %d runes, want 8", len([]rune(got)))
	}
}
