package bot

import (
	"github.com/disgo/client"
	"github.com/disgo/types"
)

type Bot struct {
	intent int
	core   *client.Client
}

func New(intent int) *Bot {
	c := client.New(intent)
	return &Bot{intent: intent, core: c}
}

func (bot *Bot) Run(token string) {
	bot.core.Run(token)
}

func (bot *Bot) OnMessage(handler func(message *types.Message)) {
	bot.core.AddHandler("MESSAGE_CREATE", handler)
}

func (bot *Bot) OnReady(handler func()) {
	bot.core.AddHandler("READY", handler)
}
