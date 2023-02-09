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
		discord.SlashCommandCreate{
			Name:                     "try_xiv_char_search",
			Description:              "Attempts to search for a FF14 character's ID by the given character name and save it",
			DefaultMemberPermissions: &adminPerm,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionUser{
					Name:        "discord_user",
					Description: "The discord user associated with the FF14 character",
					Required:    true,
				},
				discord.ApplicationCommandOptionString{
					Name:        "xiv_character_name",
					Description: "The entire name of a FF14 character",
					Required:    true,
				},
			},
		},
		discord.SlashCommandCreate{
			Name:                     "map_xiv_char_id",
			Description:              "Maps a FF14 character's ID to the given discord user",
			DefaultMemberPermissions: &adminPerm,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionUser{
					Name:        "discord_user",
					Description: "The discord user associated with the FF14 character",
					Required:    true,
				},
				discord.ApplicationCommandOptionString{
					Name:        "xiv_character_id",
					Description: "The FF14 character's ID",
					Required:    true,
				},
			},
		},
		discord.SlashCommandCreate{
			Name:                     "force_scan_xiv_mounts",
			Description:              "Force scans XIVAPI for mounts",
			DefaultMemberPermissions: &adminPerm,
		},
	}
}
