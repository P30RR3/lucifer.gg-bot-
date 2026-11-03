package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/antiraid"
	"krypton.dev/internal/cache"
	"krypton.dev/internal/cerebras"
	"krypton.dev/internal/config"
	"krypton.dev/internal/storage"
	"krypton.dev/internal/ui"
	"krypton.dev/internal/ui/live"
)

type Handler func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *Env) error
type AutocompleteHandler func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *Env) error

type Command struct {
	Name          string
	Description   string
	Category      string
	Options       []*discordgo.ApplicationCommandOption
	AppCmd        *discordgo.ApplicationCommand
	DefaultPerm   int64
	OwnerOnly     bool
	GuildOnly     bool
	Cooldown      time.Duration
	Handler       Handler
	LegacyHandler func(*Context)
	Autocomplete  AutocompleteHandler
}

type Env struct {
	Config    *config.Config
	Registry  *Registry
	Store     *storage.Store
	Cache     *cache.Client
	AI        *cerebras.Client
	Batcher   *cerebras.Batcher
	Theme     *ui.Theme
	Live      *live.Manager
	Session   *discordgo.Session
	Auditor   *Auditor
	Log       *slog.Logger
	Engine    *antiraid.Engine
	Lockdown  *antiraid.LockdownManager
	Version   string
	Commit    string
	BuildDate string
	StartedAt time.Time
}

type Registry struct {
	mu         sync.RWMutex
	commands   map[string]*Command
	categories map[string][]string
	order      []string
}

var (
	ErrDuplicateCommand = errors.New("bot: comando duplicado")
	ErrEmptyName        = errors.New("bot: comando sin nombre")
	ErrNilHandler       = errors.New("bot: comando sin handler")
)

func NewRegistry() *Registry {
	return &Registry{
		commands:   make(map[string]*Command),
		categories: make(map[string][]string),
	}
}

func (r *Registry) Register(cmd *Command) error {
	if cmd == nil {
		return ErrNilHandler
	}
	if strings.TrimSpace(cmd.Name) == "" {
		return ErrEmptyName
	}
	if cmd.Handler == nil {
		return ErrNilHandler
	}

	cmd.Name = strings.ToLower(strings.TrimSpace(cmd.Name))
	cmd.Category = strings.ToLower(strings.TrimSpace(cmd.Category))
	if cmd.Category == "" {
		cmd.Category = "general"
	}
	if cmd.Description == "" {
		cmd.Description = "Sin descripción"
	}
	cmd.Description = addCommandElegance(cmd.Description, 100)
	cmd.Options = addOptionElegance(cmd.Options)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.commands[cmd.Name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateCommand, cmd.Name)
	}

	r.commands[cmd.Name] = cmd
	r.order = append(r.order, cmd.Name)
	r.categories[cmd.Category] = append(r.categories[cmd.Category], cmd.Name)
	return nil
}

func addCommandElegance(description string, limit int) string {
	const signature = " · con eleganza"
	if strings.Contains(strings.ToLower(description), "con eleganza") {
		return description
	}
	runes := []rune(description)
	if len(runes)+utf8.RuneCountInString(signature) > limit {
		runes = runes[:limit-utf8.RuneCountInString(signature)]
		description = strings.TrimRight(string(runes), " .,:;")
	}
	return description + signature
}

func addOptionElegance(options []*discordgo.ApplicationCommandOption) []*discordgo.ApplicationCommandOption {
	for _, option := range options {
		if option == nil {
			continue
		}
		option.Description = addCommandElegance(option.Description, 100)
		for _, choice := range option.Choices {
			if choice == nil {
				continue
			}
			if utf8.RuneCountInString(choice.Name) <= 97 && !strings.HasPrefix(choice.Name, "✦ ") {
				choice.Name = "✦ " + choice.Name
			}
		}
		option.Options = addOptionElegance(option.Options)
	}
	return options
}

