package utility

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/bot"
)

type Cog struct{}

func New() *Cog {
	return &Cog{}
}

func (c *Cog) Register(r *bot.Registry) error {
	return r.RegisterAll(
		&bot.Command{
			Name:        "auditoria",
			Description: "Lucifer examina la seguridad y configuración de este servidor 🛡️",
			Category:    "utilidad",
			GuildOnly:   true,
			Handler:     c.handleAuditoria,
		},
		&bot.Command{
			Name:        "rolinfo",
			Description: "Inspecciona los permisos, miembros y color de un rol 🎭",
			Category:    "utilidad",
			GuildOnly:   true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "rol",
					Description: "El rol que deseas inspeccionar",
					Required:    true,
				},
			},
			Handler: c.handleRolInfo,
		},
		&bot.Command{
			Name:        "canalinfo",
			Description: "Examina los detalles técnicos de un canal de texto o voz 💬",
			Category:    "utilidad",
			GuildOnly:   true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "canal",
					Description: "Canal a consultar (por defecto: este canal)",
					Required:    false,
				},
			},
			Handler: c.handleCanalInfo,
		},
		&bot.Command{
			Name:        "recordatorio",
			Description: "Lucifer te enviará un recordatorio cuando venza el plazo ⏳",
			Category:    "utilidad",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "minutos",
					Description: "Minutos de espera (ej: 10)",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "mensaje",
					Description: "¿Qué deseas que Lucifer te recuerde?",
					Required:    true,
				},
			},
			Handler: c.handleRecordatorio,
		},
		&bot.Command{
			Name:        "encuesta",
			Description: "Crea una votación interactiva para las almas del servidor 📊",
			Category:    "utilidad",
			GuildOnly:   true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "pregunta",
					Description: "Tema de la votación",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "opcion1",
					Description: "Primera opción",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "opcion2",
					Description: "Segunda opción",
					Required:    true,
				},
			},
			Handler: c.handleEncuesta,
		},
		&bot.Command{
			Name:        "reloj",
			Description: "Consulta la hora actual en Los Ángeles y otras capitales 🌍",
			Category:    "utilidad",
			Handler:     c.handleReloj,
		},
		&bot.Command{
			Name:        "color",
			Description: "Previsualiza un código de color HEX y sus conversiones RGB 🎨",
			Category:    "utilidad",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "hex",
					Description: "Código hexadecimal (ej: #8B0000 o 8B0000)",
					Required:    true,
				},
			},
			Handler: c.handleColor,
		},
		&bot.Command{
			Name:        "botstats",
			Description: "Consulta las estadísticas internas del bot (solo su dueño).",
			Category:    "utilidad",
			OwnerOnly:   true,
			Handler:     c.handleBotStats,
		},
		&bot.Command{
			Name:        "invitar",
			Description: "Obtén el enlace de invitación para llevar a Lucifer a tu servidor 🍸",
			Category:    "utilidad",
			Handler:     c.handleInvitar,
		},
	)
}

func (c *Cog) handleAuditoria(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	guild, err := s.Guild(i.GuildID)
	if err != nil {
		_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{env.Theme.ErrorEmbed("Auditoría no disponible", "No se pudo consultar la información del servidor.")}},
		})
		return nil
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🛡️ Auditoría del Lux",
		Description: fmt.Sprintf("Lucifer presenta la evaluación de seguridad de **%s**.", guild.Name),
		Color:       env.Theme.ColorWarning,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: guild.IconURL("256")},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "👥 Almas Totales", Value: fmt.Sprintf("%d miembros", guild.MemberCount), Inline: true},
			{Name: "👑 Gobernante", Value: fmt.Sprintf("<@%s>", guild.OwnerID), Inline: true},
			{Name: "🛡️ Roles", Value: fmt.Sprintf("%d roles", len(guild.Roles)), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Seguridad del servidor"},
	}
	env.Theme.BrandEmbed(embed)

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	return nil
}

func (c *Cog) handleRolInfo(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	roleID := i.ApplicationCommandData().Options[0].RoleValue(s, i.GuildID).ID
	guild, _ := s.Guild(i.GuildID)

	var targetRole *discordgo.Role
	for _, r := range guild.Roles {
		if r.ID == roleID {
			targetRole = r
			break
		}
	}

	if targetRole == nil {
		_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Rol no encontrado."},
		})
		return nil
	}

	createdAt := getSnowflakeTime(targetRole.ID)

	embed := &discordgo.MessageEmbed{
		Title: "🎭 Información del Rol: " + targetRole.Name,
		Color: targetRole.Color,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "🆔 ID del Rol", Value: fmt.Sprintf("`%s`", targetRole.ID), Inline: true},
			{Name: "🎨 Color HEX", Value: fmt.Sprintf("#%06X", targetRole.Color), Inline: true},
			{Name: "📅 Fecha de Creación", Value: fmt.Sprintf("<t:%d:D>", createdAt.Unix()), Inline: true},
		},
	}
	env.Theme.BrandEmbed(embed)

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
	return nil
}

