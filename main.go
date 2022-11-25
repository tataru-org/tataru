package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/log"
)

func onReadyHandler(event *events.Ready) {
	log.Debug("bot is ready")
}

func onGuildReady(event *events.GuildReady) {
	client := event.Client()
	_, err := client.Rest().GetMembers(event.GuildID)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	discordToken := ""
	log.SetLevel(log.LevelDebug)

	client, err := disgo.New(
		discordToken,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildMembers,
				gateway.IntentGuildMessages,
				gateway.IntentMessageContent,
			),
		),
		bot.WithEventListenerFunc(onReadyHandler),
		bot.WithEventListenerFunc(onGuildReady),
	)
	if err != nil {
		log.Fatal("error while building bot instance: ", err)
	}
	defer client.Close(context.TODO())

	if err = client.OpenGateway(context.TODO()); err != nil {
		log.Fatal("error while connecting to gateway: ", err)
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-s
}