func (r *Registry) RegisterAll(cmds ...*Command) error {
	for _, c := range cmds {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) AddCommand(cmd *Command) error {
	if cmd == nil {
		return ErrNilHandler
	}
	if cmd.AppCmd != nil {
		cmd.Name = cmd.AppCmd.Name
		cmd.Description = cmd.AppCmd.Description
		cmd.Options = cmd.AppCmd.Options
		if cmd.AppCmd.DefaultMemberPermissions != nil {
			cmd.DefaultPerm = *cmd.AppCmd.DefaultMemberPermissions
		}
		cmd.GuildOnly = true
		if cmd.AppCmd.DMPermission != nil {
			cmd.GuildOnly = !*cmd.AppCmd.DMPermission
		}
	}
	if cmd.Handler == nil && cmd.LegacyHandler != nil {
		legacyHandler := cmd.LegacyHandler
		cmd.Handler = func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *Env) error {
			legacyHandler(&Context{
				Context:     ctx,
				Session:     s,
				Interaction: i,
				Env:         env,
			})
			return nil
		}
	}
	return r.Register(cmd)
}

func (r *Registry) Get(name string) (*Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.commands[strings.ToLower(name)]
	return cmd, ok
}

func (r *Registry) All() []*Command {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Command, 0, len(r.order))
	for _, name := range r.order {
		if cmd, ok := r.commands[name]; ok {
			out = append(out, cmd)
		}
	}
	return out
}

func (r *Registry) ByCategory(category string) []*Command {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := r.categories[strings.ToLower(category)]
	out := make([]*Command, 0, len(names))
	for _, name := range names {
		if cmd, ok := r.commands[name]; ok {
			out = append(out, cmd)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (r *Registry) Categories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.categories))
	for c := range r.categories {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func (r *Registry) CategoryCount(category string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.categories[strings.ToLower(category)])
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.commands)
}

// SANITIZADOR EXTREMO: Limpia tildes, eñes y caracteres raros para que Discord no llore
func sanitizeOptions(opts []*discordgo.ApplicationCommandOption) []*discordgo.ApplicationCommandOption {
	if opts == nil {
		return nil
	}
	var result []*discordgo.ApplicationCommandOption
	for _, opt := range opts {
		newOpt := *opt

		// Limpia el nombre (solo a-z, 0-9, -, _)
		var cleanName strings.Builder
		for _, r := range strings.ToLower(newOpt.Name) {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				cleanName.WriteRune(r)
			}
		}
		newOpt.Name = cleanName.String()
		if len(newOpt.Name) > 32 {
			newOpt.Name = newOpt.Name[:32]
		}

		newOpt.Description = truncateDiscordText(newOpt.Description, 100)

		if len(newOpt.Choices) > 0 {
			for _, choice := range newOpt.Choices {
				if choice == nil {
					continue
				}
				choice.Name = truncateDiscordText(choice.Name, 100)
				if strVal, ok := choice.Value.(string); ok {
					choice.Value = truncateDiscordText(strVal, 100)
				}
			}
		}

		if len(newOpt.Options) > 0 {
			newOpt.Options = sanitizeOptions(newOpt.Options)
		}

		result = append(result, &newOpt)
	}
	return result
}

func (r *Registry) ToApplicationCommands() []*discordgo.ApplicationCommand {
	all := r.All()
	out := make([]*discordgo.ApplicationCommand, 0, len(all))
	for _, cmd := range all {
		dmPerm := !cmd.GuildOnly
		var defaultPermissions *int64
		if cmd.DefaultPerm != 0 {
			permissions := cmd.DefaultPerm
			defaultPermissions = &permissions
		}

		// Limpia el nombre del comando
		var cleanName strings.Builder
		for _, r := range strings.ToLower(cmd.Name) {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				cleanName.WriteRune(r)
			}
		}
		name := cleanName.String()
		if len(name) > 32 {
			name = name[:32]
		}

		desc := truncateDiscordText(cmd.Description, 100)

		opts := sanitizeOptions(cmd.Options)

		app := &discordgo.ApplicationCommand{
			Name:                     name,
			Description:              desc,
			Options:                  opts,
			DefaultMemberPermissions: defaultPermissions,
			DMPermission:             &dmPerm,
		}
		out = append(out, app)
	}
	return out
}

func truncateDiscordText(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}

func (e *Env) Uptime() time.Duration {
	if e.StartedAt.IsZero() {
		return 0
	}
	return time.Since(e.StartedAt)
}

func (e *Env) GuildSettings(ctx context.Context, guildID string) (*storage.GuildSettings, error) {
	if e.Store == nil {
		return storage.DefaultGuildSettings(guildID), nil
	}
	return e.Store.GetOrCreateGuild(ctx, guildID)
}
