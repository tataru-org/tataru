package main

import (
	"database/sql"
	"fmt"
	"runtime/debug"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/api/sheets/v4"
)

func sendEventErrorResponse(event *events.ApplicationCommandInteractionCreate, err error) {
	trace := string(debug.Stack())
	e := event.CreateMessage(
		discord.MessageCreate{
			Content: fmt.Sprintf("Error while setting role; report this to one of the developers.\nerror: %s\nstack trace: %s", err.Error(), trace),
			Flags:   discord.MessageFlagEphemeral,
		},
	)
	if e != nil {
		log.Fatal(e)
		log.Fatal(debug.Stack())
	}
}

func buildFile(dbcon *pgxpool.Conn, badFileExists bool) (*FileID, error) {
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
	perms, err := GetPermissions(mountSpreadsheetPermissionsFilepath)
	if err != nil {
		return nil, err
	}
	permIDs := make([]PermissionID, len(perms))
	for i := 0; i < len(perms); i++ {
		pid, err := addFilePermmission(*fileID, perms[i])
		if err != nil {
			return nil, err
		}
		permIDs[i] = *pid
		log.Debugf(
			"permission added for: id=%s;email=%s;role=%s;type=%s",
			permIDs[i],
			perms[i].EmailAddress,
			perms[i].Role,
			perms[i].Type,
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
		requests = append(requests, &sheets.Request{
			AddProtectedRange: &sheets.AddProtectedRangeRequest{
				ProtectedRange: &sheets.ProtectedRange{
					Range: &sheets.GridRange{
						SheetId:       int64(sheetMap[Expansion(sheetIndex)]),
						StartRowIndex: 0,
						EndRowIndex:   1,
					},
					RequestingUserCanEdit: false,
					WarningOnly:           false,
				},
			},
		})
		requests = append(requests, &sheets.Request{
			AddProtectedRange: &sheets.AddProtectedRangeRequest{
				ProtectedRange: &sheets.ProtectedRange{
					Range: &sheets.GridRange{
						SheetId:          int64(sheetMap[Expansion(sheetIndex)]),
						StartColumnIndex: 0,
						EndColumnIndex:   2,
					},
					RequestingUserCanEdit: false,
					WarningOnly:           false,
				},
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
	_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return nil, err
	}
	log.Debug("header rows and protected ranges added to each sheet")

	// get members from db
	members, err := getMembersFromDB(dbcon)
	if err != nil {
		return nil, err
	}
	// add members to each sheet
	requests = []*sheets.Request{}
	for sheetIndex, columnIndexMap := range columnMap.Mapping {
		numColumns := len(columnIndexMap)
		cellData := make([]*sheets.CellData, numColumns)
		for columnIndex, columnData := range columnIndexMap {
			for i := 0; i < len(members); i++ {
				if columnIndex == 0 {
					uid := string(members[i].id)
					cellData[columnIndex] = &sheets.CellData{
						UserEnteredValue: &sheets.ExtendedValue{
							StringValue: &uid,
						},
						UserEnteredFormat: columnData.ColumnFormat,
					}
				} else if columnIndex == 1 {
					cellData[columnIndex] = &sheets.CellData{
						UserEnteredValue: &sheets.ExtendedValue{
							StringValue: &members[i].name,
						},
						UserEnteredFormat: columnData.ColumnFormat,
					}
				} else {
					boolVal := false
					cellData[columnIndex] = &sheets.CellData{
						DataValidation: &sheets.DataValidationRule{
							Condition: &sheets.BooleanCondition{
								Type: "BOOLEAN",
							},
						},
						UserEnteredValue: &sheets.ExtendedValue{
							BoolValue: &boolVal,
						},
						UserEnteredFormat: columnData.ColumnFormat,
					}
				}
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
	_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return nil, err
	}
	log.Debug("members added to each sheet")

	// save what is needed to the db
	queryBatch := &pgx.Batch{}
	if badFileExists {
		// delete data in file id table
		queryBatch.Queue(`delete from bot.file_ref`)
	}
	// put file id into db
	queryBatch.Queue(`insert into bot.file_ref(file_id) values($1)`, fileID)
	// put perm ids into db
	for i := 0; i < len(permIDs); i++ {
		queryBatch.Queue(`insert into bot.permissions(file_id,perm_id) values($1,$2)`, fileID, permIDs[i])
	}
	// put sheet IDs into db
	for exp, sheetID := range sheetMap {
		sheetIDStr := strconv.FormatInt(int64(sheetID), 10)
		queryBatch.Queue(`insert into bot.sheets(file_id,expansion,sheet_id) values($1,$2,$3)`, fileID, int(exp), sheetIDStr)
	}

	bresults := dbcon.SendBatch(ctx, queryBatch)
	for i := 0; i < queryBatch.Len(); i++ {
		_, err = bresults.Exec()
		if err != nil {
			return nil, err
		}
	}
	log.Debug("required data saved to db")
	return fileID, nil
}

func syncRoleMembers(dbcon *pgxpool.Conn, id FileID, guildMembers []discord.Member) {
	// get members from db
	dbMembers, err := getMembersFromDB(dbcon)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	// get the watched role id
	var roleID string
	row := dbcon.QueryRow(ctx, `select role_id from bot.role_ref`)
	err = row.Scan(&roleID)
	if err == sql.ErrNoRows {
		// exit if role is not set
		return
	} else if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}

	// filter out members without the watched role id
	roleMembers := []discord.Member{}
	for i := 0; i < len(guildMembers); i++ {
		for j := 0; j < len(guildMembers[i].RoleIDs); j++ {
			if guildMembers[i].RoleIDs[j].String() == roleID {
				roleMembers[i] = guildMembers[i]
				break
			}
		}
	}

	spreadsheet, err := gsheetsSvc.Spreadsheets.Get(string(id)).IncludeGridData(true).Do()
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	if len(dbMembers) == len(roleMembers) && len(dbMembers) == 0 {
		return
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
	// delete the members' rows in the spreadsheet
	requests := make([]*sheets.Request, len(deleteMemberMap)*len(spreadsheet.Sheets))
	requestIndex := 0
	for i := 0; i < len(spreadsheet.Sheets); i++ {
		for rowIndex, member := range deleteMemberMap {
			requests[requestIndex] = &sheets.Request{
				DeleteRange: &sheets.DeleteRangeRequest{
					Range: &sheets.GridRange{
						StartRowIndex: rowIndex,
						EndRowIndex:   rowIndex,
						SheetId:       spreadsheet.Sheets[i].Properties.SheetId,
					},
					ShiftDimension: "ROWS",
				},
			}
			log.Debugf("member %s (id:%s) queued to be deleted from spreadsheet", member.name, string(member.id))
			requestIndex++
		}
	}
	_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	log.Debug("members deleted from spreadsheet")

	// delete members from the db
	qBatch := &pgx.Batch{}
	for i := 0; i < len(deleteMembers); i++ {
		qBatch.Queue(`delete from bot.members where member_id=$1`, string(deleteMembers[i].id))
	}
	bresults := dbcon.SendBatch(ctx, qBatch)
	for i := 0; i < qBatch.Len(); i++ {
		_, err = bresults.Exec()
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
	}
	log.Debug("members deleted from db")

	spreadsheet, err = gsheetsSvc.Spreadsheets.Get(string(id)).IncludeGridData(true).Do()
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	if len(filteredDBMembers) == len(roleMembers) && len(filteredDBMembers) == 0 {
		return
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
	// map the row indices of each member to add
	testSheet = spreadsheet.Sheets[0]
	numRows = len(testSheet.Data[0].RowData) - 1
	addMemberMap := map[int64]discord.Member{}
	for i := 0; i < len(addMembers); i++ {
		for j := 0; j < numRows; j++ {
			rowIndex := j + 1
			row := testSheet.Data[0].RowData[rowIndex]
			if *row.Values[0].EffectiveValue.StringValue == addMembers[i].User.ID.String() {
				addMemberMap[int64(rowIndex)] = addMembers[i]
			}
		}
	}
	// add the members' rows in the spreadsheet
	requests = make([]*sheets.Request, len(addMemberMap)*len(spreadsheet.Sheets))
	requestIndex = 0
	for i := 0; i < len(spreadsheet.Sheets); i++ {
		for rowIndex, member := range addMemberMap {
			requests[requestIndex] = &sheets.Request{
				InsertRange: &sheets.InsertRangeRequest{
					Range: &sheets.GridRange{
						StartRowIndex: rowIndex,
						EndRowIndex:   rowIndex,
						SheetId:       spreadsheet.Sheets[i].Properties.SheetId,
					},
					ShiftDimension: "ROWS",
				},
			}
			log.Debugf("member %s (id:%s) queued to be added to spreadsheet", *member.Nick, member.User.ID.String())
			requestIndex++
		}
	}
	_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	log.Debug("members added to spreadsheet")
}
