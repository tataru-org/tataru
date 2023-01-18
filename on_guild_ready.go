package main

import (
	"fmt"
	"strconv"

	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
	"github.com/jackc/pgx/v5"
	"google.golang.org/api/sheets/v4"
)

func buildFile(countIsGreaterThan0 bool) {
	fileID, err := createFile(botConfig.MountSpreadsheetFileName)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	log.Debug("file created")
	columnMap, err := NewColumnMap(botConfig.MountSpreadsheetColumnDataFilepath)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	sheetNames, err := columnMap.GetSheetNames()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	// add permissions to the file
	perms, err := GetPermissions(botConfig.MountSpreadsheetPermissionsFilepath)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	permIDs := make([]PermissionID, len(perms))
	for i := 0; i < len(perms); i++ {
		pid, err := addFilePermmission(*fileID, perms[i])
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
		permIDs[i] = *pid
		log.Debug(
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
		panic(err)
	}
	// create the sheets
	numSheets := len(sheetNames)
	addSheetReqs := make([]*sheets.Request, numSheets)
	for i := 0; i < numSheets; i++ {
		addSheetReqs[i] = &sheets.Request{
			AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{
					Index: int64(i),
					Title: sheetNames[i],
				},
			},
		}
	}
	// the sheets api docs state that some replies may be empty, so do not rely on the response to
	// get the sheet IDs from the spreadsheet
	_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: addSheetReqs,
	}).Do()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	log.Debug("sheets created")
	// map expansions to sheet IDs
	spreadsheet, err = gsheetsSvc.Spreadsheets.Get(spreadsheet.SpreadsheetId).Do()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	sheetMap := make(map[Expansion]SheetID, numSheets)
	for i := 0; i < numSheets; i++ {
		sheet := spreadsheet.Sheets[i]
		exp, err := ExpansionNameToExpansion(ExpansionName(sheet.Properties.Title))
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
		sheetMap[exp] = SheetID(sheet.Properties.SheetId)
	}
	// add the header row to each sheet & add protected ranges
	requests := []*sheets.Request{}
	for sheetIndex, columnIndexMap := range columnMap.Mapping {
		numColumns := len(columnIndexMap)
		cellData := make([]*sheets.CellData, numColumns)
		for columnIndex, columnData := range columnIndexMap {
			expNameExp, err := ExpansionToExpansionName(Expansion(sheetIndex))
			if err != nil {
				log.Fatal(err)
				panic(err)
			}
			expName := string(expNameExp)
			cellData[columnIndex] = &sheets.CellData{
				UserEnteredValue: &sheets.ExtendedValue{
					StringValue: &expName,
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
	_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	log.Debug("header rows and protected ranges added to each sheet")

	// get users from db
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	defer dbcon.Conn().Close(ctx)
	query := `
		select
			user_id,
			user_name
		from bot.users
		order by user_name
	`
	rows, err := dbcon.Query(
		ctx,
		query,
	)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	users := []struct {
		id   string
		name string
	}{}
	for rows.Next() {
		var userID string
		var username string
		err = rows.Scan(&userID, &username)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
		user := struct {
			id   string
			name string
		}{}
		users = append(users, user)
	}
	// add users to each sheet
	requests = []*sheets.Request{}
	for sheetIndex, columnIndexMap := range columnMap.Mapping {
		numColumns := len(columnIndexMap)
		cellData := make([]*sheets.CellData, numColumns)
		for columnIndex, columnData := range columnIndexMap {
			for i := 0; i < len(users); i++ {
				if columnIndex == 0 {
					cellData[columnIndex] = &sheets.CellData{
						UserEnteredValue: &sheets.ExtendedValue{
							StringValue: &users[i].id,
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
		panic(err)
	}
	log.Debug("users added to each sheet")

	// delete the default sheet
	var defaultSheetID int64 = 0
	for i := 0; i < len(spreadsheet.Sheets); i++ {
		isInSheetMap := false
		for _, sheetID := range sheetMap {
			if spreadsheet.Sheets[i].Properties.SheetId == int64(sheetID) {
				isInSheetMap = true
			}
		}
		if !isInSheetMap {
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
		panic(err)
	}
	log.Debug("default sheet deleted")

	// save what is needed to the db
	queryBatch := &pgx.Batch{}
	if countIsGreaterThan0 {
		// truncate file id table
		queryBatch.Queue(`truncate table bot.file_ref`)
	}
	// put file id into db
	queryBatch.Queue(`insert into bot.file_ref(file_id=$1)`, fileID)
	// put perm ids into db
	for i := 0; i < len(permIDs); i++ {
		queryBatch.Queue(`insert into bot.permissions(file_id=$1,permission_id=$2)`, fileID, permIDs[i])
	}
	// put sheet IDs into db
	for exp, sheetID := range sheetMap {
		sheetIDStr := strconv.FormatInt(int64(sheetID), 10)
		queryBatch.Queue(`insert into bot.sheets(file_id=$1,expansion=$2,sheet_id=$3)`, fileID, int(exp), sheetIDStr)
	}
	dbcon.SendBatch(ctx, queryBatch)
	log.Debug("required data saved to db")
}

func onGuildReady(event *events.GuildReady) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	defer dbcon.Conn().Close(ctx)
	isValidDb, err := isValidDatabase(dbcon)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	if isValidDb {
		log.Debug("schema is valid")
	} else {
		log.Debug("schema is invalid")
		initSchema(dbcon)
		log.Debug("schema initialized")
	}

	// check if db has a record of the file
	var fileRefExists bool
	row := dbcon.QueryRow(
		ctx,
		"select exists(select file_id from bot.file_ref)",
	)
	err = row.Scan(&fileRefExists)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	members, err := event.Client().Rest().GetMembers(event.GuildID)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	if fileRefExists {
		// check if the file exists in google drive
		var fileID string
		row := dbcon.QueryRow(
			ctx,
			"select file_id from bot.file_ref",
		)
		err = row.Scan(&fileID)
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
		ok, err := fileExists(FileID(fileID))
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
		if *ok {
			// check if the file needs to be updated
		} else {
			// create the file
			log.Debug("file exists in db on startup but does not exist in google drive")
			buildFile(true)
			log.Debug("file built")
		}
	} else {
		// create the file
		log.Debug("file does not exist in db on startup")
		buildFile(false)
		log.Debug("file built")
	}
	fmt.Println(members)
}
