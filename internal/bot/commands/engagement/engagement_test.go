package engagement

import (
	"testing"

	"krypton.dev/internal/bot"
)

func TestCogRegistersEngagementCommands(t *testing.T) {
	registry := bot.NewRegistry()
	if err := New().Register(registry); err != nil {
		t.Fatalf("register engagement commands: %v", err)
	}
	for _, name := range []string{"terapia", "celestial", "celestial-configurar"} {
		if _, ok := registry.Get(name); !ok {
			t.Errorf("command %q was not registered", name)
		}
	}
}

func TestPanelPagesAndResponseModes(t *testing.T) {
	for _, page := range []string{therapyHistory, therapyPrivacy} {
		if !validTherapyPage(page) {
			t.Errorf("therapy page %q should be valid", page)
		}
	}
	for _, page := range []string{celestialMissions, celestialLeaders, celestialBlessings} {
		if !validCelestialPage(page) {
			t.Errorf("celestial page %q should be valid", page)
		}
	}
	if validCelestialPage("admin") || validLeaderboardCategory("sql") {
		t.Fatal("unknown panel pages and leaderboard filters must be rejected")
	}
	if responseModeText("private") == responseModeText("public") ||
		responseModeText("public") == responseModeText("dm") {
		t.Fatal("privacy response modes must have distinct labels")
	}
}
