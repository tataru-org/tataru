package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/log"
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

const DefaultSheetID int64 = 0

func buildFile(badFileExists bool) (*FileID, error) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()
	fileID, err := createFile(botConfig.MountSpreadsheetFileName)
	if err != nil {
		return nil, fmt.Errorf("file creation error: [%w]", err)
	}
	log.Debugf("file created: %s", *fileID)
	expansions, err := getExpansions()
	if err != nil {
		return nil, fmt.Errorf("getExpansions() error: [%w]", err)
	}

	// add permissions to the file
	permsFromDisk, err := GetPermissions(mountSpreadsheetPermissionsFilepath)
	if err != nil {
		return nil, fmt.Errorf("GetPermissions() error: [%w]", err)
	}
	newPermMap := map[string]*drive.Permission{}
	for i := 0; i < len(permsFromDisk); i++ {
		p, err := gdriveSvc.Permissions.Create(string(*fileID), permsFromDisk[i]).SupportsAllDrives(true).Do()
		if err != nil {
			return nil, fmt.Errorf("gdriveSvc.PermissionsCreate() error; i=%d, permsFromDisk=[%v]: [%w]", i, permsFromDisk[i], err)
		}
		newPermMap[p.Id] = &drive.Permission{
			EmailAddress: permsFromDisk[i].EmailAddress,
			Type:         p.Type,
			Role:         p.Role,
		}
		log.Debugf(
			"permission added for: id=%s;email=%s;role=%s;type=%s",
			p.Id,
			permsFromDisk[i].EmailAddress,
			permsFromDisk[i].Role,
			permsFromDisk[i].Type,
		)
	}

	spreadsheet, err := gsheetsSvc.Spreadsheets.Get(string(*fileID)).Do()
	if err != nil {
		return nil, fmt.Errorf("gsheetsSvc.Spreadsheets.Get() 1 error: [%w]", err)
	}
	// create the sheets
	numSheets := len(expansions)
	requests := make([]*sheets.Request, numSheets)
	for i := 0; i < numSheets; i++ {
		requests[i] = &sheets.Request{
			AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{
					Index: int64(i),
					Title: fmt.Sprintf("init-%d", i),
				},
			},
		}
	}
	requests = append(requests, &sheets.Request{
		UpdateSpreadsheetProperties: &sheets.UpdateSpreadsheetPropertiesRequest{
			Properties: &sheets.SpreadsheetProperties{
				Title: botConfig.MountSpreadsheetTitle,
			},
			Fields: "title",
		},
	})
	// the sheets api docs state that some replies may be empty, so do not rely on the response to
	// get the sheet IDs from the spreadsheet
	_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return nil, fmt.Errorf("gsheetsSvc.Spreadsheets.BatchUpdate() error: [%w]", err)
	}
	log.Debug("sheets created")

	// delete default sheet
	_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteSheet: &sheets.DeleteSheetRequest{
					SheetId: DefaultSheetID,
				},
			},
		},
	}).Do()
	if err != nil {
		return nil, fmt.Errorf("gsheetsSvc.Spreadsheets.BatchUpdate() error: [%w]", err)
	}
	log.Debug("default sheet deleted")

	// collect and map sheet metadata to expansion metadata
	spreadsheet, err = gsheetsSvc.Spreadsheets.Get(spreadsheet.SpreadsheetId).Do()
	if err != nil {
		return nil, fmt.Errorf("gsheetsSvc.Spreadsheets.Get() 2 error: [%w]", err)
	}
	expansionSheetMap := make(map[SheetID]*Expansion)
	sheetData := make([]*SheetMetadata, len(spreadsheet.Sheets))
	for i := 0; i < len(spreadsheet.Sheets); i++ {
		sheet := spreadsheet.Sheets[i]
		sheetData[i] = &SheetMetadata{
			ID:    SheetID(sheet.Properties.SheetId),
			Index: SheetIndex(sheet.Properties.Index),
		}
		for j := 0; j < len(expansions); j++ {
			if int(sheet.Properties.Index) == int(expansions[j].Index) {
				expansionSheetMap[SheetID(sheet.Properties.SheetId)] = expansions[j]
			}
		}
	}

	// save the sheet metadata and sheet-expansion map to the database
	tx, err := dbcon.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("dbcon.Begin() 1 error: [%w]", err)
	}
	if badFileExists {
		// delete data in file id table
		_, err = tx.Exec(ctx, `delete from bot.file_ref`)
		if err != nil {
			return nil, fmt.Errorf("tx.Exec() 1-1 error: [%w]", err)
		}
	}
	// put file id into db
	_, err = tx.Exec(ctx, `insert into bot.file_ref(file_gcp_id) values($1)`, string(*fileID))
	if err != nil {
		return nil, fmt.Errorf("tx.Exec() 1-2 error: [%w]", err)
	}
	for i := 0; i < len(sheetData); i++ {
		_, err = tx.Exec(
			ctx,
			`insert into bot.sheet_metadata(file_gcp_id,sheet_gcp_id,sheet_index) values($1,$2,$3)`,
			string(*fileID),
			sheetData[i].ID.String(),
			sheetData[i].Index.String(),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"tx.Exec() 1-3 error; file_gcp_id=%s,sheet_gcp_id=%s,sheet_index=%s: [%w]",
				string(*fileID),
				sheetData[i].ID.String(),
				sheetData[i].Index.String(),
				err,
			)
		}
		_, err = tx.Exec(
			ctx,
			`insert into bot.sheet_expansion_map(sheet_gcp_id,expansion_id) values($1,$2)`,
			sheetData[i].ID.String(),
			expansionSheetMap[sheetData[i].ID].ID,
		)
		if err != nil {
			return nil, fmt.Errorf("tx.Exec() 1-4 error: [%w]", err)
		}
	}
	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx.Commit() 1 error: [%w]", err)
	}

	columnMap, err := NewColumnMap()
	if err != nil {
		return nil, fmt.Errorf("NewColumnMap() error: [%w]", err)
	}

	// add the header row to each sheet & update each sheet's name
	requests = []*sheets.Request{}
	for sheet, columnIndexMap := range columnMap.Mapping {
		numColumns := len(columnIndexMap)
		cellData := make([]*sheets.CellData, numColumns)
		for columnIndex, columnData := range columnIndexMap {
			cellData[columnIndex] = &sheets.CellData{
				UserEnteredValue: &sheets.ExtendedValue{
					StringValue: (*string)(&columnData.Name),
				},
				UserEnteredFormat: columnData.HeaderFormat,
			}
		}
		requests = append(
			requests,
			&sheets.Request{
				AppendCells: &sheets.AppendCellsRequest{
					Fields: "user_entered_value,user_entered_format",
					Rows: []*sheets.RowData{
						{
							Values: cellData,
						},
					},
					SheetId: int64(sheet.ID),
				},
			},
		)
		requests = append(
			requests,
			&sheets.Request{
				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Fields: "title",
					Properties: &sheets.SheetProperties{
						Title:   string(expansionSheetMap[sheet.ID].Name),
						SheetId: int64(sheet.ID),
						Index:   int64(sheet.Index),
					},
				},
			},
		)
	}

	// intentional execution blocking
	googleSheetsWriteReqs <- &SheetBatchUpdate{
		ID: spreadsheet.SpreadsheetId,
		Batch: &sheets.BatchUpdateSpreadsheetRequest{
			Requests: requests,
		},
	}
	log.Debug("header rows added to each sheet")

	// save what is needed to the db
	tx, err = dbcon.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("dbcon.Begin() 2 error: [%w]", err)
	}
	// put perm ids into db
	for id, perm := range newPermMap {
		_, err = tx.Exec(
			ctx,
			`insert into bot.permissions(file_gcp_id,perm_gcp_id,email,role,role_type) values($1,$2,$3,$4,$5)`,
			string(*fileID),
			id,
			perm.EmailAddress,
			perm.Role,
			perm.Type,
		)
		if err != nil {
			return nil, fmt.Errorf("tx.Exec() 2-3 error; id=%s, perm=%v [%w]", id, *perm, err)
		}
	}
	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx.Commit() error: [%w]", err)
	}
	log.Debug("required data saved to db")
	return fileID, nil
}

