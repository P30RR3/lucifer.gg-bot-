package ui

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

type ButtonSpec struct {
	CustomID string
	Label    string
	Emoji    string
	Style    discordgo.ButtonStyle
	Disabled bool
	URL      string
}

func BuildButtons(specs ...ButtonSpec) []discordgo.MessageComponent {
	rows := make([]discordgo.MessageComponent, 0, 5)
	row := make([]discordgo.MessageComponent, 0, 5)

	for _, s := range specs {
		b := discordgo.Button{
			CustomID: s.CustomID,
			Label:    s.Label,
			Style:    s.Style,
			Disabled: s.Disabled,
		}
		if s.Emoji != "" {
			b.Emoji = &discordgo.ComponentEmoji{Name: s.Emoji}
		}
		if s.Style == discordgo.LinkButton {
			b.URL = s.URL
			b.CustomID = ""
		}
		row = append(row, b)
		if len(row) == 5 {
			rows = append(rows, discordgo.ActionsRow{Components: row})
			row = make([]discordgo.MessageComponent, 0, 5)
		}
	}
	if len(row) > 0 {
		rows = append(rows, discordgo.ActionsRow{Components: row})
	}
	return rows
}

func DangerConfirm(confirmID, cancelID, subject string) []discordgo.MessageComponent {
	return BuildButtons(
		ButtonSpec{
			CustomID: confirmID,
			Label:    "Confirmar " + subject,
			Emoji:    "??",
			Style:    discordgo.DangerButton,
		},
		ButtonSpec{
			CustomID: cancelID,
			Label:    "Cancelar",
			Emoji:    "??",
			Style:    discordgo.SecondaryButton,
		},
	)
}

func SuccessConfirm(confirmID, cancelID, subject string) []discordgo.MessageComponent {
	return BuildButtons(
		ButtonSpec{
			CustomID: confirmID,
			Label:    "Aplicar " + subject,
			Emoji:    "?",
			Style:    discordgo.SuccessButton,
		},
		ButtonSpec{
			CustomID: cancelID,
			Label:    "Cancelar",
			Emoji:    "??",
			Style:    discordgo.SecondaryButton,
		},
	)
}

func NavButtons(prevID, closeID, nextID string, disablePrev, disableNext bool) []discordgo.MessageComponent {
	return BuildButtons(
		ButtonSpec{CustomID: prevID, Label: "Anterior", Emoji: "?", Style: discordgo.SecondaryButton, Disabled: disablePrev},
		ButtonSpec{CustomID: closeID, Label: "Cerrar", Emoji: "?", Style: discordgo.DangerButton},
		ButtonSpec{CustomID: nextID, Label: "Siguiente", Emoji: "?", Style: discordgo.PrimaryButton, Disabled: disableNext},
	)
}

func PaginationControls(prefix string, current, total int) []discordgo.MessageComponent {
	return BuildButtons(
		ButtonSpec{CustomID: prefix + ":first", Label: "", Emoji: "?", Style: discordgo.SecondaryButton, Disabled: current <= 1},
		ButtonSpec{CustomID: prefix + ":prev", Label: "", Emoji: "?", Style: discordgo.SecondaryButton, Disabled: current <= 1},
		ButtonSpec{CustomID: prefix + ":close", Label: "Cerrar", Emoji: "?", Style: discordgo.DangerButton},
		ButtonSpec{CustomID: prefix + ":next", Label: "", Emoji: "?", Style: discordgo.PrimaryButton, Disabled: current >= total},
		ButtonSpec{CustomID: prefix + ":last", Label: "", Emoji: "?", Style: discordgo.SecondaryButton, Disabled: current >= total},
	)
}

func ActionButtons(prefix, userID string) []discordgo.MessageComponent {
	return BuildButtons(
		ButtonSpec{CustomID: prefix + ":warn:" + userID, Label: "Warn", Emoji: "??", Style: discordgo.SecondaryButton},
		ButtonSpec{CustomID: prefix + ":timeout:" + userID, Label: "Timeout", Emoji: "?", Style: discordgo.PrimaryButton},
		ButtonSpec{CustomID: prefix + ":kick:" + userID, Label: "Kick", Emoji: "??", Style: discordgo.DangerButton},
		ButtonSpec{CustomID: prefix + ":ban:" + userID, Label: "Ban", Emoji: "??", Style: discordgo.DangerButton},
	)
}

type SelectOption struct {
	Label       string
	Value       string
	Description string
	Emoji       string
	Default     bool
}

func BuildSelect(customID, placeholder string, options []SelectOption, maxValues int) []discordgo.MessageComponent {
	opts := make([]discordgo.SelectMenuOption, 0, len(options))
	for _, o := range options {
		opt := discordgo.SelectMenuOption{
			Label:       o.Label,
			Value:       o.Value,
			Description: o.Description,
			Default:     o.Default,
		}
		if o.Emoji != "" {
			opt.Emoji = &discordgo.ComponentEmoji{Name: o.Emoji}
		}
		opts = append(opts, opt)
	}
	if maxValues <= 0 {
		maxValues = 1
	}
	minValues := 1
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    customID,
					Placeholder: placeholder,
					Options:     opts,
					MaxValues:   maxValues,
					MinValues:   &minValues,
				},
			},
		},
	}
}

func BuildMultiSelect(customID, placeholder string, options []SelectOption, min, max int) []discordgo.MessageComponent {
	if min < 0 {
		min = 0
	}
	if max <= 0 || max > 25 {
		max = 25
	}
	opts := make([]discordgo.SelectMenuOption, 0, len(options))
	for _, o := range options {
		opt := discordgo.SelectMenuOption{
			Label:       o.Label,
			Value:       o.Value,
			Description: o.Description,
			Default:     o.Default,
		}
		if o.Emoji != "" {
			opt.Emoji = &discordgo.ComponentEmoji{Name: o.Emoji}
		}
		opts = append(opts, opt)
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    customID,
					Placeholder: placeholder,
					Options:     opts,
					MinValues:   &min,
					MaxValues:   max,
				},
			},
		},
	}
}

func ConfirmRow(components []discordgo.MessageComponent) discordgo.ActionsRow {
	return discordgo.ActionsRow{Components: components}
}

func TimestampRelative(t time.Time) string {
	return fmt.Sprintf("<t:%d:R>", t.Unix())
}

func TimestampFull(t time.Time) string {
	return fmt.Sprintf("<t:%d:F>", t.Unix())
}
