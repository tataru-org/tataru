package main

import (
	"fmt"
	"strconv"

	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
	"google.golang.org/api/sheets/v4"
)

func buildFile(countIsGreaterThan0 bool) {
	fileID, err := createFile(botConfig.MountSpreadsheetFileName)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
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
	for i := 0; i < len(perms); i++ {
		_, err = addFilePermmission(*fileID, perms[i])
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
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

	// DO THIS TOMORROW - CHECK DB FOR USERS AND ADD THEM TO EACH SHEET

	// put file id into db
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	defer dbcon.Conn().Close(ctx)
	if countIsGreaterThan0 {
		truncateTable(dbcon, "schema.table")
		if err != nil {
			log.Fatal(err)
			panic(err)
		}
	}
	query := `insert into schema.table(file_id=$1)`
	_, err = dbcon.Exec(
		ctx,
		query,
		fileID,
	)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
}

func onGuildReady(event *events.GuildReady) {
	client := event.Client()

	// check if db has a record of the file
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	defer dbcon.Conn().Close(ctx)

	var countStr string
	row := dbcon.QueryRow(
		ctx,
		"select count(*) from schema.table",
	)
	err = row.Scan(&countStr)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	members, err := client.Rest().GetMembers(event.GuildID)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	if count == 0 {
		// create the file
		log.Debug("file does not exist on startup")
	} else if count == 1 {
		// check if the file exists
		var fileID string
		row := dbcon.QueryRow(
			ctx,
			"select file_id from schema.table",
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
		}
	} else {
		err := fmt.Errorf("multiple files exist")
		log.Fatal(err)
		panic(err)
	}
	fmt.Println(members)
}