func syncRoleMembers(id FileID, guildMembers []discord.Member) error {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()
	// get members from db
	dbMembers, err := getMembersFromDB()
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("getMembersFromDB() error: [%w]", err)
	}
	// get the watched role id
	var roleID *string
	row := dbcon.QueryRow(ctx, `select role_id from bot.role_ref`)
	err = row.Scan(&roleID)
	if err == pgx.ErrNoRows {
		// exit if role is not set
		return nil
	} else if err != nil {
		return fmt.Errorf("getting role_id error: [%w]", err)
	}
	log.Debug("got role id")

	// get column formatting
	columnMap, err := NewColumnMap()
	if err != nil {
		return fmt.Errorf("NewColumnMap() error: [%w]", err)
	}

	// filter out members without the watched role id
	roleMembers := []discord.Member{}
	for i := 0; i < len(guildMembers); i++ {
		if guildMembers[i].User.Bot {
			continue
		}

		for j := 0; j < len(guildMembers[i].RoleIDs); j++ {
			if guildMembers[i].RoleIDs[j].String() == *roleID {
				roleMembers = append(roleMembers, guildMembers[i])
				break
			}
		}
	}
	log.Debug("filtered members")

	spreadsheet, err := gsheetsSvc.Spreadsheets.Get(string(id)).IncludeGridData(true).Do()
	if err != nil {
		return fmt.Errorf("gsheetsSvc.Spreadsheets.Get() 1 error: [%w]", err)
	}
	if len(dbMembers) == len(roleMembers) && len(dbMembers) == 0 {
		return nil
	}

	// get the members to delete from the spreadsheet
	deleteMembers := []*Member{}
	filteredDBMembers := []*Member{}
	for i := 0; i < len(dbMembers); i++ {
		memberShouldExist := false
		for j := 0; j < len(roleMembers); j++ {
			if dbMembers[i].id == MemberID(roleMembers[j].User.ID.String()) {
				memberShouldExist = true
			}
		}
		if !memberShouldExist {
			deleteMembers = append(deleteMembers, dbMembers[i])
		} else {
			filteredDBMembers = append(filteredDBMembers, dbMembers[i])
		}
	}
	log.Debug("got members to delete")
	// map the row indices of each member to delete
	deleteMemberMap := map[int64]*Member{}
	testSheet := spreadsheet.Sheets[0]
	numRows := len(testSheet.Data[0].RowData) - 1
	for i := 0; i < len(deleteMembers); i++ {
		for j := 0; j < numRows; j++ {
			rowIndex := j + 1
			row := testSheet.Data[0].RowData[rowIndex]
			if MemberID(*row.Values[0].EffectiveValue.StringValue) == deleteMembers[i].id {
				deleteMemberMap[int64(rowIndex)] = deleteMembers[i]
			}
		}
	}
	log.Debug("mapped row indices to each member to delete from spreadsheet")
	// delete the members' rows in the spreadsheet
	requests := make([]*sheets.Request, len(deleteMemberMap)*len(spreadsheet.Sheets))
	requestIndex := 0
	for i := 0; i < len(spreadsheet.Sheets); i++ {
		for rowIndex, member := range deleteMemberMap {
			requests[requestIndex] = &sheets.Request{
				DeleteRange: &sheets.DeleteRangeRequest{
					Range: &sheets.GridRange{
						StartRowIndex: rowIndex,
						EndRowIndex:   rowIndex + 1,
						SheetId:       spreadsheet.Sheets[i].Properties.SheetId,
					},
					ShiftDimension: "ROWS",
				},
			}
			log.Debugf("member %s (id:%s) queued to be deleted from spreadsheet %d", member.name, string(member.id), i)
			requestIndex++
		}
	}
	if len(requests) != 0 {
		go func() {
			googleSheetsWriteReqs <- &SheetBatchUpdate{
				ID: spreadsheet.SpreadsheetId,
				Batch: &sheets.BatchUpdateSpreadsheetRequest{
					Requests: requests,
				},
			}
		}()
		log.Debug("members deleted from spreadsheet")
	} else {
		log.Debug("members not deleted from spreadsheet")
	}

	// delete members from the db
	tx, err := dbcon.Begin(ctx)
	if err != nil {
		return fmt.Errorf("dbcon.Begin() 1 error: [%w]", err)
	}
	for i := 0; i < len(deleteMembers); i++ {
		_, err = tx.Exec(ctx, `delete from bot.member_metadata where member_discord_id=$1`, string(deleteMembers[i].id))
		if err != nil {
			return fmt.Errorf("delete from bot.member_metadata error; member_discord_id=%s: [%w]", string(deleteMembers[i].id), err)
		}
	}
	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("tx.Commit() 1 error: [%w]", err)
	}
	log.Debugf("deleted %d members from db", len(deleteMembers))

	spreadsheet, err = gsheetsSvc.Spreadsheets.Get(string(id)).IncludeGridData(true).Do()
	if err != nil {
		return fmt.Errorf("gsheetsSvc.Spreadsheets.Get() 2 error: [%w]", err)
	}
	if len(filteredDBMembers) == len(roleMembers) && len(filteredDBMembers) == 0 {
		return nil
	}

	// get the members to add to the spreadsheet based on the differences between discord and the database
	addMembers := []discord.Member{}
	for i := 0; i < len(roleMembers); i++ {
		memberAlreadyExists := false
		for j := 0; j < len(filteredDBMembers); j++ {
			if filteredDBMembers[j].id == MemberID(roleMembers[i].User.ID.String()) {
				memberAlreadyExists = true
			}
		}
		if !memberAlreadyExists {
			addMembers = append(addMembers, roleMembers[i])
		}
	}
	log.Debug("got members to add based on differences between discord and the database")
	// get members to add to the spreadsheet based on differences between the database and the spreadsheet
	ssMembers := getSpreadsheetMembers(spreadsheet)
	for i := 0; i < len(filteredDBMembers); i++ {
		existsInSpreadsheet := false
		for j := 0; j < len(ssMembers); j++ {
			if ssMembers[j].id == filteredDBMembers[i].id {
				existsInSpreadsheet = true
				break
			}
		}
		if !existsInSpreadsheet {
			snowMemberID, err := snowflake.Parse(string(filteredDBMembers[i].id))
			if err != nil {
				return fmt.Errorf("snowflake.Parse() error: [%w]", err)
			}
			m := discord.Member{
				User: discord.User{
					ID:       snowMemberID,
					Username: filteredDBMembers[i].name,
				},
				Nick: nil,
			}
			addMembers = append(addMembers, m)
		}
	}
	log.Debug("got members to add based on differences between the database and the spreadsheet")
	// add the members' rows in the spreadsheet
	counter := 0
	requests = make([]*sheets.Request, len(columnMap.Mapping))
	for sheetMetadata, sheetColumnMap := range columnMap.Mapping {
		rowData := []*sheets.RowData{}
		for j := 0; j < len(addMembers); j++ {
			userID := addMembers[j].User.ID.String()
			var username string
			if addMembers[j].Nick == nil {
				username = addMembers[j].User.Username
			} else {
				username = *addMembers[j].Nick
			}

			vals := []*sheets.CellData{
				{
					UserEnteredValue: &sheets.ExtendedValue{
						StringValue: &userID,
					},
					UserEnteredFormat: sheetColumnMap[0].ColumnFormat,
				},
				{
					UserEnteredValue: &sheets.ExtendedValue{
						StringValue: &username,
					},
					UserEnteredFormat: sheetColumnMap[1].ColumnFormat,
				},
			}
			boolVal := false
			numColumns := len(sheetColumnMap)
			for k := 0; k < numColumns-2; k++ {
				vals = append(vals, &sheets.CellData{
					UserEnteredFormat: sheetColumnMap[ColumnIndex(k+2)].ColumnFormat,
					UserEnteredValue: &sheets.ExtendedValue{
						BoolValue: &boolVal,
					},
					DataValidation: &sheets.DataValidationRule{
						Condition: &sheets.BooleanCondition{
							Type: "BOOLEAN",
						},
					},
				})
			}
			rowData = append(rowData, &sheets.RowData{
				Values: vals,
			})
			log.Debugf("member %s (id:%s) queued to be added to spreadsheet %d", username, userID, sheetMetadata.Index)
		}
		requests[counter] = &sheets.Request{
			AppendCells: &sheets.AppendCellsRequest{
				Fields:  "*",
				SheetId: int64(sheetMetadata.ID),
				Rows:    rowData,
			},
		}
		counter++
	}
	if len(requests) != 0 {
		go func() {
			googleSheetsWriteReqs <- &SheetBatchUpdate{
				ID: spreadsheet.SpreadsheetId,
				Batch: &sheets.BatchUpdateSpreadsheetRequest{
					Requests: requests,
				},
			}
		}()
		log.Debug("members added to spreadsheet")
	} else {
		log.Debug("members not added to spreadsheet")
	}

	tx, err = dbcon.Begin(ctx)
	if err != nil {
		return fmt.Errorf("dbcon.Begin() 2 error: [%w]", err)
	}
	// add members to db
	for i := 0; i < len(addMembers); i++ {
		var name string
		if addMembers[i].Nick == nil {
			name = addMembers[i].User.Username
		} else {
			name = *addMembers[i].Nick
		}
		_, err = tx.Exec(
			ctx,
			`
			insert into bot.member_metadata(
				member_discord_id,
				member_name
			) values(
				$1,
				$2
			) on conflict (
				member_discord_id
			) do nothing
			`,
			addMembers[i].User.ID.String(),
			name,
		)
		if err != nil {
			return fmt.Errorf("add member to db error; member_discord_id=%s: [%w]", addMembers[i].User.ID.String(), err)
		}
	}
	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("tx.Commit() 2 error: [%w]", err)
	}
	log.Debugf("added %d members to db", len(addMembers))
	return nil
}

