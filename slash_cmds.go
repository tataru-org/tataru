package main

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/json"
)

func createSlashCommands() []discord.ApplicationCommandCreate {
	adminPerm := json.NewNullable(discord.PermissionAdministrator)
	return []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:                     "set_role",
			Description:              "Set the role to watch for the mount spreadsheet",
			DefaultMemberPermissions: &adminPerm,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionRole{
					Name:        "role",
					Required:    true,
					Description: "the role to set",
				},
			},
		},
		discord.SlashCommandCreate{
			Name:                     "unset_role",
			Description:              "Unset the role to watch for the mount spreadsheet",
			DefaultMemberPermissions: &adminPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "force_sync",
			Description:              "Force sync the mount spreadsheet with discord",
			DefaultMemberPermissions: &adminPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "sync_formatting",
			Description:              "Syncs the spreadsheet formatting with the formatting file that is on disk",
			DefaultMemberPermissions: &adminPerm,
		},
	}
}
