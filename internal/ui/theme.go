package ui

import (
	"image/color"
	"strconv"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/config"
)

type Theme struct {
	Primary      int
	ColorSuccess int
	ColorWarning int
	ColorDanger  int
	ColorNeutral int

	FooterIcon string
	FooterText string
	AuthorName string
	AuthorIcon string

	ProgressWidth int
	LiveFPS       int
	LiveDebounce  int

	mu    sync.RWMutex
	glyph GlyphSet
}

func NewTheme(cfg config.UIConfig) *Theme {
	t := &Theme{
		Primary:      0x4A000A, // Imperial Crimson
		ColorSuccess: 0x1A472A, // Emerald Royale
		ColorWarning: 0x8C6D1F, // Antique Gold
		ColorDanger:  0x780A18, // Deep Scarlet
		ColorNeutral: 0x1A1418, // Obsidian Onyx

		FooterIcon: "https://cdn-icons-png.flaticon.com/512/10691/10691880.png",
		FooterText: "Lucifer Protocol • Soberanía, diplomacia y distinción absoluta ⚜️",
		AuthorName: "Lucifer • Sistema de Alta Distinción",
		AuthorIcon: "https://cdn-icons-png.flaticon.com/512/10691/10691880.png",

		ProgressWidth: cfg.ProgressWidth,
		LiveFPS:       cfg.LiveFPS,
		LiveDebounce:  1000 / max(cfg.LiveFPS, 1),
		glyph:         GlyphsBlock,
	}
	if t.ProgressWidth < 5 {
		t.ProgressWidth = 18
	}
	if t.ProgressWidth > 60 {
		t.ProgressWidth = 60
	}
	return t
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (t *Theme) Color(name string) int {
	switch strings.ToLower(name) {
	case "success":
		return t.ColorSuccess
	case "warning":
		return t.ColorWarning
	case "danger":
		return t.ColorDanger
	case "neutral":
		return t.ColorNeutral
	default:
		return t.Primary
	}
}

func (t *Theme) ColorRGBA(name string) color.RGBA {
	c := t.Color(name)
	return color.RGBA{
		R: uint8((c >> 16) & 0xFF),
		G: uint8((c >> 8) & 0xFF),
		B: uint8(c & 0xFF),
		A: 0xFF,
	}
}

func (t *Theme) ColorHex(name string) string {
	return "#" + strings.ToUpper(strconv.FormatInt(int64(t.Color(name)), 16))
}

func (t *Theme) BrandEmbed(embed *discordgo.MessageEmbed) {
	if embed == nil {
		return
	}
	if embed.Color == 0 {
		embed.Color = t.Primary
	}
	if embed.Title != "" && !strings.HasPrefix(embed.Title, "⚜️ ") && !strings.HasPrefix(embed.Title, "✦ ") {
		embed.Title = "⚜️ " + embed.Title
	}
	if embed.Author == nil {
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name:    t.AuthorName,
			IconURL: t.AuthorIcon,
		}
	}
	if embed.Footer == nil {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text:    t.FooterText,
			IconURL: t.FooterIcon,
		}
	}
}

// ==========================================
// 🔴 EMBEDS QUE PEDÍA EL BOT
// ==========================================

func (t *Theme) ErrorEmbed(args ...string) *discordgo.MessageEmbed {
	title := "Acción Interrumpida"
	desc := ""
	if len(args) == 1 {
		desc = args[0]
	} else if len(args) >= 2 {
		title = args[0]
		desc = args[1]
	}
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc,
		Color:       t.ColorDanger,
	}
	t.BrandEmbed(embed)
	return embed
}

func (t *Theme) SuccessEmbed(args ...string) *discordgo.MessageEmbed {
	title := "Operación Concluida"
	desc := ""
	if len(args) == 1 {
		desc = args[0]
	} else if len(args) >= 2 {
		title = args[0]
		desc = args[1]
	}
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc,
		Color:       t.ColorSuccess,
	}
	t.BrandEmbed(embed)
	return embed
}

func (t *Theme) WarningEmbed(args ...string) *discordgo.MessageEmbed {
	title := "Advertencia de Protocolo"
	desc := ""
	if len(args) == 1 {
		desc = args[0]
	} else if len(args) >= 2 {
		title = args[0]
		desc = args[1]
	}
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc,
		Color:       t.ColorWarning,
	}
	t.BrandEmbed(embed)
	return embed
}

// ==========================================
// FORMATOS DE TEXTO PLANO
// ==========================================

func FormatErrorMessage(title, desc string) string {
	if desc == "" {
		desc = title
		title = "Aviso Protocolar"
	}
	return "⚠️ **" + title + "**\n> " + desc + "\n*Por favor, verifique su solicitud o acuda a la cancillería de asistencia.*"
}

func (t *Theme) ErrorMessage(args ...string) string {
	title := "Acción Interrumpida"
	desc := ""
	if len(args) == 1 {
		desc = args[0]
	} else if len(args) >= 2 {
		title = args[0]
		desc = args[1]
	}
	return FormatErrorMessage(title, desc)
}

func (t *Theme) WarningMessage(args ...string) string {
	title := "Advertencia de Protocolo"
	desc := ""
	if len(args) == 1 {
		desc = args[0]
	} else if len(args) >= 2 {
		title = args[0]
		desc = args[1]
	}
	return "⚠️ **" + title + "**\n> " + desc
}

func (t *Theme) SuccessMessage(args ...string) string {
	title := "Operación Concluida"
	desc := ""
	if len(args) == 1 {
		desc = args[0]
	} else if len(args) >= 2 {
		title = args[0]
		desc = args[1]
	}
	return "✨ **" + title + "**\n> " + desc
}

func FrameMessage(content string) string {
	if content == "" || strings.Contains(content, "LUCIFER • PROTOCOLO SOBERANO") {
		return content
	}
	return "╭────── ⚜️ LUCIFER • PROTOCOLO SOBERANO ──────╮\n" + content + "\n╰────── Con distinción y respeto absoluto. 🥂 ──────╯"
}