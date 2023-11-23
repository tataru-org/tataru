package main

import (
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

func setRoleHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "set_role" {
		return
	}

	err := event.DeferCreateMessage(true)
	if err != nil {
		log.Error(err)
		return
	}
	// check if a role ref exists
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	defer dbcon.Release()
	row := dbcon.QueryRow(ctx, `select count(*) > 0 from bot.role_ref`)
	var hasRoleID bool
	err = row.Scan(&hasRoleID)
	if err != nil {

		log.Error(err)
		return
	}
	if hasRoleID {
		content := "A role is already set."
		_, err = event.Client().Rest().UpdateInteractionResponse(
			event.ApplicationID(),
			event.Token(),
			discord.MessageUpdate{
				Content: &content,
			},
		)
		if err != nil {
			log.Error(err)
			return
		}
	} else {
		// set role ref
		role := eventData.Role("role")
		_, err = dbcon.Exec(ctx, `insert into bot.role_ref(role_id) values($1)`, role.ID.String())
		if err != nil {
			log.Error(err)
			return
		}
		content := fmt.Sprintf("Role %s has been set.", role.Name)
		_, err = event.Client().Rest().UpdateInteractionResponse(
			event.ApplicationID(),
			event.Token(),
			discord.MessageUpdate{
				Content: &content,
			},
		)
		if err != nil {
			log.Error(err)
		}
	}
}

func unsetRoleHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "unset_role" {
		return
	}

	err := event.DeferCreateMessage(true)
	if err != nil {
		log.Error(err)
		return
	}
	// check if a role ref exists
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	defer dbcon.Release()
	row := dbcon.QueryRow(ctx, `select count(*) > 0 from bot.role_ref`)
	var hasRoleID bool
	err = row.Scan(&hasRoleID)
	if err != nil {
		log.Error(err)
		return
	}
	if hasRoleID {
		// unset role ref
		_, err = dbcon.Exec(ctx, `truncate table bot.role_ref`)
		if err != nil {
			log.Error(err)
			return
		}
		content := "Role has been unset."
		_, err = event.Client().Rest().UpdateInteractionResponse(
			event.ApplicationID(),
			event.Token(),
			discord.MessageUpdate{
				Content: &content,
			},
		)
		if err != nil {
			log.Error(err)
			return
		}
	} else {
		content := "Unable to unset role; a role ref has not been set"
		_, err = event.Client().Rest().UpdateInteractionResponse(
			event.ApplicationID(),
			event.Token(),
			discord.MessageUpdate{
				Content: &content,
			},
		)
		if err != nil {
			log.Error(err)
			return
		}
	}
}

func spreadsheetDiscordMemberSyncHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "spreadsheet_discord_member_sync" {
		return
	}

	err := event.DeferCreateMessage(true)
	if err != nil {
		log.Error(err)
		return
	}
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	defer dbcon.Release()

	// get file id
	var fileID string
	row := dbcon.QueryRow(
		ctx,
		"select file_gcp_id from bot.file_ref",
	)
	err = row.Scan(&fileID)
	if err != nil {
		log.Error(err)
		return
	}
	dbcon.Release()
	// get the discord members
	members, err := event.Client().Rest().GetMembers(*event.GuildID(), guildMemberCountRequestLimit, nullSnowflake)
	if err != nil {
		log.Error(err)
		return
	}
	// sync the spreadsheet with the discord members
	err = syncRoleMembers(FileID(fileID), members)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug("force member sync successfully completed")
	content := "Force member sync successfully completed"
	_, err = event.Client().Rest().UpdateInteractionResponse(
		event.ApplicationID(),
		event.Token(),
		discord.MessageUpdate{
			Content: &content,
		},
	)
	if err != nil {
		log.Error(err)
		return
	}
}

func syncSpreadsheetStylingHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "sync_spreadsheet_styling" {
		return
	}

	err := event.DeferCreateMessage(true)
	if err != nil {
		log.Error(err)
		return
	}
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	defer dbcon.Release()
	// get file id
	var fileID *string
	row := dbcon.QueryRow(
		ctx,
		"select file_gcp_id from bot.file_ref",
	)
	err = row.Scan(&fileID)
	if err != nil {
		log.Error(err)
		return
	}
	dbcon.Release()
	if fileID == nil {
		log.Error(err)
		return
	}

	columnMap, err := NewColumnMap()
	if err != nil {
		log.Error(err)
		return
	}
	spreadsheet, err := gsheetsSvc.Spreadsheets.Get(*fileID).IncludeGridData(true).Do()
	if err != nil {
		log.Error(err)
		return
	}

	// make requests for the header rows
	requests := make([]*sheets.Request, len(spreadsheet.Sheets)*2)
	for i := 0; i < len(spreadsheet.Sheets); i++ {
		sheet := spreadsheet.Sheets[i]
		row := sheet.Data[0].RowData[0]
		vals := make([]*sheets.CellData, len(row.Values))
		for k := 0; k < len(row.Values); k++ {
			colName := string(columnMap.Mapping[SheetMetadata{
				ID:    SheetID(sheet.Properties.SheetId),
				Index: SheetIndex(sheet.Properties.Index),
			}][ColumnIndex(k)].Name)
			vals[k] = &sheets.CellData{
				UserEnteredFormat: columnMap.Mapping[SheetMetadata{
					ID:    SheetID(sheet.Properties.SheetId),
					Index: SheetIndex(sheet.Properties.Index),
				}][ColumnIndex(k)].HeaderFormat,
				UserEnteredValue: &sheets.ExtendedValue{
					StringValue: &colName,
				},
			}
		}
		requests[i] = &sheets.Request{
			UpdateCells: &sheets.UpdateCellsRequest{
				Fields: "userEnteredFormat,userEnteredValue",
				Rows: []*sheets.RowData{
					{
						Values: vals,
					},
				},
				Start: &sheets.GridCoordinate{
					ColumnIndex: 0,
					RowIndex:    0,
					SheetId:     sheet.Properties.SheetId,
				},
			},
		}
	}
	// make requests for the cell data
	for i := 0; i < len(spreadsheet.Sheets); i++ {
		sheet := spreadsheet.Sheets[i]
		rowData := make([]*sheets.RowData, len(sheet.Data[0].RowData))
		for j := 1; j < len(sheet.Data[0].RowData); j++ {
			row := sheet.Data[0].RowData[j]
			vals := make([]*sheets.CellData, len(row.Values))
			for k := 0; k < len(row.Values); k++ {
				vals[k] = &sheets.CellData{
					UserEnteredFormat: columnMap.Mapping[SheetMetadata{
						ID:    SheetID(sheet.Properties.SheetId),
						Index: SheetIndex(sheet.Properties.Index),
					}][ColumnIndex(k)].ColumnFormat,
				}
			}
			rowData[j] = &sheets.RowData{
				Values: vals,
			}
		}
		requests[i+len(spreadsheet.Sheets)-1] = &sheets.Request{
			UpdateCells: &sheets.UpdateCellsRequest{
				Fields: "userEnteredFormat",
				Rows:   rowData,
				Start: &sheets.GridCoordinate{
					ColumnIndex: 0,
					RowIndex:    1,
					SheetId:     sheet.Properties.SheetId,
				},
			},
		}
	}
	googleSheetsWriteReqs <- &SheetBatchUpdate{
		ID: spreadsheet.SpreadsheetId,
		Batch: &sheets.BatchUpdateSpreadsheetRequest{
			Requests: requests,
		},
	}
	log.Debug("formatting successfully synced")
	content := "Formatting successfully synced"
	_, err = event.Client().Rest().UpdateInteractionResponse(
		event.ApplicationID(),
		event.Token(),
		discord.MessageUpdate{
			Content: &content,
		},
	)
	if err != nil {
		log.Error(err)
		return
	}
}

func syncFilePermsHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "sync_file_perms" {
		return
	}

	err := event.DeferCreateMessage(true)
	if err != nil {
		log.Error(err)
		return
	}
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	defer dbcon.Release()
	// get file id
	var fileID *string
	row := dbcon.QueryRow(
		ctx,
		"select file_gcp_id from bot.file_ref",
	)
	err = row.Scan(&fileID)
	if err != nil {
		log.Error(err)
		return
	}
	if fileID == nil {
		log.Error(err)
		return
	}
	// get perms from db
	rows, err := dbcon.Query(ctx, `select perm_gcp_id, email, role, role_type from bot.permissions`)
	if err != nil {
		log.Error(err)
		return
	}
	dbPerms := map[string]*drive.Permission{}
	for rows.Next() {
		var id string
		var email string
		var role string
		var roleType string
		err = rows.Scan(&id, &email, &role, &roleType)
		if err != nil {
			log.Error(err)
			return
		}
		dbPerms[id] = &drive.Permission{
			EmailAddress: email,
			Role:         role,
			Type:         roleType,
		}
	}

	// get perms from the perm file
	permsOnDisk, err := GetPermissions(mountSpreadsheetPermissionsFilepath)
	if err != nil {
		log.Error(err)
		return
	}
	// get perms from
	// determine perms that need to be added
	permsToAdd := []*drive.Permission{}
	for i := 0; i < len(permsOnDisk); i++ {
		alreadyExists := false
		for _, dbPerm := range dbPerms {
			if permsOnDisk[i].Type == dbPerm.Type && permsOnDisk[i].Type == "anyone" {
				// the anyone permission
				alreadyExists = true
				break
			} else if strings.EqualFold(permsOnDisk[i].EmailAddress, dbPerm.EmailAddress) {
				// all other permissions
				alreadyExists = true
				break
			}
		}
		if !alreadyExists {
			permsToAdd = append(permsToAdd, permsOnDisk[i])
			log.Debugf(
				"file permission queued to be created (email:%s,type:%s,role:%s)",
				permsOnDisk[i].EmailAddress,
				permsOnDisk[i].Type,
				permsOnDisk[i].Role,
			)
		}
	}
	// determine perms that need to be updated
	permsToUpdate := map[string]*drive.Permission{}
	for i := 0; i < len(permsOnDisk); i++ {
		shouldBeUpdated := false
		var id string
		for dbPermID, dbPerm := range dbPerms {
			if permsOnDisk[i].Type == dbPerm.Type && permsOnDisk[i].Type == "anyone" {
				// the anyone permission
				if permsOnDisk[i].Role != dbPerm.Role {
					shouldBeUpdated = true
					id = dbPermID
					break
				}
			} else if strings.EqualFold(permsOnDisk[i].EmailAddress, dbPerm.EmailAddress) {
				// all other permissions
				if permsOnDisk[i].Role != dbPerm.Role {
					shouldBeUpdated = true
					id = dbPermID
					break
				}
			}
		}
		if shouldBeUpdated {
			permsToUpdate[id] = &drive.Permission{
				Role: permsOnDisk[i].Role,
			}
			log.Debugf(
				"file permission queued to be updated (email:%s,type:%s,role:%s)",
				permsOnDisk[i].EmailAddress,
				permsOnDisk[i].Type,
				permsOnDisk[i].Role,
			)
		}
	}
	// determine perms that need to be deleted
	permIDsToDelete := []string{}
	for dbPermID, dbPerm := range dbPerms {
		isNotInPermsOnDisk := true
		for i := 0; i < len(permsOnDisk); i++ {
			if permsOnDisk[i].Type == dbPerm.Type && permsOnDisk[i].Type == "anyone" {
				// the anyone permission
				isNotInPermsOnDisk = false
				break
			} else if permsOnDisk[i].EmailAddress == dbPerm.EmailAddress {
				// all other permissions
				isNotInPermsOnDisk = false
				break
			}
		}
		if isNotInPermsOnDisk {
			permIDsToDelete = append(permIDsToDelete, dbPermID)
			log.Debugf(
				"file permission queued to be deleted (email:%s,type:%s,role:%s)",
				dbPerm.EmailAddress,
				dbPerm.Type,
				dbPerm.Role,
			)
		}
	}
	// delete perms
	for i := 0; i < len(permIDsToDelete); i++ {
		err = gdriveSvc.Permissions.Delete(*fileID, permIDsToDelete[i]).SupportsAllDrives(true).Do()
		if err != nil {
			log.Error(err)
			return
		}
	}
	log.Debug("perms deleted")
	// update perms
	for permID, perm := range permsToUpdate {
		_, err = gdriveSvc.Permissions.Update(*fileID, permID, perm).SupportsAllDrives(true).Do()
		if err != nil {
			log.Error(err)
			return
		}
	}
	log.Debug("perms updated")
	// create new perms
	newPermMap := map[string]*drive.Permission{}
	for i := 0; i < len(permsToAdd); i++ {
		p, err := gdriveSvc.Permissions.Create(*fileID, permsToAdd[i]).SupportsAllDrives(true).Do()
		if err != nil {
			log.Error(err)
			return
		}
		newPermMap[p.Id] = &drive.Permission{
			EmailAddress: permsToAdd[i].EmailAddress,
			Type:         p.Type,
			Role:         p.Role,
		}
	}
	log.Debug("perms added")

	tx, err := dbcon.Begin(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	// delete perms from db
	for i := 0; i < len(permIDsToDelete); i++ {
		_, err = tx.Exec(ctx, `delete from bot.permissions where perm_gcp_id=$1`, permIDsToDelete[i])
		if err != nil {
			log.Error(err)
			return
		}
	}
	log.Debug("perms queued to be deleted from db")
	// add perms to db
	for id, perm := range newPermMap {
		_, err = tx.Exec(
			ctx,
			`insert into bot.permissions(file_gcp_id,perm_gcp_id,email,role,role_type) values($1,$2,$3,$4,$5)`,
			*fileID,
			id,
			perm.EmailAddress,
			perm.Role,
			perm.Type,
		)
		if err != nil {
			log.Error(err)
			return
		}
	}
	log.Debug("perms queued to be added to db")
	// update perms in db
	for permID, perm := range permsToUpdate {
		_, err = tx.Exec(
			ctx,
			`update bot.permissions set
				role=$1
			where perm_gcp_id=$2`,
			perm.Role,
			permID,
		)
		if err != nil {
			log.Error(err)
			return
		}
	}
	log.Debug("perms queued to be updated in db")
	err = tx.Commit(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	dbcon.Release()
	log.Debug("file permissions successfully synced")

	content := "File permissions successfully synced"
	_, err = event.Client().Rest().UpdateInteractionResponse(
		event.ApplicationID(),
		event.Token(),
		discord.MessageUpdate{
			Content: &content,
		},
	)
	if err != nil {
		log.Error(err)
		return
	}
}

func anyXivCharacterSearchHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "any_xiv_char_search" {
		return
	}

	err := event.DeferCreateMessage(true)
	if err != nil {
		log.Error(err)
		return
	}
	xivCharName := eventData.String("xiv_character_name")
	xivDiscUser := eventData.User("discord_user")
	err = xivCharacterSearch(
		xivDiscUser,
		xivCharName,
		event.Client(),
		event.ApplicationID(),
		event.Token(),
	)
	if err != nil {
		log.Error(err)
	}
}

func xivCharacterSearchHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "xiv_char_search" {
		return
	}

	err := event.DeferCreateMessage(true)
	if err != nil {
		log.Error(err)
		return
	}
	xivCharName := eventData.String("xiv_character_name")
	xivDiscUser := event.Member().User
	err = xivCharacterSearch(
		xivDiscUser,
		xivCharName,
		event.Client(),
		event.ApplicationID(),
		event.Token(),
	)
	if err != nil {
		log.Error(err)
	}
}

func mapAnyXivCharacterIDHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "map_any_xiv_char_id" {
		return
	}

	err := event.DeferCreateMessage(true)
	if err != nil {
		log.Error(err)
		return
	}
	xivCharID := eventData.String("xiv_character_id")
	xivDiscUser := eventData.User("discord_user")
	err = mapXivCharacterID(
		xivDiscUser,
		xivCharID,
		event.Client(),
		event.ApplicationID(),
		event.Token(),
	)
	if err != nil {
		log.Error(err)
	}
}

func mapXivCharacterIDHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "map_xiv_char_id" {
		return
	}

	err := event.DeferCreateMessage(true)
	if err != nil {
		log.Error(err)
		return
	}
	xivCharID := eventData.String("xiv_character_id")
	xivDiscUser := event.Member().User
	err = mapXivCharacterID(
		xivDiscUser,
		xivCharID,
		event.Client(),
		event.ApplicationID(),
		event.Token(),
	)
	if err != nil {
		log.Error(err)
	}
}

func scanXivMountsHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "scan_xiv_mounts" {
		return
	}

	err := event.DeferCreateMessage(true)
	if err != nil {
		log.Error(err)
		return
	}
	err = xivMountScan()
	if err != nil {
		log.Error(err)
		return
	}
	content := "Force mount scan completed"
	_, err = event.Client().Rest().UpdateInteractionResponse(
		event.ApplicationID(),
		event.Token(),
		discord.MessageUpdate{
			Content: &content,
		},
	)
	if err != nil {
		log.Error(err)
	}
}

func updateMemberNamesHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "update_member_names" {
		return
	}

	err := event.DeferCreateMessage(true)
	if err != nil {
		log.Error(err)
		return
	}
	// get all members from discord
	discMembers, err := event.Client().Rest().GetMembers(*event.GuildID(), guildMemberCountRequestLimit, nullSnowflake)
	if err != nil {
		log.Error(err)
		return
	}
	err = discordNicknameScan(discMembers)
	if err != nil {
		log.Error(err)
		return
	}
	content := "Names updated in spreadsheet"
	_, err = event.Client().Rest().UpdateInteractionResponse(
		event.ApplicationID(),
		event.Token(),
		discord.MessageUpdate{
			Content: &content,
		},
	)
	if err != nil {
		log.Error(err)
	}
}
