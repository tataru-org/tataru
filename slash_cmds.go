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
			Name:                     "force_member_sync",
			Description:              "Force syncs the members in the mount spreadsheet with discord",
			DefaultMemberPermissions: &adminPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "sync_formatting",
			Description:              "Syncs the spreadsheet's formatting that is specified on disk",
			DefaultMemberPermissions: &adminPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "sync_file_perms",
			Description:              "Syncs the spreadsheet's file permissions that are specified on disk",
			DefaultMemberPermissions: &adminPerm,
		},
	}
}
