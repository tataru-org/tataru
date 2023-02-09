package main

import (
	"strconv"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

func buildFile(badFileExists bool) (*FileID, error) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer dbcon.Release()
	fileID, err := createFile(botConfig.MountSpreadsheetFileName)
	if err != nil {
		return nil, err
	}
	log.Debugf("file created: %s", *fileID)
	columnMap, err := NewColumnMap(mountSpreadsheetColumnDataFilepath)
	if err != nil {
		return nil, err
	}
	sheetNames, err := columnMap.GetSheetNames()
	if err != nil {
		return nil, err
	}

	// add permissions to the file
	permsFromDisk, err := GetPermissions(mountSpreadsheetPermissionsFilepath)
	if err != nil {
		return nil, err
	}
	newPermMap := map[string]*drive.Permission{}
	for i := 0; i < len(permsFromDisk); i++ {
		p, err := gdriveSvc.Permissions.Create(string(*fileID), permsFromDisk[i]).SupportsAllDrives(true).Do()
		if err != nil {
			return nil, err
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
		return nil, err
	}
	// create the sheets
	numSheets := len(sheetNames)
	requests := make([]*sheets.Request, numSheets)
	for i := 0; i < numSheets; i++ {
		requests[i] = &sheets.Request{
			AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{
					Index: int64(i),
					Title: sheetNames[i],
				},
			},
		}
	}
	requests = append(requests, &sheets.Request{
		UpdateSpreadsheetProperties: &sheets.UpdateSpreadsheetPropertiesRequest{
			Properties: &sheets.SpreadsheetProperties{
				Title: botConfig.MountSpreadsheetTitle,
			},
			Fields: "*",
		},
	})
	// the sheets api docs state that some replies may be empty, so do not rely on the response to
	// get the sheet IDs from the spreadsheet
	_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return nil, err
	}
	log.Debug("sheets created")

	// delete the default sheet
	var defaultSheetID int64 = 0
	expansionTitles := getExpansionNames()
	for i := 0; i < len(spreadsheet.Sheets); i++ {
		isExpansionSheet := false
		sheet := spreadsheet.Sheets[i]
		for j := 0; j < len(expansionTitles); j++ {
			if sheet.Properties.Title == string(expansionTitles[j]) {
				isExpansionSheet = true
				break
			}
		}
		if !isExpansionSheet {
			defaultSheetID = spreadsheet.Sheets[i].Properties.SheetId
			break
		}
	}
	_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteSheet: &sheets.DeleteSheetRequest{
					SheetId: defaultSheetID,
				},
			},
		},
	}).Do()
	if err != nil {
		return nil, err
	}
	log.Debug("default sheet deleted")

	// map expansions to sheet IDs
	spreadsheet, err = gsheetsSvc.Spreadsheets.Get(spreadsheet.SpreadsheetId).Do()
	if err != nil {
		return nil, err
	}
	sheetMap := make(map[Expansion]SheetID, numSheets)
	for i := 0; i < numSheets; i++ {
		sheet := spreadsheet.Sheets[i]
		exp, err := ExpansionNameToExpansion(ExpansionName(sheet.Properties.Title))
		if err != nil {
			continue
		}
		sheetMap[exp] = SheetID(sheet.Properties.SheetId)
	}

	// add the header row to each sheet & add protected ranges
	requests = []*sheets.Request{}
	for sheetIndex, columnIndexMap := range columnMap.Mapping {
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
		requests = append(requests, &sheets.Request{
			AppendCells: &sheets.AppendCellsRequest{
				Fields: "*",
				Rows: []*sheets.RowData{
					{
						Values: cellData,
					},
				},
				SheetId: int64(sheetMap[Expansion(sheetIndex)]),
			},
		})
	}
	requests = append(requests, &sheets.Request{
		UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
			Properties: &sheets.SheetProperties{
				Index:   0,
				SheetId: int64(sheetMap[Expansion(0)]),
			},
			Fields: "index,sheetId",
		},
	})
	googleSheetsWriteReqs <- &SheetBatchUpdate{
		ID: spreadsheet.SpreadsheetId,
		Batch: &sheets.BatchUpdateSpreadsheetRequest{
			Requests: requests,
		},
	}
	log.Debug("header rows and protected ranges added to each sheet")

	// save what is needed to the db
	tx, err := dbcon.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if badFileExists {
		// delete data in file id table
		_, err = tx.Exec(ctx, `delete from bot.file_ref`)
		if err != nil {
			return nil, err
		}
	}
	// put file id into db
	_, err = tx.Exec(ctx, `insert into bot.file_ref(file_id) values($1)`, fileID)
	if err != nil {
		return nil, err
	}
	// put perm ids into db
	for id, perm := range newPermMap {
		_, err = tx.Exec(
			ctx,
			`insert into bot.permissions(file_id,perm_id,email,role,role_type) values($1,$2,$3,$4,$5)`,
			fileID,
			id,
			perm.EmailAddress,
			perm.Role,
			perm.Type,
		)
		if err != nil {
			return nil, err
		}
	}
	// put sheet IDs into db
	for exp, sheetID := range sheetMap {
		sheetIDStr := strconv.FormatInt(int64(sheetID), 10)
		_, err = tx.Exec(ctx, `insert into bot.sheets(file_id,expansion,sheet_id) values($1,$2,$3)`, fileID, int(exp), sheetIDStr)
		if err != nil {
			return nil, err
		}
	}
	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}
	log.Debug("required data saved to db")
	return fileID, nil
}