func xivMountScan() error {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()
	// get all members that have xiv character IDs and create requests
	query := `
		select
			member_discord_id,
			member_name,
			member_xiv_id
		from bot.member_metadata
		where member_xiv_id is not null
		order by member_name
	`
	rows, err := dbcon.Query(
		ctx,
		query,
	)
	if err != nil {
		return fmt.Errorf("get all members for mount scan error: [%w]", err)
	}
	reqMap := map[snowflake.ID]XivCharacterRequest{}
	requests := []XivCharacterRequest{}
	for rows.Next() {
		var memberIDStr string
		var membername string
		var memberXivID string
		err = rows.Scan(&memberIDStr, &membername, &memberXivID)
		if err != nil {
			return fmt.Errorf("row scan 1 error: [%w]", err)
		}
		memberID, err := snowflake.Parse(memberIDStr)
		if err != nil {
			return fmt.Errorf("parse snowflake 1 error; member_discord_id=%s: [%w]", memberIDStr, err)
		}
		req := XivCharacterRequest{
			Token: uuid.New().String(),
			XivID: memberXivID,
			Data: []XivCharacterData{
				XivCharacterDataMountsMinions,
			},
			Do: xivapiClient.GetCharacter,
		}
		reqMap[memberID] = req
		requests = append(requests, req)
	}
	log.Debugf("# of character requests created: %d", len(requests))
	if len(requests) == 0 {
		return nil
	}
	// send requests and collect character profiles containing the mount data
	log.Debug("sending requests")
	xivCharProfiles, err := xivapiCollectCharacterResponses(requests)
	log.Debug("character profiles collected")
	if err != nil {
		return fmt.Errorf("xivapiCollectCharacterResponses() error: [%w]", err)
	}
	if len(xivCharProfiles) == 0 {
		return nil
	}
	// map discord user IDs to xiv character profiles
	profileMap := map[snowflake.ID]XivCharacter{}
	for i := 0; i < len(xivCharProfiles); i++ {
		for discUserID, xivCharRequest := range reqMap {
			if xivCharRequest.XivID == strconv.FormatUint(uint64(xivCharProfiles[i].Character.ID), 10) {
				profileMap[discUserID] = xivCharProfiles[i]
				break
			}
		}
	}
	// update database with member data
	mountMetadata, err := getXivMountMetadata()
	if err != nil {
		return fmt.Errorf("getXivMountMetadata() error: [%w]", err)
	}
	tx, err := dbcon.Begin(ctx)
	if err != nil {
		return fmt.Errorf("dbcon.Begin() 1 error: [%w]", err)
	}
	for memberID, xivChar := range profileMap {
		for i := 0; i < len(mountMetadata); i++ {
			for j := 0; j < len(xivChar.Mounts); j++ {
				if string(mountMetadata[i].Name) == xivChar.Mounts[j].Name {
					_, err = tx.Exec(
						ctx,
						`
						insert into bot.member_data(
							member_discord_id,
							mount_id,
							has_mount
						) values(
							$1,
							$2,
							$3
						)
						on conflict (
							member_discord_id,
							mount_id
						)
						do update set
							has_mount=$3
						where
							member_data.member_discord_id=$1
							and member_data.mount_id=$2
						`,
						memberID.String(),
						mountMetadata[i].ID,
						true,
					)
					if err != nil {
						return fmt.Errorf("upsert member data error: [%w]", err)
					}
					break
				}
			}
		}
	}
	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("tx.Commit() 1 error: [%w]", err)
	}

	// everything below this tldr: sync member data in google sheets
	bossMountMap, err := getXivBossMountMapping()
	if err != nil {
		return fmt.Errorf("getXivBossMountMapping() error: [%w]", err)
	}
	// get the spreadsheet file id
	var fileID string
	row := dbcon.QueryRow(ctx, `select file_gcp_id from bot.file_ref`)
	err = row.Scan(&fileID)
	if err != nil {
		return fmt.Errorf("row scan 2 error: [%w]", err)
	}
	// get the spreadsheet with all file data
	spreadsheet, err := gsheetsSvc.Spreadsheets.Get(fileID).IncludeGridData(true).Do()
	if err != nil {
		return fmt.Errorf("gsheetsSvc.Spreadsheets.Get() error: [%w]", err)
	}
	// get the column format mapping
	columnMap, err := NewColumnMap()
	if err != nil {
		return fmt.Errorf("NewColumnMap() error: [%w]", err)
	}
	// create sheets api requests to update values according
	// to the mount list in the corresponding character profile
	gapiRequests := []*sheets.Request{}
	for i := 0; i < len(spreadsheet.Sheets); i++ {
		for memberID, charProfile := range profileMap {
			for j := 1; j < len(spreadsheet.Sheets[i].Data[0].RowData); j++ {
				sheet := spreadsheet.Sheets[i]
				row := sheet.Data[0].RowData[j]
				curID, err := snowflake.Parse(*row.Values[0].EffectiveValue.StringValue)
				if err != nil {
					return fmt.Errorf("parse snowflake 2 error: [%w]", err)
				}
				if curID != memberID {
					continue
				}

				vals := []*sheets.CellData{}
				for k := 2; k < len(row.Values); k++ {
					hasMount := false
					bossName := columnMap.Mapping[SheetMetadata{
						ID:    SheetID(sheet.Properties.SheetId),
						Index: SheetIndex(sheet.Properties.Index),
					}][ColumnIndex(k)].Name
					for a := 0; a < len(charProfile.Mounts); a++ {
						mountName := charProfile.Mounts[a].Name
						if mountName == string(bossMountMap[BossName(bossName)]) {
							hasMount = true
							break
						}
					}
					vals = append(vals, &sheets.CellData{
						UserEnteredValue: &sheets.ExtendedValue{
							BoolValue: &hasMount,
						},
					})
				}

				gapiRequests = append(gapiRequests, &sheets.Request{
					UpdateCells: &sheets.UpdateCellsRequest{
						Fields: "userEnteredValue",
						Range: &sheets.GridRange{
							SheetId:          spreadsheet.Sheets[i].Properties.SheetId,
							StartRowIndex:    int64(j),
							EndRowIndex:      int64(j + 1),
							StartColumnIndex: 2,
						},
						Rows: []*sheets.RowData{
							{
								Values: vals,
							},
						},
					},
				})
			}
		}
	}
	if len(gapiRequests) > 0 {
		// send the batch request
		go func() {
			googleSheetsWriteReqs <- &SheetBatchUpdate{
				ID: spreadsheet.SpreadsheetId,
				Batch: &sheets.BatchUpdateSpreadsheetRequest{
					Requests: gapiRequests,
				},
			}
		}()
		log.Debug("Data in spreadsheet successfully queued to be updated")
	} else {
		log.Debug("Nothing to update")
	}
	return nil
}

