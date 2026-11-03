package audit

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/bot"
)

func TestRegisterAuditCommands(t *testing.T) {
	registry := bot.NewRegistry()
	if err := New().Register(registry); err != nil {
		t.Fatalf("register audit commands: %v", err)
	}

	for _, name := range []string{"logs", "soporte", "anuncio", "staff"} {
		if _, ok := registry.Get(name); !ok {
			t.Errorf("command %q was not registered", name)
		}
	}

	logs, ok := registry.Get("logs")
	if !ok {
		t.Fatal("logs command missing")
	}
	if len(logs.Options) != 4 {
		t.Fatalf("logs has %d subcommands, want 4", len(logs.Options))
	}
}

func TestAnnouncementPreviewHasPublicReviewActions(t *testing.T) {
	content, embeds, components := announcementPreview("123456789012345678", "La casa abre esta noche.", false)
	if !strings.Contains(content, "La casa abre esta noche.") {
		t.Fatalf("announcement preview should display the AI draft as text, got %q", content)
	}
	if len(embeds) != 0 {
		t.Fatalf("text announcement preview should not contain embeds, got %d", len(embeds))
	}
	if len(components) != 1 {
		t.Fatalf("announcement preview should have one action row, got %d", len(components))
	}
	row, ok := components[0].(discordgo.ActionsRow)
	if !ok || len(row.Components) != 4 {
		t.Fatalf("announcement preview should expose delete, edit, publish, and embed buttons, got %#v", components[0])
	}
}

func TestAnnouncementPreviewCanToggleEmbedFormat(t *testing.T) {
	content, embeds, components := announcementPreview("123456789012345678", "La casa abre esta noche.", true)
	if content != "" || len(embeds) != 1 || embeds[0].Description != "La casa abre esta noche." {
		t.Fatalf("embed preview = content %q, embeds %#v", content, embeds)
	}
	row := components[0].(discordgo.ActionsRow)
	button := row.Components[3].(discordgo.Button)
	if button.Label != "Usar texto" {
		t.Fatalf("toggle label = %q, want %q", button.Label, "Usar texto")
	}
}

func TestAnnouncementColorsAreNamedAndDark(t *testing.T) {
	expected := map[string]int{
		"rojo oscuro":  0x4A0000,
		"borgoña":      0x3B0008,
		"granate":      0x510A14,
		"verde oscuro": 0x15301F,
		"negro":        0x171014,
	}
	for name, want := range expected {
		color, ok := announcementColor(name)
		if !ok || color != want {
			t.Errorf("announcement color %q = %#06x, supported=%t; want %#06x", name, color, ok, want)
		}
	}
	if _, ok := announcementColor("#ff0000"); ok {
		t.Fatal("announcement color input should accept names, not HEX codes")
	}
}

func TestAnnouncementSanitizationDisablesMentions(t *testing.T) {
	got := sanitizeAnnouncement("@everyone revisa <@123> y @here.")
	want := "@\u200beveryone revisa <\u200b@123> y @\u200bhere."
	if got != want {
		t.Fatalf("sanitizeAnnouncement() = %q, want %q", got, want)
	}
}

func TestEligibleStaffRolesExcludesEveryoneAndNonStaff(t *testing.T) {
	roles := []*discordgo.Role{
		{ID: "guild", Permissions: discordgo.PermissionManageMessages},
		{ID: "member", Permissions: discordgo.PermissionViewChannel},
		{ID: "moderator", Permissions: discordgo.PermissionModerateMembers},
		{ID: "integrated", Permissions: discordgo.PermissionBanMembers, Managed: true},
	}
	got := eligibleStaffRoles(roles, "guild")
	if len(got) != 1 || got[0].ID != "moderator" {
		t.Fatalf("eligibleStaffRoles() = %#v, want only moderator role", got)
	}
}

func TestIsPrivateToEveryoneRequiresViewChannelDeny(t *testing.T) {
	private := &discordgo.Channel{
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{ID: "guild", Type: discordgo.PermissionOverwriteTypeRole, Deny: discordgo.PermissionViewChannel},
		},
	}
	if !isPrivateToEveryone(private, "guild") {
		t.Fatal("expected a channel denying ViewChannel to @everyone to be private")
	}
	public := &discordgo.Channel{
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{ID: "guild", Type: discordgo.PermissionOverwriteTypeRole, Allow: discordgo.PermissionViewChannel},
		},
	}
	if isPrivateToEveryone(public, "guild") {
		t.Fatal("expected a channel allowing ViewChannel to @everyone to be rejected")
	}
}
