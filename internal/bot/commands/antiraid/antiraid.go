package antiraid

import (
	"krypton.dev/internal/antiraid"
	"krypton.dev/internal/bot"
)

type Cog struct {
	engine *antiraid.Engine
}

func New(engine *antiraid.Engine) *Cog {
	return &Cog{engine: engine}
}

// Register se deja vacío: Cero comandos slash, cero prefijos
func (c *Cog) Register(r *bot.Registry) error {
	return nil
}
