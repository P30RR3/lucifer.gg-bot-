package economy

import "krypton.dev/pkg/discordutil"
import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/bwmarrin/discordgo"

	"krypton.dev/internal/bot"
	"krypton.dev/internal/storage"
	"krypton.dev/internal/ui"
)

const maxBet int64 = 1_000_000

type Cog struct{}

func New() *Cog {
	return &Cog{}
}

func (c *Cog) Register(r *bot.Registry) error {
	return r.RegisterAll(
		&bot.Command{
			Name:        "saldo",
			Description: "Consulta tus fichas del club Lux.",
			Category:    "economia",
			GuildOnly:   true,
			Handler:     c.handleBalance,
		},
		&bot.Command{
			Name:        "fichas",
			Description: "Reclama las fichas diarias que Lucifer reserva para la casa.",
			Category:    "economia",
			GuildOnly:   true,
			Handler:     c.handleDailyChips,
		},
		&bot.Command{
			Name:        "ruleta",
			Description: "Apuesta fichas en la ruleta del Lux.",
			Category:    "economia",
			GuildOnly:   true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "apuesta",
					Description: "Fichas a apostar (1-1000000)",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "color",
					Description: "Color al que apuestas",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Rojo (paga 2x)", Value: "rojo"},
						{Name: "Negro (paga 2x)", Value: "negro"},
						{Name: "Verde (paga 36x)", Value: "verde"},
					},
				},
			},
			Handler: c.handleRoulette,
		},
		&bot.Command{
			Name:        "propina",
			Description: "Deja unas fichas en la caja del Lux.",
			Category:    "economia",
			GuildOnly:   true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "cantidad",
					Description: "Cantidad de fichas para la caja del Lux",
					Required:    true,
				},
			},
			Handler: c.handleTip,
		},
		&bot.Command{
			Name:        "tienda",
			Description: "Explora y compra artículos coleccionables del Lux.",
			Category:    "economia",
			GuildOnly:   true,
			Handler:     c.handleShop,
		},
		&bot.Command{
			Name:        "tienda-admin",
			Description: "Añade artículos o ajusta precios y rangos del servidor.",
			Category:    "economia",
			DefaultPerm: discordgo.PermissionManageGuild,
			GuildOnly:   true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type: discordgo.ApplicationCommandOptionSubCommand, Name: "precio",
					Description: "Cambia el precio de un artículo existente",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionString, Name: "articulo", Description: "Nombre exacto del artículo", Required: true, MaxLength: 100},
						{Type: discordgo.ApplicationCommandOptionInteger, Name: "monedas", Description: "Nuevo precio en monedas", Required: true, MinValue: float64Pointer(0), MaxValue: 10000000},
					},
				},
				{
					Type: discordgo.ApplicationCommandOptionSubCommand, Name: "articulo",
					Description: "Añade un artículo al catálogo",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionString, Name: "nombre", Description: "Nombre del artículo", Required: true, MaxLength: 100},
						{Type: discordgo.ApplicationCommandOptionString, Name: "descripcion", Description: "Descripción para la tienda", Required: true, MaxLength: 200},
						{Type: discordgo.ApplicationCommandOptionInteger, Name: "monedas", Description: "Precio en monedas", Required: true, MinValue: float64Pointer(0), MaxValue: 10000000},
					},
				},
				{
					Type: discordgo.ApplicationCommandOptionSubCommand, Name: "rango",
					Description: "Personaliza el título que se muestra desde un nivel",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionInteger, Name: "nivel", Description: "Nivel mínimo (1-100000)", Required: true, MinValue: float64Pointer(1), MaxValue: 100000},
						{Type: discordgo.ApplicationCommandOptionString, Name: "nombre", Description: "Nombre visible del rango", Required: true, MaxLength: 60},
					},
				},
			},
			Handler: c.handleShopAdmin,
		},
	)
}

func RegisterComponents(router *bot.Router) {
	router.RegisterComponent("shop", handleShopComponent)
}

