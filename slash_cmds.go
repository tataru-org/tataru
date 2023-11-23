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
			Description:              "Set the mount farm role to watch for discord member updates",
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
			Description:              "Unset the mount farm role to watch for discord member updates",
			DefaultMemberPermissions: &adminPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "spreadsheet_discord_member_sync",
			Description:              "Syncs the spreadsheet with discord member data",
			DefaultMemberPermissions: &adminPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "sync_spreadsheet_styling",
			Description:              "Syncs the spreadsheet styling with the stored styling data",
			DefaultMemberPermissions: &adminPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "sync_file_perms",
			Description:              "Syncs the spreadsheet file permissions with the stored file permissions data",
			DefaultMemberPermissions: &adminPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "any_xiv_char_search",
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
			Name:        "xiv_char_search",
			Description: "Attempts to search for the user's FF14 character's ID by the given character name and save it",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "xiv_character_name",
					Description: "The entire name of a FF14 character",
					Required:    true,
				},
			},
		},
		discord.SlashCommandCreate{
			Name:                     "map_any_xiv_char_id",
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
			Name:        "map_xiv_char_id",
			Description: "Maps a FF14 character's ID to the discord user that used the command",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "xiv_character_id",
					Description: "The FF14 character's ID",
					Required:    true,
				},
			},
		},
		discord.SlashCommandCreate{
			Name:                     "scan_xiv_mounts",
			Description:              "Scans XIVAPI for mounts",
			DefaultMemberPermissions: &adminPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "update_member_names",
			Description:              "Updates member names",
			DefaultMemberPermissions: &adminPerm,
		},
	}
}