func (c *Cog) handleCanalInfo(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	channelID := i.ChannelID
	if len(i.ApplicationCommandData().Options) > 0 {
		channelID = i.ApplicationCommandData().Options[0].ChannelValue(s).ID
	}

	ch, err := s.Channel(channelID)
	if err != nil {
		_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ No pude leer la información del canal."},
		})
		return nil
	}

	createdAt := getSnowflakeTime(ch.ID)

	embed := &discordgo.MessageEmbed{
		Title: "💬 Canal: #" + ch.Name,
		Color: env.Theme.Primary,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "🆔 ID", Value: fmt.Sprintf("`%s`", ch.ID), Inline: true},
			{Name: "⏱️ Modo Lento", Value: fmt.Sprintf("%d seg", ch.RateLimitPerUser), Inline: true},
			{Name: "📅 Creado el", Value: fmt.Sprintf("<t:%d:D>", createdAt.Unix()), Inline: true},
		},
	}
	env.Theme.BrandEmbed(embed)

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
	return nil
}

func (c *Cog) handleRecordatorio(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	minutes := i.ApplicationCommandData().Options[0].IntValue()
	message := i.ApplicationCommandData().Options[1].StringValue()

	targetTime := time.Now().Add(time.Duration(minutes) * time.Minute)

	embed := &discordgo.MessageEmbed{
		Title:       "⏳ Favor Concedido por Lucifer",
		Description: fmt.Sprintf("Te recordaré tu pendiente en **%d minutos** (<t:%d:R>).\n\n**Nota:** *\"%s\"*", minutes, targetTime.Unix(), message),
		Color:       env.Theme.ColorWarning,
		Footer:      &discordgo.MessageEmbedFooter{Text: "El Diablo nunca olvida una cita."},
	}
	env.Theme.BrandEmbed(embed)

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})

	go func(userID, channelID, note string, wait time.Duration) {
		time.Sleep(wait)
		remindEmbed := &discordgo.MessageEmbed{
			Title:       "🔔 Recordatorio del Lux",
			Description: fmt.Sprintf("<@%s>, ha llegado el momento.\n\n**Tu recordatorio:**\n> %s", userID, note),
			Color:       0x8B0000,
			Footer:      &discordgo.MessageEmbedFooter{Text: "Cita programada concluida."},
		}
		env.Theme.BrandEmbed(remindEmbed)
		_, _ = discordutil.ChannelMessageSendEmbed(s, channelID, remindEmbed)
	}(i.Member.User.ID, i.ChannelID, message, time.Duration(minutes)*time.Minute)

	return nil
}

func (c *Cog) handleEncuesta(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	opts := i.ApplicationCommandData().Options
	pregunta := opts[0].StringValue()
	op1 := opts[1].StringValue()
	op2 := opts[2].StringValue()

	description := fmt.Sprintf("**%s**\n\n1️⃣ %s\n2️⃣ %s", pregunta, op1, op2)

	embed := &discordgo.MessageEmbed{
		Title:       "📊 Votación en el Lux",
		Description: description,
		Color:       env.Theme.Primary,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Iniciada por " + i.Member.User.Username},
	}
	env.Theme.BrandEmbed(embed)

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Encuesta publicada.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	msg, err := discordutil.ChannelMessageSendEmbed(s, i.ChannelID, embed)
	if err == nil {
		_ = s.MessageReactionAdd(i.ChannelID, msg.ID, "1️⃣")
		_ = s.MessageReactionAdd(i.ChannelID, msg.ID, "2️⃣")
	}
	return nil
}

func (c *Cog) handleReloj(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	laLocation, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return fmt.Errorf("cargar zona horaria de Los Ángeles: %w", err)
	}
	madridLocation, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return fmt.Errorf("cargar zona horaria de Madrid: %w", err)
	}
	now := time.Now()
	laTime := now.In(laLocation)
	madridTime := now.In(madridLocation)

	embed := &discordgo.MessageEmbed{
		Title: "🌍 Reloj Mundial del Club",
		Color: env.Theme.ColorNeutral,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "🍸 Los Ángeles (Lux)", Value: laTime.Format("03:04 PM (15:04)"), Inline: true},
			{Name: "💃 Madrid", Value: madridTime.Format("03:04 PM (15:04)"), Inline: true},
			{Name: "🌐 UTC", Value: now.Format("03:04 PM (15:04)"), Inline: true},
		},
	}
	env.Theme.BrandEmbed(embed)

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
	return nil
}