func (c *Cog) handleBalance(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	balance, err := env.Store.EconomyBalance(ctx, i.GuildID, i.Member.User.ID)
	if err != nil {
		return fmt.Errorf("consultar saldo soberano: %w", err)
	}
	embed := &discordgo.MessageEmbed{
		Title:       "⚜️・TESORERÍA SOBERANA",
		Description: fmt.Sprintf("Registro de haberes del distinguido miembro <@%s>.", i.Member.User.ID),
		Color:       env.Theme.Primary,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "🪙 Fondos en Bóveda", Value: fmt.Sprintf("**%d monedas soberanas**", balance), Inline: true},
			{Name: "👑 Estipendio Diario", Value: "Reclamable vía `/fichas`", Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Lucifer Protocol • Soberanía y excelencia financiera"},
	}
	env.Theme.BrandEmbed(embed)
	return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *Cog) handleDailyChips(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	balance, err := env.Store.ClaimDailyChips(ctx, i.GuildID, i.Member.User.ID)
	if errors.Is(err, storage.ErrDailyCooldown) {
		// Plain text error (NO EMBED) as requested
		return respondPlainText(s, i, "⏳ **Estipendio diario ya reclamado:** Su asignación protocolar de 250 monedas para la presente jornada ya ha sido acreditada. Por favor, regrese en 24 horas.", true)
	}
	if err != nil {
		return fmt.Errorf("reclamar estipendio diario: %w", err)
	}
	// Plain text confirmation
	return respondPlainText(s, i, fmt.Sprintf("✨ **Estipendio soberano acreditado:** Se han transferido **250 monedas** a su tesorería personal. Su balance actual asciende a **%d monedas**.", balance), false)
}

func (c *Cog) handleRoulette(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	options := i.ApplicationCommandData().Options
	bet := options[0].IntValue()
	choice := options[1].StringValue()
	if bet < 1 || bet > maxBet {
		return respondPlainText(s, i, "⚠️ **Monto inválido:** La postura protocolar debe situarse estrictamente entre **1 y 1.000.000 de monedas**.", true)
	}

	spin, err := rand.Int(rand.Reader, big.NewInt(37))
	if err != nil {
		return fmt.Errorf("girar ruleta: %w", err)
	}
	number := int(spin.Int64())
	resultColor := rouletteColor(number)
	payout := roulettePayout(bet, choice, resultColor)

	balance, err := env.Store.PlayRoulette(ctx, i.GuildID, i.Member.User.ID, bet, payout)
	if errors.Is(err, storage.ErrInsufficientChips) {
		return respondPlainText(s, i, "💸 **Fondos insuficientes:** Su tesorería no cuenta con la cantidad requerida para respaldar esta postura. Reclame su estipendio con `/fichas`.", true)
	}
	if err != nil {
		return fmt.Errorf("resolver apuesta de ruleta: %w", err)
	}

	embedColor := env.Theme.ColorDanger
	title := "🎰・MESA DE RULETA SOBERANA"
	desc := fmt.Sprintf("La bola de marfil se ha detenido en el número **%d (%s)**.\n\n", number, strings.ToUpper(resultColor))
	if payout > 0 {
		embedColor = env.Theme.ColorSuccess
		desc += fmt.Sprintf("✨ **¡Victoria distinguida!** Su postura ha sido premiada con **%d monedas**. Su nuevo balance es de **%d monedas**.", payout, balance)
	} else {
		desc += fmt.Sprintf("⚜️ **La banca retiene la postura.** Agradecemos su distinguida participación. Balance actual: **%d monedas**.", balance)
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc,
		Color:       embedColor,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Lucifer Protocol • Alta Ruleta de Salón"},
	}
	env.Theme.BrandEmbed(embed)
	return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *Cog) handleTip(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	amount := i.ApplicationCommandData().Options[0].IntValue()
	if amount < 1 || amount > maxBet {
		return respond(s, i, "⚠️ La propina debe ser de entre **1 y 1.000.000 fichas**.", true)
	}

	balance, err := env.Store.TipMaze(ctx, i.GuildID, i.Member.User.ID, amount)
	if errors.Is(err, storage.ErrInsufficientChips) {
		return respond(s, i, "💸 No tienes suficientes fichas para esa propina.", true)
	}
	if err != nil {
		return fmt.Errorf("dejar fichas en la caja del Lux: %w", err)
	}
	return respond(s, i, fmt.Sprintf("🍸 Lucifer acepta tu aporte de **%d fichas**. Te quedan **%d** en la caja del Lux.", amount, balance), false)
}

