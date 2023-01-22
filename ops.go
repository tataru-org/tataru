package main

import (
	"fmt"
	"runtime/debug"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
	"github.com/jackc/pgx/v5"
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

func buildFile(badFileExists bool) {
	fileID, err := createFile(botConfig.MountSpreadsheetFileName)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	log.Debugf("file created: %s", *fileID)
	columnMap, err := NewColumnMap(mountSpreadsheetColumnDataFilepath)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	sheetNames, err := columnMap.GetSheetNames()
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}

	// add permissions to the file
	perms, err := GetPermissions(mountSpreadsheetPermissionsFilepath)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	permIDs := make([]PermissionID, len(perms))
	for i := 0; i < len(perms); i++ {
		pid, err := addFilePermmission(*fileID, perms[i])
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
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
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
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
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
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
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	log.Debug("default sheet deleted")

	// map expansions to sheet IDs
	spreadsheet, err = gsheetsSvc.Spreadsheets.Get(spreadsheet.SpreadsheetId).Do()
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
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
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	log.Debug("header rows and protected ranges added to each sheet")

	// get users from db
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	defer dbcon.Conn().Close(ctx)
	users, err := getUsersFromDB(dbcon)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	// add users to each sheet
	requests = []*sheets.Request{}
	for sheetIndex, columnIndexMap := range columnMap.Mapping {
		numColumns := len(columnIndexMap)
		cellData := make([]*sheets.CellData, numColumns)
		for columnIndex, columnData := range columnIndexMap {
			for i := 0; i < len(users); i++ {
				if columnIndex == 0 {
					uid := string(users[i].id)
					cellData[columnIndex] = &sheets.CellData{
						UserEnteredValue: &sheets.ExtendedValue{
							StringValue: &uid,
						},
						UserEnteredFormat: columnData.ColumnFormat,
					}
				} else if columnIndex == 1 {
					cellData[columnIndex] = &sheets.CellData{
						UserEnteredValue: &sheets.ExtendedValue{
							StringValue: &users[i].name,
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
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	log.Debug("users added to each sheet")

	// save what is needed to the db
	queryBatch := &pgx.Batch{}
	if badFileExists {
		// truncate file id table
		queryBatch.Queue(`truncate table bot.file_ref`)
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
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
	}
	log.Debug("required data saved to db")
}

func syncRoleMembers(id FileID, guildMembers []discord.Member) {
	// get users from db
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	defer dbcon.Conn().Close(ctx)
	// TODO
	_, err = getUsersFromDB(dbcon)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
}
