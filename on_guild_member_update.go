package main

import (
	"runtime/debug"

	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
	"google.golang.org/api/sheets/v4"
)

func onGuildMemberUpdateHandler(event *events.GuildMemberUpdate) {
	if event.Member.User.Bot {
		return
	}

	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Error(err)
		log.Error(debug.Stack())
		return
	}
	defer dbcon.Release()

	// get the watched role
	var roleID *string
	row := dbcon.QueryRow(ctx, `select role_id from bot.role_ref`)
	err = row.Scan(&roleID)
	if err != nil {
		log.Error(err)
		log.Error(debug.Stack())
		return
	}
	if roleID == nil {
		log.Error(err)
		log.Error(debug.Stack())
		return
	}

	// determine to continue
	oldMemberHasRole := false
	for i := 0; i < len(event.OldMember.RoleIDs); i++ {
		if event.OldMember.RoleIDs[i].String() == *roleID {
			oldMemberHasRole = true
		}
	}
	newMemberHasRole := false
	for i := 0; i < len(event.Member.RoleIDs); i++ {
		if event.Member.RoleIDs[i].String() == *roleID {
			newMemberHasRole = true
		}
	}
	if oldMemberHasRole && newMemberHasRole {
		return
	}

	// get file id
	var fileID *string
	row = dbcon.QueryRow(
		ctx,
		"select file_id from bot.file_ref",
	)
	err = row.Scan(&fileID)
	if err != nil {
		log.Error(err)
		log.Error(debug.Stack())
		return
	}
	if fileID == nil {
		log.Error(err)
		log.Error(debug.Stack())
		return
	}

	// get column formatting
	columnMap, err := NewColumnMap(mountSpreadsheetColumnDataFilepath)
	if err != nil {
		log.Error(err)
		log.Error(debug.Stack())
		return
	}

	// get the spreadsheet
	spreadsheet, err := gsheetsSvc.Spreadsheets.Get(*fileID).IncludeGridData(true).Do()
	if err != nil {
		log.Error(err)
		log.Error(debug.Stack())
		return
	}

	userID := event.Member.User.ID.String()
	var username string
	if event.Member.Nick == nil {
		username = event.Member.User.Username
	} else {
		username = *event.Member.Nick
	}
	if !oldMemberHasRole && newMemberHasRole {
		// add the member to the spreadsheet
		requests := make([]*sheets.Request, len(spreadsheet.Sheets))
		for i := 0; i < len(spreadsheet.Sheets); i++ {
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
			requests[i] = &sheets.Request{
				AppendCells: &sheets.AppendCellsRequest{
					Fields:  "*",
					SheetId: spreadsheet.Sheets[i].Properties.SheetId,
					Rows: []*sheets.RowData{
						{
							Values: vals,
						},
					},
				},
			}
		}
		_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: requests,
		}).Do()
		if err != nil {
			log.Error(err)
			log.Error(debug.Stack())
			return
		}
		log.Debugf("member %s (id:%s) added to spreadsheet", username, userID)

		_, err = dbcon.Exec(ctx, `insert into bot.members(member_id,member_name) values($1,$2)`, userID, username)
		if err != nil {
			log.Error(err)
			log.Error(debug.Stack())
			return
		}
		log.Debugf("member %s (id:%s) added to db", username, userID)
	} else {
		// delete the member from the spreadsheet

		// map the row indices of each member to delete
		var rowIndex *int64 = nil
		testSheet := spreadsheet.Sheets[0]
		numRows := len(testSheet.Data[0].RowData) - 1
		for j := 0; j < numRows; j++ {
			index := int64(j + 1)
			row := testSheet.Data[0].RowData[index]
			if *row.Values[0].EffectiveValue.StringValue == userID {
				rowIndex = &index
				break
			}
		}
		log.Debug("mapped row indices of member to delete")

		// delete the members' rows in the spreadsheet
		requests := make([]*sheets.Request, len(spreadsheet.Sheets))
		for i := 0; i < len(spreadsheet.Sheets); i++ {
			requests[i] = &sheets.Request{
				DeleteRange: &sheets.DeleteRangeRequest{
					Range: &sheets.GridRange{
						StartRowIndex: *rowIndex,
						EndRowIndex:   *rowIndex + 1,
						SheetId:       spreadsheet.Sheets[i].Properties.SheetId,
					},
					ShiftDimension: "ROWS",
				},
			}
		}
		_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: requests,
		}).Do()
		if err != nil {
			log.Error(err)
			log.Error(debug.Stack())
			return
		}
		log.Debugf("member %s (id:%s) deleted from spreadsheet", username, userID)

		_, err = dbcon.Exec(ctx, `delete from bot.members where member_id=$1`, userID)
		if err != nil {
			log.Error(err)
			log.Error(debug.Stack())
			return
		}
		log.Debugf("member %s (id:%s) deleted from db", username, userID)
	}
}
