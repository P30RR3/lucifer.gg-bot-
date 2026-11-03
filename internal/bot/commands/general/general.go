package general

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"fmt"
	"math/rand"
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
			Name:        "help",
			Description: "📖 Abre el índice del Lux: elige una sección y descubre cada comando.",
			Category:    "general",
			Handler:     c.handleHelp,
		},
		&bot.Command{
			Name:        "ping",
			Description: "🏓 Comprueba en un instante si el Lux y sus portales siguen en línea.",
			Category:    "general",
			Handler:     c.handlePing,
		},
		&bot.Command{
			Name:        "perfil",
			Description: "📜 Consulta el perfil público y la antigüedad de un miembro del club.",
			Category:    "general",
			GuildOnly:   true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "usuario",
					Description: "El alma que deseas investigar (opcional)",
					Required:    false,
				},
			},
			Handler: c.handleProfile,
		},
		&bot.Command{
			Name:        "deseo",
			Description: "👁️ Lucifer mira a los ojos a un alma y le pregunta qué desea de verdad.",
			Category:    "general",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "victima",
					Description: "La persona a la que Lucifer mirará a los ojos",
					Required:    true,
				},
			},
			Handler: c.handleDesire,
		},
		&bot.Command{
			Name:        "trato",
			Description: "🎲 Lanza los dados con Lucifer: más de siete y la fortuna te sonríe.",
			Category:    "general",
			Handler:     c.handleDeal,
		},
		&bot.Command{
			Name:        "avatar",
			Description: "🪞 Contempla el reflejo de un miembro en alta resolución.",
			Category:    "general",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "usuario",
					Description: "El usuario del que quieres ver el avatar",
					Required:    false,
				},
			},
			Handler: c.handleAvatar,
		},
		&bot.Command{
			Name:        "moneda",
			Description: "🪙 Lucifer lanza una moneda; la suerte decidirá entre cara y cruz.",
			Category:    "general",
			Handler:     c.handleCoin,
		},
	)
}

func RegisterComponents(router *bot.Router) {
	router.RegisterComponent("help", handleHelpComponent)
}

func (c *Cog) handlePing(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	start := time.Now()

	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🦇 *Abriendo portal al inframundo...* `[■■□□□□□□□□]`",
		},
	})
	time.Sleep(500 * time.Millisecond)

	latency := time.Since(start).Milliseconds()

	embed := &discordgo.MessageEmbed{
		Title:       "🏓・EL PORTAL RESPONDE",
		Description: fmt.Sprintf("Una conexión impecable, como debe ser en el Lux.\n\n**Latencia del Gateway:** `%d ms`", latency),
		Color:       env.Theme.ColorSuccess,
		Footer:      &discordgo.MessageEmbedFooter{Text: env.Theme.FooterText, IconURL: env.Theme.FooterIcon},
	}
	env.Theme.BrandEmbed(embed)

	_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
		Content: stringPtr("✅ *Portal estabilizado.* `[■■■■■■■■■■]`"),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
	return nil
}

func (c *Cog) handleProfile(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	var user *discordgo.User
	if len(i.ApplicationCommandData().Options) > 0 {
		user = i.ApplicationCommandData().Options[0].UserValue(s)
	} else {
		user = i.Member.User
	}

	member, err := s.GuildMember(i.GuildID, user.ID)
	if err != nil {
		_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ No pude encontrar el expediente de esta alma."},
		})
		return nil
	}

	// member.JoinedAt ya es un time.Time directo en discordgo
	joinedAt := member.JoinedAt
	createdAt := getSnowflakeTime(user.ID)

	embed := &discordgo.MessageEmbed{
		Title:       "📜・EXPEDIENTE DEL CLUB・" + user.Username,
		Description: "Una mirada elegante a la trayectoria de esta alma en el Lux.",
		Color:       env.Theme.Primary,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: user.AvatarURL("256")},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "🏷️ Nombre", Value: user.String(), Inline: true},
			{Name: "🆔 ID", Value: user.ID, Inline: true},
			{Name: "📅 Llegó al mundo", Value: fmt.Sprintf("<t:%d:R>", createdAt.Unix()), Inline: false},
			{Name: "🚪 Entró al club", Value: fmt.Sprintf("<t:%d:R>", joinedAt.Unix()), Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: env.Theme.FooterText, IconURL: env.Theme.FooterIcon},
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