func syncRoleMembers(id FileID, guildMembers []discord.Member) error {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer dbcon.Release()
	// get members from db
	dbMembers, err := getMembersFromDB()
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	// get the watched role id
	var roleID *string
	row := dbcon.QueryRow(ctx, `select role_id from bot.role_ref`)
	err = row.Scan(&roleID)
	if err == pgx.ErrNoRows {
		// exit if role is not set
		return nil
	} else if err != nil {
		return err
	}
	log.Debug("got role id")

	// get column formatting
	columnMap, err := NewColumnMap(mountSpreadsheetColumnDataFilepath)
	if err != nil {
		return err
	}

	// filter out members without the watched role id
	roleMembers := []discord.Member{}
	for i := 0; i < len(guildMembers); i++ {
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
		return err
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
	log.Debug("mapped row indices to each member to delete")
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
		return err
	}
	for i := 0; i < len(deleteMembers); i++ {
		_, err = tx.Exec(ctx, `delete from bot.members where member_id=$1`, string(deleteMembers[i].id))
		if err != nil {
			return err
		}
	}
	err = tx.Commit(ctx)
	if err != nil {
		return err
	}
	log.Debugf("deleted %d members from db", len(deleteMembers))

	spreadsheet, err = gsheetsSvc.Spreadsheets.Get(string(id)).IncludeGridData(true).Do()
	if err != nil {
		return err
	}
	if len(filteredDBMembers) == len(roleMembers) && len(filteredDBMembers) == 0 {
		return nil
	}

	// get the members to add to the spreadsheet
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
	log.Debug("got members to add")
	// add the members' rows in the spreadsheet
	requests = make([]*sheets.Request, len(spreadsheet.Sheets))
	for i := 0; i < len(spreadsheet.Sheets); i++ {
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
					UserEnteredFormat: columnMap.Mapping[SheetIndex(i)][0].ColumnFormat,
				},
				{
					UserEnteredValue: &sheets.ExtendedValue{
						StringValue: &username,
					},
					UserEnteredFormat: columnMap.Mapping[SheetIndex(i)][1].ColumnFormat,
				},
			}
			boolVal := false
			numColumns := len(columnMap.Mapping[SheetIndex(i)])
			for k := 0; k < numColumns-2; k++ {
				vals = append(vals, &sheets.CellData{
					UserEnteredFormat: columnMap.Mapping[SheetIndex(i)][ColumnIndex(k+2)].ColumnFormat,
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
			log.Debugf("member %s (id:%s) queued to be added to spreadsheet %d", username, userID, i)
		}
		requests[i] = &sheets.Request{
			AppendCells: &sheets.AppendCellsRequest{
				Fields:  "*",
				SheetId: spreadsheet.Sheets[i].Properties.SheetId,
				Rows:    rowData,
			},
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
		log.Debug("members added to spreadsheet")
	} else {
		log.Debug("members not added to spreadsheet")
	}

	tx, err = dbcon.Begin(ctx)
	if err != nil {
		return err
	}
	// add members to db
	for i := 0; i < len(addMembers); i++ {
		var name string
		if addMembers[i].Nick == nil {
			name = addMembers[i].User.Username
		} else {
			name = *addMembers[i].Nick
		}
		_, err = tx.Exec(ctx, `insert into bot.members(member_id,member_name) values($1,$2)`, addMembers[i].User.ID.String(), name)
		if err != nil {
			return err
		}
	}
	err = tx.Commit(ctx)
	log.Debugf("added %d members to db", len(addMembers))
	return err
}

func xivMountScan() error {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer dbcon.Release()
	// get all members that have xiv character IDs and create requests
	query := `
			select
				member_id,
				member_name,
				member_xiv_id
			from bot.members
			where member_xiv_id is not null
			order by member_name
			`
	rows, err := dbcon.Query(
		ctx,
		query,
	)
	if err != nil {
		return err
	}
	reqMap := map[string]XivCharacterRequest{}
	requests := []XivCharacterRequest{}
	for rows.Next() {
		var memberID string
		var membername string
		var memberXivID string
		err = rows.Scan(&memberID, &membername, &memberXivID)
		if err != nil {
			return err
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
		return err
	}
	if len(xivCharProfiles) == 0 {
		return nil
	}
	// map discord user IDs to xiv character profiles
	profileMap := map[string]XivCharacter{}
	for i := 0; i < len(xivCharProfiles); i++ {
		for discUserID, xivCharRequest := range reqMap {
			if xivCharRequest.XivID == strconv.FormatUint(uint64(xivCharProfiles[i].Character.ID), 10) {
				profileMap[discUserID] = xivCharProfiles[i]
				break
			}
		}
	}
	// get the spreadsheet file id
	var fileID string
	row := dbcon.QueryRow(ctx, `select file_id from bot.file_ref`)
	err = row.Scan(&fileID)
	if err != nil {
		return err
	}
	// get the spreadsheet with all file data
	spreadsheet, err := gsheetsSvc.Spreadsheets.Get(fileID).IncludeGridData(true).Do()
	if err != nil {
		return err
	}
	// get the column format mapping
	columnMap, err := NewColumnMap(mountSpreadsheetColumnDataFilepath)
	if err != nil {
		return err
	}
	// create sheets api requests to update values according
	// to the mount list in the corresponding character profile
	gapiRequests := []*sheets.Request{}
	bossMountMap := getXivBossMountMapping()
	for i := 0; i < len(spreadsheet.Sheets); i++ {
		for memberID, charProfile := range profileMap {
			for j := 1; j < len(spreadsheet.Sheets[i].Data[0].RowData); j++ {
				row := spreadsheet.Sheets[i].Data[0].RowData[j]
				curID := row.Values[0].EffectiveValue.StringValue
				if *curID != memberID {
					continue
				}

				vals := []*sheets.CellData{}
				for k := 2; k < len(row.Values); k++ {
					hasMount := false
					bossName := columnMap.Mapping[SheetIndex(i)][ColumnIndex(k)].Name
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
		log.Debug("Mounts in spreadsheet successfully updated")
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
