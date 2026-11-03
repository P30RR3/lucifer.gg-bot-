package sanctions

import (
	"testing"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/bot"
)

func TestCogRegistersModerationTools(t *testing.T) {
	registry := bot.NewRegistry()
	if err := New().Register(registry); err != nil {
		t.Fatalf("register moderation tools: %v", err)
	}
	for _, name := range []string{"panel-staff", "inframundo", "infierno", "puertas-infernales"} {
		if _, ok := registry.Get(name); !ok {
			t.Errorf("command %q was not registered", name)
		}
	}
	inframundo, _ := registry.Get("inframundo")
	if len(inframundo.Options) != 1 || inframundo.Options[0].Name != "menu" {
		t.Fatal("inframundo should only expose the menu panel")
	}
}

func TestInvestigationButtonsMatchDiscordPermissions(t *testing.T) {
	rows := investigationButtons("guild", "target", discordgo.PermissionBanMembers)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want one row with the ban action", len(rows))
	}
	row, ok := rows[0].(discordgo.ActionsRow)
	if !ok || len(row.Components) != 1 {
		t.Fatalf("ban-only controls = %#v, want one button", rows[0])
	}
	button, ok := row.Components[0].(discordgo.Button)
	if !ok || button.CustomID != "investigate:guild:target:ban" {
		t.Fatalf("ban-only control = %#v, want ban button", row.Components[0])
	}
}

func TestHasStaffPermissionIncludesAdministrator(t *testing.T) {
	if !hasStaffPermission(discordgo.PermissionAdministrator, discordgo.PermissionBanMembers) {
		t.Fatal("administrator should satisfy action-specific permission checks")
	}
	if hasStaffPermission(discordgo.PermissionKickMembers, discordgo.PermissionBanMembers) {
		t.Fatal("kick permission must not grant ban permission")
	}
}