func scanForMounts() {
	for {
		err := xivMountScan()
		if err != nil {
			log.Error(err)
		}
		<-time.After(xivapiMountScanSleepDuration)
	}
}

func discordNicknameScan(discMembers []discord.Member) error {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()
	// get file id
	var fileID string
	row := dbcon.QueryRow(ctx, `select file_gcp_id from bot.file_ref`)
	err = row.Scan(&fileID)
	if err != nil {
		return fmt.Errorf("getting file id error: [%w]", err)
	}
	// get all members in db
	dbMembers, err := getMembersFromDB()
	if err != nil {
		return fmt.Errorf("getMembersFromDB() error: [%w]", err)
	}
	// map discord ID to current member nickname
	memberMap := map[string]string{}
	for i := 0; i < len(dbMembers); i++ {
		for j := 0; j < len(discMembers); j++ {
			if string(dbMembers[i].id) != discMembers[j].User.ID.String() || discMembers[j].User.Bot {
				continue
			}
			var username string
			if discMembers[j].Nick == nil {
				username = discMembers[j].User.Username
			} else {
				username = *discMembers[j].Nick
			}
			memberMap[string(dbMembers[i].id)] = username
			break
		}
	}
	spreadsheet, err := gsheetsSvc.Spreadsheets.Get(fileID).IncludeGridData(true).Do()
	if err != nil {
		return fmt.Errorf("gsheetsSvc.Spreadsheets.Get() error: [%w]", err)
	}
	// update all member names in the spreadsheet
	requests := []*sheets.Request{}
	for i := 0; i < len(spreadsheet.Sheets); i++ {
		for j := 0; j < len(spreadsheet.Sheets[i].Data[0].RowData); j++ {
			for userID, userName := range memberMap {
				row := spreadsheet.Sheets[i].Data[0].RowData[j]
				if *row.Values[0].EffectiveValue.StringValue != userID {
					continue
				}
				if *row.Values[1].EffectiveValue.StringValue == userName {
					break
				}
				requests = append(requests, &sheets.Request{
					UpdateCells: &sheets.UpdateCellsRequest{
						Fields: "userEnteredValue",
						Range: &sheets.GridRange{
							SheetId:          int64(spreadsheet.Sheets[i].Properties.SheetId),
							StartRowIndex:    int64(j),
							EndRowIndex:      int64(j + 1),
							StartColumnIndex: 1,
							EndColumnIndex:   2,
						},
						Rows: []*sheets.RowData{
							{
								Values: []*sheets.CellData{
									{
										UserEnteredValue: &sheets.ExtendedValue{
											StringValue: &userName,
										},
									},
								},
							},
						},
					},
				})
				break
			}
		}
	}
	// update in database
	if len(requests) > 0 {
		go func() {
			googleSheetsWriteReqs <- &SheetBatchUpdate{
				ID: spreadsheet.SpreadsheetId,
				Batch: &sheets.BatchUpdateSpreadsheetRequest{
					Requests: requests,
				},
			}
		}()

		tx, err := dbcon.Begin(ctx)
		if err != nil {
			return fmt.Errorf("dbcon.Begin() error: [%w]", err)
		}
		for userID, userName := range memberMap {
			_, err = tx.Exec(ctx, `update bot.member_metadata set member_name=$1 where member_discord_id=$2`, userName, userID)
			if err != nil {
				return fmt.Errorf("update bot.member_metadata error: [%w]", err)
			}
		}
		err = tx.Commit(ctx)
		if err != nil {
			return fmt.Errorf("tx.Commit() error: [%w]", err)
		}
	}
	return nil
}