func (c *Cog) handleShop(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	items, err := env.Store.ListShopItems(ctx, i.GuildID)
	if err != nil {
		return fmt.Errorf("consultar tienda del Lux: %w", err)
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🛍️・LA TIENDA DEL LUX",
		Description: "Elige un objeto del menú para comprarlo con tus monedas. La compra y el recibo son privados.",
		Color:       env.Theme.Primary,
	}
	options := make([]discordgo.SelectMenuOption, 0, len(items))
	for _, item := range items {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s · %d monedas", item.Name, item.Price),
			Value:  item.Description,
			Inline: true,
		})
		options = append(options, discordgo.SelectMenuOption{
			Label:       item.Name,
			Value:       item.Key,
			Description: fmt.Sprintf("%d monedas · %s", item.Price, truncateShopText(item.Description, 70)),
		})
	}
	if len(items) == 0 {
		return respond(s, i, "La tienda no tiene artículos disponibles todavía.", true)
	}
	env.Theme.BrandEmbed(embed)
	return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						MenuType:    discordgo.StringSelectMenu,
						CustomID:    "shop:buy",
						Placeholder: "✦ Elige un artículo ✦",
						Options:     options,
						MinValues:   intPointer(1),
						MaxValues:   1,
					},
				}},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) handleShopAdmin(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		return respond(s, i, "Elige precio, artículo o rango.", true)
	}
	subcommand := options[0]
	switch subcommand.Name {
	case "precio":
		name := subOptionString(subcommand.Options, "articulo")
		price := subOptionInt(subcommand.Options, "monedas")
		if err := env.Store.SetShopItemPrice(ctx, i.GuildID, name, price); err != nil {
			return fmt.Errorf("cambiar precio de tienda: %w", err)
		}
		return respond(s, i, fmt.Sprintf("✅ El precio de **%s** ahora es **%d monedas**.", name, price), true)
	case "articulo":
		name := subOptionString(subcommand.Options, "nombre")
		description := subOptionString(subcommand.Options, "descripcion")
		price := subOptionInt(subcommand.Options, "monedas")
		if err := env.Store.AddShopItem(ctx, i.GuildID, name, description, price); err != nil {
			return fmt.Errorf("añadir artículo de tienda: %w", err)
		}
		return respond(s, i, fmt.Sprintf("✅ **%s** ya está en la tienda por **%d monedas**.", name, price), true)
	case "rango":
		level := subOptionInt(subcommand.Options, "nivel")
		name := strings.TrimSpace(subOptionString(subcommand.Options, "nombre"))
		if err := env.Store.SetLuxRank(ctx, i.GuildID, level, name); err != nil {
			return fmt.Errorf("personalizar rango Lux: %w", err)
		}
		return respond(s, i, fmt.Sprintf("✅ Desde el nivel **%d**, el rango se llamará **%s**.", level, name), true)
	default:
		return respond(s, i, "Esa acción de administración no existe.", true)
	}
}

func handleShopComponent(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, env *bot.Env) error {
	data := i.MessageComponentData()
	if len(data.Values) != 1 {
		return respond(s, i, "Elige un solo artículo.", true)
	}
	if i.Member == nil || i.Member.User == nil {
		return respond(s, i, "Esta compra solo está disponible dentro del servidor.", true)
	}
	balance, err := env.Store.BuyShopItem(ctx, i.GuildID, i.Member.User.ID, data.Values[0])
	if errors.Is(err, storage.ErrInsufficientChips) {
		return respond(s, i, "No tienes suficientes monedas para comprar ese artículo.", true)
	}
	if err != nil {
		return fmt.Errorf("comprar artículo del Lux: %w", err)
	}
	return respond(s, i, fmt.Sprintf("✅・Compra completada. El artículo ya está en tu inventario.\n🪙 Saldo restante: **%d monedas**.", balance), true)
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) error {
	flags := discordgo.MessageFlags(0)
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: ui.FrameMessage(content),
			Flags:   flags,
		},
	})
}

func respondPlainText(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) error {
	flags := discordgo.MessageFlags(0)
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	return discordutil.InteractionRespond(s, i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   flags,
		},
	})
}

func rouletteColor(number int) string {
	if number == 0 {
		return "verde"
	}
	switch number {
	case 1, 3, 5, 7, 9, 12, 14, 16, 18, 19, 21, 23, 25, 27, 30, 32, 34, 36:
		return "rojo"
	default:
		return "negro"
	}
}

func roulettePayout(bet int64, choice, resultColor string) int64 {
	if choice != resultColor {
		return 0
	}
	if choice == "verde" {
		return bet * 36
	}
	return bet * 2
}

func subOptionString(options []*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	for _, option := range options {
		if option.Name == name {
			return option.StringValue()
		}
	}
	return ""
}

func subOptionInt(options []*discordgo.ApplicationCommandInteractionDataOption, name string) int64 {
	for _, option := range options {
		if option.Name == name {
			return option.IntValue()
		}
	}
	return 0
}

func truncateShopText(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

func float64Pointer(value float64) *float64 {
	return &value
}

func intPointer(value int) *int {
	return &value
}