func (c *Cog) handleColor(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	hexInput := i.ApplicationCommandData().Options[0].StringValue()
	cleanHex := strings.TrimPrefix(hexInput, "#")

	val, err := strconv.ParseUint(cleanHex, 16, 32)
	if err != nil || len(cleanHex) != 6 {
		_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ Ingresa un HEX válido (ej: `#8B0000`).",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return nil
	}

	embed := &discordgo.MessageEmbed{
		Title: "🎨 Previsualización: #" + strings.ToUpper(cleanHex),
		Color: int(val),
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: fmt.Sprintf("https://dummyimage.com/200x200/%s/%s.png", cleanHex, cleanHex),
		},
	}
	env.Theme.BrandEmbed(embed)

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
	return nil
}

func (c *Cog) handleBotStats(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	checks := checkBotDependencies(ctx, s, env)
	embed := &discordgo.MessageEmbed{
		Title: "⚙️ Estado del bot · Lucifer",
		Color: env.Theme.Primary,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "🧠 RAM", Value: fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024), Inline: true},
			{Name: "🧵 Hilos", Value: fmt.Sprintf("%d", runtime.NumGoroutine()), Inline: true},
			{Name: "⏱️ Uptime", Value: env.Uptime().Round(time.Second).String(), Inline: true},
			{Name: "Discord Gateway", Value: checks.discord, Inline: true},
			{Name: "PostgreSQL", Value: checks.postgres, Inline: true},
			{Name: "Redis", Value: checks.redis, Inline: true},
			{Name: "Cerebras AI", Value: checks.cerebras, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Comprobación de IA: una petición breve a Cerebras; puede contar para el límite de uso."},
	}
	env.Theme.BrandEmbed(embed)

	return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

type botHealthChecks struct {
	discord  string
	postgres string
	redis    string
	cerebras string
}

func checkBotDependencies(ctx context.Context, session *discordgo.Session, env *bot.Env) botHealthChecks {
	checks := botHealthChecks{
		discord:  "❌ Gateway no listo",
		postgres: "⚪ No configurado",
		redis:    "⚪ No configurado",
		cerebras: "⚪ No configurado",
	}
	if session != nil && session.DataReady {
		checks.discord = "✅ Gateway listo"
	}

	type probe struct {
		name string
		run  func(context.Context) error
	}
	probes := make([]probe, 0, 3)
	if env != nil && env.Store != nil {
		probes = append(probes, probe{name: "postgres", run: env.Store.Health})
	}
	if env != nil && env.Cache != nil {
		probes = append(probes, probe{name: "redis", run: env.Cache.Health})
	}
	if env != nil && env.AI != nil {
		probes = append(probes, probe{name: "cerebras", run: env.AI.Health})
	}

	type result struct {
		name    string
		elapsed time.Duration
		err     error
	}
	results := make(chan result, len(probes))
	for _, current := range probes {
		current := current
		go func() {
			probeCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()
			started := time.Now()
			err := current.run(probeCtx)
			results <- result{name: current.name, elapsed: time.Since(started), err: err}
		}()
	}
	for range probes {
		current := <-results
		status := fmt.Sprintf("✅ Disponible · %d ms", current.elapsed.Milliseconds())
		if current.err != nil {
			status = fmt.Sprintf("❌ Sin respuesta · %d ms", current.elapsed.Milliseconds())
		}
		switch current.name {
		case "postgres":
			checks.postgres = status
		case "redis":
			checks.redis = status
		case "cerebras":
			checks.cerebras = status
		}
	}
	return checks
}

func (c *Cog) handleInvitar(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	permissions := discordgo.PermissionViewChannel |
		discordgo.PermissionSendMessages |
		discordgo.PermissionEmbedLinks |
		discordgo.PermissionReadMessageHistory |
		discordgo.PermissionManageMessages |
		discordgo.PermissionBanMembers |
		discordgo.PermissionKickMembers |
		discordgo.PermissionModerateMembers |
		discordgo.PermissionManageChannels |
		discordgo.PermissionManageRoles |
		discordgo.PermissionManageEvents
	inviteURL := fmt.Sprintf("https://discord.com/oauth2/authorize?client_id=%s&permissions=%d&scope=bot%%20applications.commands", s.State.User.ID, permissions)

	embed := &discordgo.MessageEmbed{
		Title:       "🍸 Lleva el Lux a tu Propio Servidor",
		Description: fmt.Sprintf("[**Haz clic aquí para invitar a Lucifer**](%s)", inviteURL),
		Color:       env.Theme.ColorDanger,
	}
	env.Theme.BrandEmbed(embed)

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
	return nil
}

func getSnowflakeTime(snowflake string) time.Time {
	id, err := strconv.ParseInt(snowflake, 10, 64)
	if err != nil {
		return time.Time{}
	}
	const discordEpoch = int64(1420070400000)
	return time.UnixMilli((id >> 22) + discordEpoch)
}

func stringPtr(value string) *string {
	return &value
}
