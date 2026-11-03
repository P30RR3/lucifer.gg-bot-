package bot

import (
	"github.com/bwmarrin/discordgo"

	"krypton.dev/pkg/discordutil"
)

func RespondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return discordutil.RespondError(s, i, message)
}

func RespondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	return discordutil.RespondEphemeral(s, i, message)
}