func getSpreadsheetMembers(ss *sheets.Spreadsheet) []*Member {
	members := []*Member{}
	testSheet := ss.Sheets[0]
	numRows := len(testSheet.Data[0].RowData) - 1
	for i := 0; i < numRows; i++ {
		row := testSheet.Data[0].RowData[i+1]
		member := &Member{
			id:   MemberID(*row.Values[0].EffectiveValue.StringValue),
			name: *row.Values[1].EffectiveValue.StringValue,
		}
		members = append(members, member)
	}
	return members
}

func xivCharacterSearch(
	user discord.User,
	xivCharName string,
	discClient bot.Client,
	discAppID snowflake.ID,
	discToken string,
) error {
	searchResponses, err := xivapiCollectCharacterSearchResponses([]XivCharacterSearchRequest{
		{
			Token: uuid.New().String(),
			Name:  xivCharName,
			Params: []XivApiQueryParam{
				{
					Name:  "server",
					Value: "Behemoth",
				},
			},
			Do: xivapiClient.SearchForCharacter,
		},
	})
	if err != nil {
		return fmt.Errorf("xivapiCollectCharacterSearchResponses() error: [%w]", err)
	}
	if len(searchResponses) == 0 {
		content := "No matching search results were found"
		_, err = discClient.Rest().UpdateInteractionResponse(
			discAppID,
			discToken,
			discord.MessageUpdate{
				Content: &content,
			},
		)
		if err != nil {
			return fmt.Errorf("discClient.Rest().UpdateInteractionResponse() 1 error: [%w]", err)
		}
		return nil
	}
	// discord ID -> xiv character ID
	var xivCharID *string = nil
	charSearch := searchResponses[0]
	for j := 0; j < len(charSearch.Results); j++ {
		if charSearch.Results[j].Name == xivCharName {
			s := strconv.FormatUint(uint64(charSearch.Results[j].ID), 10)
			xivCharID = &s
			break
		}
	}
	if xivCharID == nil {
		content := "No matching search results were found"
		_, err = discClient.Rest().UpdateInteractionResponse(
			discAppID,
			discToken,
			discord.MessageUpdate{
				Content: &content,
			},
		)
		if err != nil {
			return fmt.Errorf("discClient.Rest().UpdateInteractionResponse() 2 error: [%w]", err)
		}
		return nil
	}

	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()
	_, err = dbcon.Exec(ctx, `update bot.member_metadata set member_xiv_id=$1 where member_discord_id=$2`, *xivCharID, user.ID.String())
	if err != nil {
		return fmt.Errorf("update bot.member_metadata error: [%w]", err)
	}
	content := fmt.Sprintf("Character ID %s was found with character name %s for discord user %s", *xivCharID, xivCharName, user.ID.String())
	_, err = discClient.Rest().UpdateInteractionResponse(
		discAppID,
		discToken,
		discord.MessageUpdate{
			Content: &content,
		},
	)
	if err != nil {
		return fmt.Errorf("discClient.Rest().UpdateInteractionResponse() 3 error: [%w]", err)
	}
	err = mapXivCharacterID(
		user,
		*xivCharID,
		discClient,
		discAppID,
		discToken,
	)
	if err != nil {
		return fmt.Errorf("mapXivCharacterID() error: [%w]", err)
	}
	return nil
}

