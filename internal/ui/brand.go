package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/config"
)

var defaultBrand = NewTheme(config.UIConfig{})

func BrandMessage(content string) string {
	return FrameMessage(content)
}

func BrandEmbed(embed *discordgo.MessageEmbed) {
	defaultBrand.BrandEmbed(embed)
}

func BrandModalTitle(title string) string {
	return brandLabel(title, 43)
}

func BrandEmbeds(embeds []*discordgo.MessageEmbed) {
	for _, embed := range embeds {
		BrandEmbed(embed)
	}
}

func BrandMessageSend(message *discordgo.MessageSend) {
	if message == nil {
		return
	}
	message.Content = BrandMessage(message.Content)
	BrandEmbed(message.Embed)
	BrandEmbeds(message.Embeds)
	BrandComponents(message.Components)
}

func BrandMessageEdit(edit *discordgo.MessageEdit) {
	if edit == nil {
		return
	}
	if edit.Content != nil {
		content := BrandMessage(*edit.Content)
		edit.Content = &content
	}
	BrandEmbed(edit.Embed)
	if edit.Embeds != nil {
		embeds := append([]*discordgo.MessageEmbed(nil), (*edit.Embeds)...)
		BrandEmbeds(embeds)
		edit.Embeds = &embeds
	}
	if edit.Components != nil {
		components := append([]discordgo.MessageComponent(nil), (*edit.Components)...)
		BrandComponents(components)
		edit.Components = &components
	}
}

func BrandComponents(components []discordgo.MessageComponent) {
	for i, component := range components {
		switch value := component.(type) {
		case discordgo.ActionsRow:
			value.Components = append([]discordgo.MessageComponent(nil), value.Components...)
			BrandComponents(value.Components)
			components[i] = value
		case *discordgo.ActionsRow:
			if value != nil {
				row := *value
				row.Components = append([]discordgo.MessageComponent(nil), row.Components...)
				BrandComponents(row.Components)
				components[i] = &row
			}
		case discordgo.Button:
			value.Label = brandLabel(value.Label, 77)
			components[i] = value
		case *discordgo.Button:
			if value != nil {
				button := *value
				button.Label = brandLabel(button.Label, 77)
				components[i] = &button
			}
		case discordgo.SelectMenu:
			value.Placeholder = brandLabel(value.Placeholder, 147)
			for j := range value.Options {
				value.Options[j].Label = brandLabel(value.Options[j].Label, 97)
				value.Options[j].Description = brandLabel(value.Options[j].Description, 97)
			}
			components[i] = value
		case *discordgo.SelectMenu:
			if value != nil {
				menu := *value
				menu.Placeholder = brandLabel(menu.Placeholder, 147)
				menu.Options = append([]discordgo.SelectMenuOption(nil), menu.Options...)
				for j := range menu.Options {
					menu.Options[j].Label = brandLabel(menu.Options[j].Label, 97)
					menu.Options[j].Description = brandLabel(menu.Options[j].Description, 97)
				}
				components[i] = &menu
			}
		case discordgo.TextInput:
			value.Label = brandLabel(value.Label, 42)
			value.Placeholder = brandLabel(value.Placeholder, 97)
			components[i] = value
		case *discordgo.TextInput:
			if value != nil {
				input := *value
				input.Label = brandLabel(input.Label, 42)
				input.Placeholder = brandLabel(input.Placeholder, 97)
				components[i] = &input
			}
		}
	}
}

func brandLabel(label string, maxRunes int) string {
	if strings.TrimSpace(label) == "" || strings.HasPrefix(label, "⚜️ ") || strings.HasPrefix(label, "✦ ") || utf8.RuneCountInString(label) > maxRunes {
		return label
	}
	return "⚜️ " + label
}
