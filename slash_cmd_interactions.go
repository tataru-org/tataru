package main

import (
	"fmt"
	"runtime/debug"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/log"
	"google.golang.org/api/sheets/v4"
)

func setRoleHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "set_role" {
		return
	}
	// check if a role ref exists
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	defer dbcon.Release()
	row := dbcon.QueryRow(ctx, `select count(*) > 0 from bot.role_ref`)
	var hasRoleID bool
	err = row.Scan(&hasRoleID)
	if err != nil {

		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	if hasRoleID {
		err = event.CreateMessage(
			discord.MessageCreate{
				Content: "A role is already set.",
				Flags:   discord.MessageFlagEphemeral,
			},
		)
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
	} else {
		// set role ref
		role := eventData.Role("role")
		_, err = dbcon.Exec(ctx, `insert into bot.role_ref(role_id) values($1)`, role.ID.String())
		if err != nil {

			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
		err = event.CreateMessage(
			discord.MessageCreate{
				Content: fmt.Sprintf("Role %s has been set.", role.Name),
				Flags:   discord.MessageFlagEphemeral,
			},
		)
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
		}
	}
}

func unsetRoleHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "unset_role" {
		return
	}
	// check if a role ref exists
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	defer dbcon.Release()
	row := dbcon.QueryRow(ctx, `select count(*) > 0 from bot.role_ref`)
	var hasRoleID bool
	err = row.Scan(&hasRoleID)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	if hasRoleID {
		// unset role ref
		_, err = dbcon.Exec(ctx, `truncate table bot.role_ref`)
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
		err = event.CreateMessage(
			discord.MessageCreate{
				Content: "Role has been unset.",
				Flags:   discord.MessageFlagEphemeral,
			},
		)
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
	} else {
		err = event.CreateMessage(
			discord.MessageCreate{
				Content: "Unable to unset role; a role ref has not been set.",
				Flags:   discord.MessageFlagEphemeral,
			},
		)
		if err != nil {
			log.Fatal(err)
			log.Fatal(debug.Stack())
			return
		}
	}
}

func forceSyncHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "force_sync" {
		return
	}
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	defer dbcon.Release()

	// get file id
	var fileID string
	row := dbcon.QueryRow(
		ctx,
		"select file_id from bot.file_ref",
	)
	err = row.Scan(&fileID)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	dbcon.Release()
	// get the discord members
	members, err := event.Client().Rest().GetMembers(*event.GuildID(), guildMemberCountRequestLimit, nullSnowflake)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	// sync the spreadsheet with the discord members
	err = syncRoleMembers(FileID(fileID), members)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	log.Debug("force sync successfully completed")
}

func syncFormattingHandler(event *events.ApplicationCommandInteractionCreate) {
	eventData := event.SlashCommandInteractionData()
	if eventData.CommandName() != "sync_formatting" {
		return
	}

	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	defer dbcon.Release()
	// get file id
	var fileID *string
	row := dbcon.QueryRow(
		ctx,
		"select file_id from bot.file_ref",
	)
	err = row.Scan(&fileID)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	dbcon.Release()
	if fileID == nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}

	columnMap, err := NewColumnMap(mountSpreadsheetColumnDataFilepath)
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	spreadsheet, err := gsheetsSvc.Spreadsheets.Get(*fileID).IncludeGridData(true).Do()
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}

	// make requests for the header rows
	requests := make([]*sheets.Request, len(spreadsheet.Sheets)*2)
	for i := 0; i < len(spreadsheet.Sheets); i++ {
		sheet := spreadsheet.Sheets[i]
		row := sheet.Data[0].RowData[0]
		vals := make([]*sheets.CellData, len(row.Values))
		for k := 0; k < len(row.Values); k++ {
			colName := string(columnMap.Mapping[SheetIndex(i)][ColumnIndex(k)].Name)
			vals[k] = &sheets.CellData{
				UserEnteredFormat: columnMap.Mapping[SheetIndex(i)][ColumnIndex(k)].HeaderFormat,
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
					UserEnteredFormat: columnMap.Mapping[SheetIndex(i)][ColumnIndex(k)].ColumnFormat,
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
	_, err = gsheetsSvc.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
	log.Debug("formatting successfully synced")
	err = event.CreateMessage(discord.MessageCreate{
		Content: "Formatting successfully synced",
		Flags:   discord.MessageFlagEphemeral,
	})
	if err != nil {
		log.Fatal(err)
		log.Fatal(debug.Stack())
		return
	}
}