func (c *Cog) handleDesire(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	targetUser := i.ApplicationCommandData().Options[0].UserValue(s)

	embed := &discordgo.MessageEmbed{
		Title:       "👁️ El Don de Lucifer",
		Description: fmt.Sprintf("╭────────────── ✦ ──────────────╮\n%s, mírame a los ojos, querido.\n\n**«Dime... ¿qué es lo que realmente deseas?»**\n╰────────────── ✦ ──────────────╯", targetUser.Mention()),
		Color:       env.Theme.ColorDanger,
		Image:       &discordgo.MessageEmbedImage{URL: "https://media.giphy.com/media/l0HlCqV35hdEg2GUo/giphy.gif"},
		Footer:      &discordgo.MessageEmbedFooter{Text: "Responde con cuidado... el Diablo siempre cobra sus favores."},
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

func (c *Cog) handleDeal(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🎲 *Haciendo un trato con el Diablo... Lanzando los dados...* `[■■■□□□□□□□]`",
		},
	})
	time.Sleep(1 * time.Second)

	rand.Seed(time.Now().UnixNano())
	dado1 := rand.Intn(6) + 1
	dado2 := rand.Intn(6) + 1
	total := dado1 + dado2

	var resultado string
	var color int

	if total > 7 {
		resultado = "¡Has ganado! Lucifer te debe un favor. Úsalo sabiamente."
		color = env.Theme.ColorSuccess
	} else if total == 7 {
		resultado = "Empate. El Diablo te da una segunda oportunidad para vivir."
		color = env.Theme.ColorWarning
	} else {
		resultado = "Has perdido. Tu alma ahora le pertenece al Lux. 😈"
		color = env.Theme.ColorDanger
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎲・EL TRATO DEL LUX",
		Description: fmt.Sprintf("Lanzaste los dados y sacaste **%d** y **%d** (Total: **%d**).\n\n%s", dado1, dado2, total, resultado),
		Color:       color,
	}
	env.Theme.BrandEmbed(embed)

	_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
		Content: stringPtr("✅ *Los dados han hablado.* `[■■■■■■■■■■]`"),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
	return nil
}

func (c *Cog) handleAvatar(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	var user *discordgo.User
	if len(i.ApplicationCommandData().Options) > 0 {
		user = i.ApplicationCommandData().Options[0].UserValue(s)
	} else {
		user = i.Member.User
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🪞・EL REFLEJO DEL LUX・" + user.Username,
		Description: "╭─────────────── ✦ ───────────────╮\n      Este es tu reflejo en el Lux.\n╰─────────────── ✦ ───────────────╯",
		Color:       env.Theme.Primary,
		Image:       &discordgo.MessageEmbedImage{URL: user.AvatarURL("1024")},
		Footer:      &discordgo.MessageEmbedFooter{Text: "🍸 La noche siempre devuelve la mirada."},
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

func (c *Cog) handleHelp(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	embed, components := renderHelp(env, s, "")
	return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: components},
	})
}

func handleHelpComponent(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	category := ""
	if i.MessageComponentData().CustomID == "help:section" {
		selected := i.MessageComponentData().Values
		if len(selected) != 1 {
			return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Scegli una sola sezione del club e ti mostrerò i suoi comandi.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
		category = selected[0]
		if len(env.Registry.ByCategory(category)) == 0 {
			return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Esa sección ya no está disponible. Vuelve a abrir `/help` para consultar el índice actualizado.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
	}
	embed, components := renderHelp(env, s, category)
	return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: components},
	})
}

func renderHelp(env *bot.Env, session *discordgo.Session, selectedCategory string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := &discordgo.MessageEmbed{
		Title:       "📖  La guía del Lux",
		Description: "╭────────────── ✦ ──────────────╮\nUna buena noche empieza por la puerta correcta.\nScegli una sezione nel menu e scopri cosa sa fare ogni comando.\n╰────────────── ✦ ──────────────╯",
		Color:       env.Theme.Primary,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: session.State.User.AvatarURL("256")},
	}
	if selectedCategory != "" {
		commands := env.Registry.ByCategory(selectedCategory)
		embed.Title = helpCategoryName(selectedCategory)
		embed.Description = "Questi sono i comandi di questa sezione. Le spiegazioni sono qui, nello stesso messaggio; usa il menu per esplorare il club o torna all'indice."
		for _, command := range commands {
			for _, line := range helpCommandLines(command) {
				if len(embed.Fields) >= 25 {
					break
				}
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:  truncateHelpField(line.name, 256),
					Value: truncateHelpField(line.description, 1024),
				})
			}
		}
		if len(embed.Fields) == 0 {
			embed.Description = "Esta sección aún no tiene comandos registrados."
		}
	}
	env.Theme.BrandEmbed(embed)
	return embed, helpComponents(env.Registry, selectedCategory)
}

type helpEntry struct {
	name        string
	description string
}

