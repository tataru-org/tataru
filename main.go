package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/log"
	"golang.org/x/oauth2/google"
)

const (
	gscope       = "https://www.googleapis.com/auth/drive"
	fileMimeType = "application/vnd.google-apps.spreadsheet"
)

var (
	gclient *http.Client
	ctx     context.Context
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

	ctx = context.Background()
	b, err := os.ReadFile("svc-creds.json")
	if err != nil {
		panic(err)
	}
	gconfig, err := google.JWTConfigFromJSON(b, gscope)
	if err != nil {
		panic(err)
	}
	gclient = gconfig.Client(ctx)

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
		panic(err)
	}
	defer client.Close(context.TODO())

	if err = client.OpenGateway(context.TODO()); err != nil {
		panic(err)
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-s
}
