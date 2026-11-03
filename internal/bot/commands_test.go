package bot_test

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/bot"
	"krypton.dev/internal/bot/commands/agenda"
	aiCog "krypton.dev/internal/bot/commands/ai"
	antiraidCog "krypton.dev/internal/bot/commands/antiraid"
	auditCog "krypton.dev/internal/bot/commands/audit"
	backupCog "krypton.dev/internal/bot/commands/backup"
	"krypton.dev/internal/bot/commands/community"
	configCog "krypton.dev/internal/bot/commands/config"
	"krypton.dev/internal/bot/commands/economy"
	"krypton.dev/internal/bot/commands/general"
	"krypton.dev/internal/bot/commands/moderation"
	"krypton.dev/internal/bot/commands/utility"
)

func TestCommandCogsRegister(t *testing.T) {
	registry := bot.NewRegistry()
	cogs := []interface {
		Register(*bot.Registry) error
	}{
		general.New(),
		utility.New(),
		moderation.New(),
		economy.New(),
		community.New(),
		agenda.New(),
		aiCog.New(),
		configCog.New(),
		auditCog.New(),
		backupCog.New(),
		antiraidCog.New(nil),
	}

	for index, cog := range cogs {
		if err := cog.Register(registry); err != nil {
			t.Fatalf("cog %d registration failed: %v", index, err)
		}
	}

	if registry.Count() == 0 {
		t.Fatal("expected commands to be registered")
	}
	stats, ok := registry.Get("botstats")
	if !ok || !stats.OwnerOnly {
		t.Fatal("botstats must be registered as owner-only")
	}
	backupPanel, ok := registry.Get("backup-panel")
	if !ok || backupPanel.DefaultPerm != discordgo.PermissionManageGuild {
		t.Fatal("backup-panel must be registered for server managers")
	}
	if _, ok := registry.Get("investigar"); ok {
		t.Fatal("investigar should be consolidated into the staff-only panel-staff command")
	}
}

func TestToApplicationCommandsLeavesUnrestrictedCommandsAvailable(t *testing.T) {
	registry := bot.NewRegistry()
	if err := registry.Register(&bot.Command{
		Name:        "publico",
		Description: "Comando disponible para todos",
		Handler:     func(context.Context, *discordgo.Session, *discordgo.InteractionCreate, *bot.Env) error { return nil },
	}); err != nil {
		t.Fatal(err)
	}

	if err := registry.Register(&bot.Command{
		Name:        "botstats",
		Description: "Comando filtrado por el router",
		OwnerOnly:   true,
		Handler:     func(context.Context, *discordgo.Session, *discordgo.InteractionCreate, *bot.Env) error { return nil },
	}); err != nil {
		t.Fatal(err)
	}
	commands := registry.ToApplicationCommands()
	for _, command := range commands {
		if command.DefaultMemberPermissions != nil {
			t.Errorf("%s unexpectedly has default member permissions %d", command.Name, *command.DefaultMemberPermissions)
		}
	}
}

func TestCommandCopyUsesElegantLuxSignatureWithinDiscordLimits(t *testing.T) {
	registry := bot.NewRegistry()
	if err := registry.Register(&bot.Command{
		Name:        "elegancia",
		Description: "Una descripción suficientemente larga para probar que el registro conserva el límite permitido por Discord sin perder el tono del Lux",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "motivo",
				Description: "Describe el motivo con todos los detalles que consideres necesarios para que el equipo pueda comprender bien tu solicitud",
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Opción elegante", Value: "elegante"},
				},
			},
		},
		Handler: func(context.Context, *discordgo.Session, *discordgo.InteractionCreate, *bot.Env) error {
			return nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	command, ok := registry.Get("elegancia")
	if !ok {
		t.Fatal("expected command to be registered")
	}
	if !strings.Contains(command.Description, "con eleganza") {
		t.Errorf("expected command description to carry the Italian Lux signature, got %q", command.Description)
	}
	if utf8.RuneCountInString(command.Description) > 100 {
		t.Errorf("command description exceeds Discord's limit: %d runes", utf8.RuneCountInString(command.Description))
	}
	if got := command.Options[0].Description; !strings.Contains(got, "con eleganza") {
		t.Errorf("expected option description to carry the Lux signature, got %q", got)
	}
	if got := command.Options[0].Choices[0].Name; !strings.HasPrefix(got, "✦ ") {
		t.Errorf("expected command choice to carry the Lux signature, got %q", got)
	}
	applicationCommands := registry.ToApplicationCommands()
	if len(applicationCommands) != 1 || !utf8.ValidString(applicationCommands[0].Description) {
		t.Fatal("expected a valid UTF-8 application-command description")
	}
	if got := utf8.RuneCountInString(applicationCommands[0].Description); got > 100 {
		t.Errorf("application-command description exceeds Discord's limit: %d runes", got)
	}
}