func helpCommandLines(command *bot.Command) []helpEntry {
	if command == nil {
		return nil
	}
	ownerLabel := ""
	if command.OwnerOnly {
		ownerLabel = " · solo propietario"
	}
	options := command.Options
	if command.AppCmd != nil {
		options = command.AppCmd.Options
	}
	if len(options) == 0 {
		return []helpEntry{{name: "/" + command.Name, description: command.Description + ownerLabel}}
	}

	var lines []helpEntry
	var visit func(options []*discordgo.ApplicationCommandOption, path []string)
	visit = func(options []*discordgo.ApplicationCommandOption, path []string) {
		for _, option := range options {
			if option == nil {
				continue
			}
			switch option.Type {
			case discordgo.ApplicationCommandOptionSubCommandGroup:
				visit(option.Options, append(path, option.Name))
			case discordgo.ApplicationCommandOptionSubCommand:
				fullPath := append(append([]string(nil), path...), option.Name)
				lines = append(lines, helpEntry{
					name:        fmt.Sprintf("/%s %s", command.Name, strings.Join(fullPath, " ")),
					description: option.Description + ownerLabel,
				})
			}
		}
	}
	visit(options, nil)
	if len(lines) == 0 {
		lines = append(lines, helpEntry{name: "/" + command.Name, description: command.Description + ownerLabel})
	}
	return lines
}

func helpComponents(registry *bot.Registry, selected string) []discordgo.MessageComponent {
	categories := registry.Categories()
	options := make([]discordgo.SelectMenuOption, 0, len(categories))
	for _, category := range categories {
		options = append(options, discordgo.SelectMenuOption{
			Label:       truncateHelpField(helpCategoryName(category), 100),
			Value:       category,
			Description: fmt.Sprintf("%d comandos en esta sección", len(registry.ByCategory(category))),
			Default:     selected == category,
		})
	}
	if len(options) == 0 {
		return nil
	}
	components := []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.SelectMenu{
			MenuType:    discordgo.StringSelectMenu,
			CustomID:    "help:section",
			Placeholder: "✦ Elige una sección del club ✦",
			Options:     options,
			MinValues:   intPtr(1),
			MaxValues:   1,
		},
	}}}
	if selected != "" {
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				CustomID: "help:home",
				Label:    "Volver al panel de ayuda",
				Emoji:    &discordgo.ComponentEmoji{Name: "↩️"},
				Style:    discordgo.SecondaryButton,
			},
		}})
	}
	return components
}

func truncateHelpField(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

func intPtr(value int) *int {
	return &value
}

func helpCategoryName(category string) string {
	names := map[string]string{
		"agenda":     "🗓️・Agenda del Lux",
		"antiraid":   "🚨・Defensas del club",
		"auditoria":  "📜・Diario del Inframundo",
		"comunidad":  "💡・Comunidad y ayuda",
		"config":     "⚙️・Decretos del servidor",
		"diversion":  "🎭・Secretos y juegos",
		"economia":   "🪙・Economía del club",
		"general":    "🍸・El club",
		"ia":         "🧠・Escritorio de Lucifer",
		"moderacion": "🛡️・Guardia del Lux",
		"sanciones":  "🔥・El Inframundo",
		"utilidad":   "🕯️・Herramientas del Lux",
	}
	if name, ok := names[category]; ok {
		return name
	}
	return "✦・" + strings.ToUpper(category)
}

func (c *Cog) handleCoin(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	_ = discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{env.Theme.WarningEmbed("Lucifer lanza la moneda", "La suerte está a punto de decidir.")},
		},
	})
	time.Sleep(1 * time.Second)

	rand.Seed(time.Now().UnixNano())
	resultado := "Cara"
	if rand.Intn(2) == 0 {
		resultado = "Cruz"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🪙・LA SUERTE DEL LUX",
		Description: fmt.Sprintf("La daga gira, corta el aire y cae con elegancia: **%s**.", resultado),
		Color:       env.Theme.ColorWarning,
	}
	env.Theme.BrandEmbed(embed)

	_, _ = discordutil.InteractionResponseEdit(s, i.Interaction, &discordgo.WebhookEdit{
		Content: stringPtr("✅ *La suerte está echada.* `[■■■■■■■■■■]`"),
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	})
	return nil
}

func stringPtr(s string) *string {
	return &s
}

func getSnowflakeTime(snowflake string) time.Time {
	var id int64
	fmt.Sscanf(snowflake, "%d", &id)
	timestamp := (id >> 22) + 1420070400000
	return time.Unix(0, timestamp*int64(time.Millisecond))
}
