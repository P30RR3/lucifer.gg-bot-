package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/cache"
	"krypton.dev/pkg/discordutil"
)

type ComponentHandler func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *Env) error

type ModalHandler func(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *Env) error
type LegacyComponentHandler func(ctx *Context)

type Router struct {
	session          *discordgo.Session
	env              *Env
	log              *slog.Logger
	cooldowns        sync.Map
	components       map[string]ComponentHandler
	modals           map[string]ModalHandler
	legacyComponents map[string]LegacyComponentHandler
	legacyModals     map[string]LegacyComponentHandler
	mu               sync.RWMutex
}

var (
	ErrHandlerPanic = errors.New("router: handler panic")
	ErrOnCooldown   = errors.New("router: comando en cooldown")
	ErrNoPermission = errors.New("router: permisos insuficientes")
	ErrDMOnly       = errors.New("router: comando solo en servidor")
)

func NewRouter(session *discordgo.Session, env *Env) *Router {
	return &Router{
		session:          session,
		env:              env,
		log:              env.Log.With("component", "router"),
		components:       make(map[string]ComponentHandler),
		modals:           make(map[string]ModalHandler),
		legacyComponents: make(map[string]LegacyComponentHandler),
		legacyModals:     make(map[string]LegacyComponentHandler),
	}
}

func (r *Router) RegisterComponent(prefix string, h ComponentHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.components[strings.ToLower(prefix)] = h
}

func (r *Router) RegisterModal(prefix string, h ModalHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modals[strings.ToLower(prefix)] = h
}

func (r *Router) RegisterLegacyComponent(prefix string, h LegacyComponentHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.legacyComponents[strings.ToLower(prefix)] = h
}

func (r *Router) RegisterLegacyModal(prefix string, h LegacyComponentHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.legacyModals[strings.ToLower(prefix)] = h
}

func (r *Router) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if rec := recover(); rec != nil {
			r.log.Error("panic recuperado en handler",
				"panic", rec,
				"stack", string(debug.Stack()),
			)
			_ = r.respondError(s, i, "Error interno. El equipo ha sido notificado.")
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	r.loadResponseVisibility(ctx, i)

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		r.dispatchCommand(ctx, s, i)
	case discordgo.InteractionApplicationCommandAutocomplete:
		r.dispatchAutocomplete(ctx, s, i)
	case discordgo.InteractionMessageComponent:
		r.dispatchComponent(ctx, s, i)
	case discordgo.InteractionModalSubmit:
		r.dispatchModal(ctx, s, i)
	default:
		r.log.Debug("unknown interaction type", "type", i.Type)
	}
}

func (r *Router) loadResponseVisibility(ctx context.Context, interaction *discordgo.InteractionCreate) {
	if r.env == nil || r.env.Store == nil || interaction == nil || interaction.GuildID == "" {
		return
	}
	userID := r.userID(interaction)
	if userID == "" {
		return
	}
	visible, err := r.env.Store.LuxResponsesPublic(ctx, interaction.GuildID, userID)
	if err != nil {
		r.log.Warn("no se pudo cargar la preferencia de visibilidad", "guild_id", interaction.GuildID, "user_id", userID, "err", err)
		visible = false
	}
	discordutil.SetResponseVisibility(interaction.GuildID, userID, visible)
}

func (r *Router) dispatchCommand(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	cmd, ok := r.env.Registry.Get(data.Name)
	if !ok {
		_ = r.respondError(s, i, "Comando no encontrado.")
		return
	}

	if cmd.GuildOnly && i.GuildID == "" {
		_ = r.respondError(s, i, "Este comando solo funciona en servidores.")
		return
	}

	if cmd.OwnerOnly && !r.isOwner(i) {
		_ = r.respondError(s, i, "Este comando es exclusivo del owner del bot.")
		return
	}

	if cmd.DefaultPerm != 0 && !r.hasPermissions(i, cmd.DefaultPerm) {
		_ = r.respondError(s, i, "No tienes los permisos necesarios.")
		return
	}

	if cmd.Cooldown > 0 {
		key := r.cooldownKey(i, cmd.Name)
		if remaining := r.checkCooldown(key); remaining > 0 {
			_ = r.respondCooldown(s, i, remaining)
			return
		}
		r.setCooldown(key, cmd.Cooldown)
	}

	start := time.Now()
	if err := cmd.Handler(ctx, s, i, r.env); err != nil {
		r.log.Error("error ejecutando comando",
			"command", cmd.Name,
			"guild", i.GuildID,
			"err", err,
		)
		_ = r.respondError(s, i, err.Error())
		return
	}

	r.log.Info("comando ejecutado",
		"command", cmd.Name,
		"guild", i.GuildID,
		"user", r.userID(i),
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

func (r *Router) dispatchAutocomplete(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmd, ok := r.env.Registry.Get(i.ApplicationCommandData().Name)
	if !ok || cmd.Autocomplete == nil {
		_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "",
				Choices: []*discordgo.ApplicationCommandOptionChoice{},
			},
		})
		return
	}
	if err := cmd.Autocomplete(ctx, s, i, r.env); err != nil {
		r.log.Warn("error en autocomplete", "command", cmd.Name, "err", err)
	}
}

