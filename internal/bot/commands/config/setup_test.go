package config

import (
	"bytes"
	"image/png"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/bot"
)

func TestMissingSetupPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions int64
		wantMissing int
	}{
		{name: "all required permissions", permissions: setupPermission},
		{name: "no permissions", permissions: 0, wantMissing: 3},
		{name: "send and view only", permissions: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages, wantMissing: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := missingSetupPermissions(test.permissions)
			if len(got) != test.wantMissing {
				t.Fatalf("missingSetupPermissions() returned %d missing permissions, want %d: %v", len(got), test.wantMissing, got)
			}
		})
	}
}

func TestConfigurationAssistantCommandIsRegisteredForGuildManagers(t *testing.T) {
	registry := bot.NewRegistry()
	if err := New().Register(registry); err != nil {
		t.Fatalf("register config commands: %v", err)
	}

	command, ok := registry.Get("configuracion")
	if !ok {
		t.Fatal("expected /configuracion to be registered")
	}
	if command.DefaultPerm != discordgo.PermissionManageGuild {
		t.Errorf("/configuracion default permission = %d, want ManageGuild (%d)", command.DefaultPerm, discordgo.PermissionManageGuild)
	}
	if !command.GuildOnly {
		t.Error("/configuracion should only be available in guilds")
	}
}

func TestRenderWelcomeMessageReplacesPlaceholders(t *testing.T) {
	got := RenderWelcomeMessage("Hola {usuario}, bienvenido a {servidor}.", "<@123>", "El Lux")
	want := "Hola <@123>, bienvenido a El Lux."
	if got != want {
		t.Fatalf("RenderWelcomeMessage() = %q, want %q", got, want)
	}
}

func TestRenderWelcomeMessageUsesDefaultWhenTemplateIsEmpty(t *testing.T) {
	got := RenderWelcomeMessage("  ", "<@123>", "El Lux")
	if got == "" || !strings.Contains(got, "<@123>") || !strings.Contains(got, "El Lux") {
		t.Fatalf("default welcome message did not render placeholders: %q", got)
	}
}

func TestWelcomeTemplateSummaryIsBounded(t *testing.T) {
	got := welcomeTemplateSummary(strings.Repeat("x", maxWelcomeTemplateLength))
	if len([]rune(got)) > 180 {
		t.Fatalf("welcome template summary has %d runes, want at most 180", len([]rune(got)))
	}
}

func TestWelcomeImageURLMustUseHTTPS(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "https image", value: "https://cdn.example.com/welcome.png"},
		{name: "http image", value: "http://cdn.example.com/welcome.png", wantErr: true},
		{name: "missing host", value: "https:///welcome.png", wantErr: true},
		{name: "embedded credentials", value: "https://user:pass@cdn.example.com/welcome.png", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateWelcomeImageURL(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateWelcomeImageURL(%q) error = %v, wantErr %v", test.value, err, test.wantErr)
			}
		})
	}
}

func TestWelcomeImageFilesUseInfernoBackgroundByDefault(t *testing.T) {
	files, err := WelcomeImageFiles("")
	if err != nil {
		t.Fatalf("generate default welcome background: %v", err)
	}
	if len(files) != 1 || files[0].Name != "welcome-inferno.png" {
		t.Fatalf("default welcome files = %#v, want one inferno PNG attachment", files)
	}
	config, err := png.DecodeConfig(files[0].Reader)
	if err != nil {
		t.Fatalf("decode default welcome background: %v", err)
	}
	if config.Width != infernoWidth || config.Height != infernoHeight {
		t.Fatalf("background size = %dx%d, want %dx%d", config.Width, config.Height, infernoWidth, infernoHeight)
	}
}

func TestWelcomeImageFilesOmitAttachmentForCustomURL(t *testing.T) {
	files, err := WelcomeImageFiles("https://cdn.example.com/welcome.png")
	if err != nil {
		t.Fatalf("resolve custom welcome image: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("custom welcome image unexpectedly generated fallback files: %d", len(files))
	}
}

func TestResolveWelcomeImageURLPrefersServerSetting(t *testing.T) {
	got := ResolveWelcomeImageURL("https://server.example/welcome.png", "https://global.example/default.png")
	if got != "https://server.example/welcome.png" {
		t.Fatalf("ResolveWelcomeImageURL() = %q, want server override", got)
	}
	if got := ResolveWelcomeImageURL("", "https://global.example/default.png"); got != "https://global.example/default.png" {
		t.Fatalf("ResolveWelcomeImageURL() without server value = %q, want global default", got)
	}
	if got := ResolveWelcomeImageURL("", ""); got != "" {
		t.Fatalf("ResolveWelcomeImageURL() without images = %q, want inferno fallback selection", got)
	}
}

func TestInfernoBackgroundBytesAreStable(t *testing.T) {
	first, err := infernoBackgroundPNG()
	if err != nil {
		t.Fatal(err)
	}
	second, err := infernoBackgroundPNG()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("cached inferno background changed between calls")
	}
}
