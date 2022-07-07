package main

import (
	"github.com/disgo/core/bot"
	"github.com/disgo/core/types"
	"log"
	"os"
)

func main() {
	b := bot.New(33283)
	b.OnMessage(OnMessage)
	b.OnReady(OnReady)
	b.OnInteraction(OnInteraction)
	b.Run(os.Getenv("DISCORD_TOKEN"))
}

func OnMessage(message *types.Message) {
	log.Println(message.Content)
}
func OnReady() {
	log.Println("[-------- READY --------]")
}
func OnInteraction(interaction *types.Interaction) {
	log.Println(interaction)
}