func (r *Router) dispatchComponent(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	prefix := extractPrefix(customID)

	r.mu.RLock()
	h, ok := r.components[prefix]
	legacy, legacyOK := r.legacyComponents[prefix]
	r.mu.RUnlock()

	if !ok && !legacyOK {
		r.log.Debug("componente sin handler", "custom_id", customID, "prefix", prefix)
		_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Este componente ya expiro. Vuelve a ejecutar el comando.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if legacyOK {
		legacy(&Context{Context: ctx, Session: s, Interaction: i, Env: r.env})
		return
	}

	if err := h(ctx, s, i, r.env); err != nil {
		r.log.Error("error en componente",
			"custom_id", customID,
			"err", err,
		)
		_ = r.respondError(s, i, err.Error())
	}
}

func (r *Router) dispatchModal(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	prefix := extractPrefix(customID)

	r.mu.RLock()
	h, ok := r.modals[prefix]
	legacy, legacyOK := r.legacyModals[prefix]
	r.mu.RUnlock()

	if !ok && !legacyOK {
		r.log.Debug("modal sin handler", "custom_id", customID, "prefix", prefix)
		_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Este formulario ya expiro. Vuelve a ejecutar el comando.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if legacyOK {
		legacy(&Context{Context: ctx, Session: s, Interaction: i, Env: r.env})
		return
	}

	if err := h(ctx, s, i, r.env); err != nil {
		r.log.Error("error en modal", "custom_id", customID, "err", err)
	}
}

func (r *Router) cooldownKey(i *discordgo.InteractionCreate, cmd string) string {
	return r.userID(i) + ":" + cmd
}

func (r *Router) checkCooldown(key string) time.Duration {
	val, ok := r.cooldowns.Load(key)
	if !ok {
		return 0
	}
	until, ok := val.(time.Time)
	if !ok {
		return 0
	}
	remaining := time.Until(until)
	if remaining <= 0 {
		r.cooldowns.Delete(key)
		return 0
	}
	return remaining
}

func (r *Router) setCooldown(key string, d time.Duration) {
	r.cooldowns.Store(key, time.Now().Add(d))
}

func (r *Router) hasPermissions(i *discordgo.InteractionCreate, perm int64) bool {
	if i.Member == nil {
		return false
	}
	if i.Member.Permissions&perm == perm {
		return true
	}
	if i.Member.Permissions&discordgo.PermissionAdministrator == discordgo.PermissionAdministrator {
		return true
	}
	return false
}

func (r *Router) isOwner(i *discordgo.InteractionCreate) bool {
	return r.env != nil && r.env.Config != nil &&
		r.env.Config.Discord.BotOwnerID != "" &&
		r.userID(i) == r.env.Config.Discord.BotOwnerID
}

func (r *Router) userID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func (r *Router) respondError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) error {
	embed := r.env.Theme.ErrorEmbed("Error", discordutil.Truncate(msg, 1800))
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	}
	if err := discordutil.InteractionRespond(s, i.Interaction, response); err != nil {
		_, followupErr := discordutil.FollowupMessageCreate(s, i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		})
		if followupErr != nil {
			return fmt.Errorf("initial response failed: %w; follow-up failed: %v", err, followupErr)
		}
	}
	return nil
}

func (r *Router) respondCooldown(s *discordgo.Session, i *discordgo.InteractionCreate, remaining time.Duration) error {
	embed := r.env.Theme.WarningEmbed(
		"? Cooldown activo",
		fmt.Sprintf("Espera **%s** antes de volver a usar este comando.", discordutil.HumanDuration(remaining)),
	)
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	}
	if err := discordutil.InteractionRespond(s, i.Interaction, response); err != nil {
		_, followupErr := discordutil.FollowupMessageCreate(s, i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		})
		if followupErr != nil {
			return fmt.Errorf("initial response failed: %w; follow-up failed: %v", err, followupErr)
		}
	}
	return nil
}

func extractPrefix(customID string) string {
	idx := strings.Index(customID, ":")
	if idx < 0 {
		return strings.ToLower(customID)
	}
	return strings.ToLower(customID[:idx])
}

var _ = cache.KeyNamespace
