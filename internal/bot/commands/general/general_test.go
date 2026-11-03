package general

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/bot"
)

func TestRegisterIncludesDynamicHelpCommand(t *testing.T) {
	registry := bot.NewRegistry()
	if err := New().Register(registry); err != nil {
		t.Fatalf("register general commands: %v", err)
	}

	command, ok := registry.Get("help")
	if !ok {
		t.Fatal("/help was not registered")
	}
	if command.Handler == nil {
		t.Fatal("/help has no handler")
	}
}

func TestHelpCommandLinesListsSubcommandsFromRegistryOptions(t *testing.T) {
	command := &bot.Command{
		Name:        "ideas",
		Description: "Panel de ideas",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "panel",
				Description: "Abre el tablero de ideas",
			},
		},
	}

	lines := helpCommandLines(command)
	if len(lines) != 1 || !strings.Contains(lines[0].name, "/ideas panel") {
		t.Fatalf("expected /ideas panel in help output, got %v", lines)
	}
	if lines[0].description != "Abre el tablero de ideas" {
		t.Fatalf("help entry should retain the command explanation, got %q", lines[0].description)
	}
}

func TestHelpSectionIncludesReturnButton(t *testing.T) {
	registry := bot.NewRegistry()
	if err := New().Register(registry); err != nil {
		t.Fatalf("register general commands: %v", err)
	}

	components := helpComponents(registry, "general")
	if len(components) != 2 {
		t.Fatalf("section help should include the selector and return button, got %d component rows", len(components))
	}
	row, ok := components[1].(discordgo.ActionsRow)
	if !ok || len(row.Components) != 1 {
		t.Fatalf("expected a return-button row, got %#v", components[1])
	}
	button, ok := row.Components[0].(discordgo.Button)
	if !ok || button.CustomID != "help:home" {
		t.Fatalf("expected the return-to-help button, got %#v", row.Components[0])
	}
}

func TestHelpCommandLinesListsSubcommandGroups(t *testing.T) {
	command := &bot.Command{
		Name: "inframundo",
		AppCmd: &discordgo.ApplicationCommand{
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type: discordgo.ApplicationCommandOptionSubCommandGroup,
					Name: "menu",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Name:        "casos",
							Description: "Revisa casos activos",
						},
					},
				},
			},
		},
	}

	lines := helpCommandLines(command)
	if len(lines) != 1 || !strings.Contains(lines[0].name, "/inframundo menu casos") {
		t.Fatalf("expected grouped command path in help output, got %v", lines)
	}
}
