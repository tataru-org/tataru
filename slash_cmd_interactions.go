package main

import (
	"fmt"
	"runtime/debug"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
)

func setRoleHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "set_role" {
		return
	}
	// check if a role ref exists
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		sendEventErrorResponse(event, err)
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	defer dbcon.Release()
	row := dbcon.QueryRow(ctx, `select count(*) > 0 from bot.role_ref`)
	var hasRoleID bool
	err = row.Scan(&hasRoleID)
	if err != nil {
		sendEventErrorResponse(event, err)
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	if hasRoleID {
		err = event.CreateMessage(
			discord.MessageCreate{
				Content: "A role is already set.",
				Flags:   discord.MessageFlagEphemeral,
			},
		)
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
	} else {
		// set role ref
		role := eventData.Role("role")
		_, err = dbcon.Exec(ctx, `insert into bot.role_ref(role_id) values($1)`, role.ID.String())
		if err != nil {
			sendEventErrorResponse(event, err)
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
		err = event.CreateMessage(
			discord.MessageCreate{
				Content: fmt.Sprintf("Role %s has been set.", role.Name),
				Flags:   discord.MessageFlagEphemeral,
			},
		)
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
		}
	}
}

func unsetRoleHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "unset_role" {
		return
	}
	// check if a role ref exists
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		sendEventErrorResponse(event, err)
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	defer dbcon.Release()
	row := dbcon.QueryRow(ctx, `select count(*) > 0 from bot.role_ref`)
	var hasRoleID bool
	err = row.Scan(&hasRoleID)
	if err != nil {
		sendEventErrorResponse(event, err)
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	if hasRoleID {
		// unset role ref
		_, err = dbcon.Exec(ctx, `truncate table bot.role_ref`)
		if err != nil {
			sendEventErrorResponse(event, err)
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
		err = event.CreateMessage(
			discord.MessageCreate{
				Content: "Role has been unset.",
				Flags:   discord.MessageFlagEphemeral,
			},
		)
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
	} else {
		err = event.CreateMessage(
			discord.MessageCreate{
				Content: "Unable to unset role; a role ref has not been set.",
				Flags:   discord.MessageFlagEphemeral,
			},
		)
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
	}
}

func forceSyncHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "force_sync" {
		return
	}
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	defer dbcon.Release()

	// get file id
	var fileID string
	row := dbcon.QueryRow(
		ctx,
		"select file_id from bot.file_ref",
	)
	err = row.Scan(&fileID)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	dbcon.Release()
	// get the discord members
	members, err := event.Client().Rest().GetMembers(*event.GuildID(), guildMemberCountRequestLimit, nullSnowflake)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	// sync the spreadsheet with the discord members
	err = syncRoleMembers(FileID(fileID), members)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	log.Debug("force sync successfully completed")
}
