package luxpanel

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/bot"
	"krypton.dev/internal/storage"
)

func TestCogRegistersLuxPanel(t *testing.T) {
	registry := bot.NewRegistry()
	if err := New().Register(registry); err != nil {
		t.Fatalf("register Lux panel: %v", err)
	}

	command, ok := registry.Get("lux")
	if !ok {
		t.Fatal("/lux was not registered")
	}
	if !command.GuildOnly {
		t.Fatal("/lux should only be available in a server")
	}
}

func TestRenderPanelContainsPrivateNavigationAndDailyReward(t *testing.T) {
	panel := &storage.LuxPanelData{
		Balance:    1250,
		Experience: 125,
		Preferences: storage.LuxPreferences{
			Language:             "es",
			NotificationsEnabled: true,
		},
	}

	embed, components := renderPanel("member-1", pageProfile, panel, 0x123456)
	if got, want := embed.Title, "👤・Tu perfil del Lux"; got != want {
		t.Fatalf("profile title = %q, want %q", got, want)
	}
	if len(components) != 2 {
		t.Fatalf("profile component rows = %d, want 2", len(components))
	}

	row, ok := components[1].(discordgo.ActionsRow)
	if !ok || len(row.Components) != 1 {
		t.Fatal("expected a single daily reward button")
	}
	button, ok := row.Components[0].(discordgo.Button)
	if !ok {
		t.Fatal("daily reward component is not a button")
	}
	if got, want := button.CustomID, "lux:member-1:profile:daily"; got != want {
		t.Fatalf("daily reward custom ID = %q, want %q", got, want)
	}
}

func TestSettingsPanelIncludesResponseVisibilityToggle(t *testing.T) {
	panel := &storage.LuxPanelData{
		Preferences: storage.LuxPreferences{
			Language:        "es",
			ResponsesPublic: true,
		},
	}
	embed, components := renderPanel("member-1", pageSettings, panel, 0x123456)
	if !strings.Contains(embed.Description, "Respuestas del bot: **pública**") {
		t.Fatalf("settings description should show current visibility, got %q", embed.Description)
	}
	if len(components) != 5 {
		t.Fatalf("settings panel has %d component rows, want 5", len(components))
	}
	row, ok := components[4].(discordgo.ActionsRow)
	if !ok || len(row.Components) != 1 {
		t.Fatal("expected a response visibility toggle button")
	}
	button, ok := row.Components[0].(discordgo.Button)
	if !ok || button.CustomID != "lux:member-1:settings:responses_public" {
		t.Fatalf("visibility toggle = %#v", row.Components[0])
	}
	if button.Label != "👁️ Visibilidad: pública" {
		t.Fatalf("visibility label = %q", button.Label)
	}
}

func TestRankForLevel(t *testing.T) {
	tests := []struct {
		level   int64
		english bool
		want    string
	}{
		{level: 1, want: "✨ Alma nueva"},
		{level: 5, want: "🍸 Habitual"},
		{level: 10, want: "💎 VIP"},
		{level: 20, want: "👑 Leyenda del Lux"},
		{level: 1, english: true, want: "✨ New soul"},
	}

	for _, test := range tests {
		if got := rankForLevel(test.level, test.english); got != test.want {
			t.Errorf("rankForLevel(%d, %t) = %q, want %q", test.level, test.english, got, test.want)
		}
	}
}
