package agenda

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/bot"
)

func TestCogRegistersEventWorkflow(t *testing.T) {
	registry := bot.NewRegistry()
	if err := New().Register(registry); err != nil {
		t.Fatalf("register agenda cog: %v", err)
	}
	command, ok := registry.Get("agenda")
	if !ok {
		t.Fatal("agenda command was not registered")
	}
	if got, want := len(command.Options), 1; got != want {
		t.Fatalf("agenda actions = %d, want %d", got, want)
	}
	if command.Options[0].Name != "panel" {
		t.Fatalf("agenda action = %q, want panel", command.Options[0].Name)
	}
}

func TestParseStartTimeRequiresValidFutureDate(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	valid := "2026-10-30T12:30:00Z"
	if got, err := parseStartTime(valid, now); err != nil || !got.Equal(now.Add(30*time.Minute)) {
		t.Fatalf("parse valid start time = %v, %v", got, err)
	}
	for _, invalid := range []string{
		"not-a-date",
		"2026-10-30T12:05:00Z",
		"2027-06-01T12:00:00Z",
	} {
		if _, err := parseStartTime(invalid, now); err == nil {
			t.Errorf("parseStartTime(%q) unexpectedly succeeded", invalid)
		}
	}
}

func TestHasPermissionIncludesAdministrator(t *testing.T) {
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{Permissions: discordgo.PermissionAdministrator},
		},
	}
	if !hasPermission(interaction, discordgo.PermissionManageEvents) {
		t.Fatal("administrator should satisfy event-management permission")
	}
	interaction.Member.Permissions = 0
	if hasPermission(interaction, discordgo.PermissionManageEvents) {
		t.Fatal("member without permissions should not manage events")
	}
}