func mapXivCharacterID(
	user discord.User,
	xivCharID string,
	discClient bot.Client,
	discAppID snowflake.ID,
	discToken string,
) error {
	resps, err := xivapiCollectCharacterResponses([]XivCharacterRequest{
		{
			Token: uuid.New().String(),
			XivID: xivCharID,
			Data:  nil,
			Do:    xivapiClient.GetCharacter,
		},
	})
	if err != nil {
		return fmt.Errorf("xivapiCollectCharacterResponses() error: [%w]", err)
	}
	if len(resps) == 0 {
		content := fmt.Sprintf("No matching character was found for character ID %s", xivCharID)
		_, err = discClient.Rest().UpdateInteractionResponse(
			discAppID,
			discToken,
			discord.MessageUpdate{
				Content: &content,
			},
		)
		if err != nil {
			return fmt.Errorf("discClient.Rest().UpdateInteractionResponse() 1 error: [%w]", err)
		}
		return nil
	}
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()
	_, err = dbcon.Exec(ctx, `update bot.member_metadata set member_xiv_id=$1 where member_discord_id=$2`, xivCharID, user.ID.String())
	if err != nil {
		return fmt.Errorf("update bot.member_metadata error: [%w]", err)
	}
	dbcon.Release()
	content := fmt.Sprintf("Character ID %s was found for discord user %s", xivCharID, user.ID.String())
	_, err = discClient.Rest().UpdateInteractionResponse(
		discAppID,
		discToken,
		discord.MessageUpdate{
			Content: &content,
		},
	)
	if err != nil {
		return fmt.Errorf("discClient.Rest().UpdateInteractionResponse() 2 error: [%w]", err)
	}
	return nil
}
