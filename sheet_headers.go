package main

import (
	"fmt"
	"strconv"

	"google.golang.org/api/sheets/v4"
)

type ExpansionID string
type ExpansionIndex int
type ExpansionName string

type Expansion struct {
	ID    ExpansionID
	Name  ExpansionName
	Index ExpansionIndex
}

func getExpansions() ([]*Expansion, error) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()
	query := `
		select
			expansion_id,
			expansion_name,
			expansion_index
		from bot.expansion_metadata
	`
	rows, err := dbcon.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("get expansion metadata error: [%w]", err)
	}
	expansions := []*Expansion{}
	for rows.Next() {
		var expansionID string
		var expansionName string
		var expansionIndex int
		err = rows.Scan(&expansionID, &expansionName, &expansionIndex)
		if err != nil {
			return nil, fmt.Errorf("row scan error: [%w]", err)
		}
		expansions = append(expansions,
			&Expansion{
				ID:    ExpansionID(expansionID),
				Name:  ExpansionName(expansionName),
				Index: ExpansionIndex(expansionIndex),
			},
		)
	}
	return expansions, nil
}

func countExpansions() (*int, error) {
	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()
	query := `
		select count(*) as num_expansions
		from bot.expansion_metadata
	`
	row := dbcon.QueryRow(
		ctx,
		query,
	)
	var numExpansions int
	err = row.Scan(&numExpansions)
	if err != nil {
		return nil, fmt.Errorf("row scan error: [%w]", err)
	}
	return &numExpansions, nil
}

type ColumnName string

type ColumnStyleData struct {
	Name         ColumnName
	HeaderFormat *sheets.CellFormat
	ColumnFormat *sheets.CellFormat
}

type SheetIndex int

func (s SheetIndex) String() string {
	return fmt.Sprintf("%d", s)
}

type SheetID int64

func (s SheetID) String() string {
	return fmt.Sprintf("%d", s)
}

type SheetMetadata struct {
	ID    SheetID
	Index SheetIndex
}

type ColumnIndex int

type ColumnMap struct {
	Mapping map[SheetMetadata]map[ColumnIndex]*ColumnStyleData
}

type HeaderBackgroundColor *RGBA
type HeaderForegroundColor *RGBA
type CheckboxBackgroundColor *RGBA
type CheckboxForegroundColor *RGBA

func NewColumnMap() (*ColumnMap, error) {
	numExpansions, err := countExpansions()
	if err != nil {
		return nil, fmt.Errorf("countExpansions() error: [%w]", err)
	}

	dbcon, err := dbpool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("database connection acquire error: [%w]", err)
	}
	defer dbcon.Release()
	query := `
		select
			b.boss_name,
			m.boss_expansion_index,
			e.expansion_index,
			s.sheet_gcp_id,
			s.sheet_index,
			bs.header_background_hex_color,
			bs.header_foreground_hex_color,
			bs.checkbox_background_hex_color,
			bs.checkbox_foreground_hex_color
		from bot.boss_metadata b
		inner join bot.boss_expansion_map m
		on b.boss_id = m.boss_id
		inner join bot.expansion_metadata e
		on e.expansion_id = m.expansion_id
		inner join bot.sheet_expansion_map sm
		on e.expansion_id = sm.expansion_id
		inner join bot.sheet_metadata s
		on s.sheet_gcp_id = sm.sheet_gcp_id
		inner join bot.boss_styling_data bs
		on b.boss_id = bs.boss_id
	`
	rows, err := dbcon.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("get style data error: [%w]", err)
	}

	innerMap := map[SheetMetadata]map[ColumnIndex]*ColumnStyleData{}
	columnDataMaps := make([]map[ColumnIndex]*ColumnStyleData, *numExpansions)
	for rows.Next() {
		var bossName string
		var bossExpansionIndex int
		var expansionIndex int
		var sheetIdStr string
		var sheetIndex int
		var headerBackgroundHex string
		var headerForegroundHex string
		var checkboxBackgroundHex string
		var checkboxForegroundHex string
		err = rows.Scan(
			&bossName,
			&bossExpansionIndex,
			&expansionIndex,
			&sheetIdStr,
			&sheetIndex,
			&headerBackgroundHex,
			&headerForegroundHex,
			&checkboxBackgroundHex,
			&checkboxForegroundHex,
		)
		if err != nil {
			return nil, fmt.Errorf("row scan error error: [%w]", err)
		}

		sheetId, err := strconv.ParseInt(sheetIdStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("strconv.ParseInt() error: [%w]", err)
		}

		if !isHex(headerBackgroundHex) {
			return nil, fmt.Errorf("%s is not a valid header background hex color for %s", headerBackgroundHex, bossName)
		}
		if !isHex(headerForegroundHex) {
			return nil, fmt.Errorf("%s is not a valid header foreground hex color for %s", headerForegroundHex, bossName)
		}
		if !isHex(checkboxBackgroundHex) {
			return nil, fmt.Errorf("%s is not a valid checkbox background hex color for %s", checkboxBackgroundHex, bossName)
		}
		if !isHex(checkboxForegroundHex) {
			return nil, fmt.Errorf("%s is not a valid checkbox background hex color for %s", checkboxForegroundHex, bossName)
		}

		headerBackgroundRgba, err := hex2rgba(headerBackgroundHex)
		if err != nil {
			return nil, fmt.Errorf("hex2rgba() 1 error: [%w]", err)
		}
		headerForegroundRgba, err := hex2rgba(headerForegroundHex)
		if err != nil {
			return nil, fmt.Errorf("hex2rgba() 2 error: [%w]", err)
		}
		checkboxBackgroundRgba, err := hex2rgba(checkboxBackgroundHex)
		if err != nil {
			return nil, fmt.Errorf("hex2rgba() 3 error: [%w]", err)
		}
		checkboxForegroundRgba, err := hex2rgba(checkboxForegroundHex)
		if err != nil {
			return nil, fmt.Errorf("hex2rgba() 4 error: [%w]", err)
		}

		headerCellFormat := &sheets.CellFormat{
			TextFormat: &sheets.TextFormat{
				Bold: true,
				ForegroundColorStyle: &sheets.ColorStyle{
					RgbColor: headerForegroundRgba.ToGoogleSheetsColor(),
				},
			},
			BackgroundColorStyle: &sheets.ColorStyle{
				RgbColor: headerBackgroundRgba.ToGoogleSheetsColor(),
			},
		}
		columnFormat := &sheets.CellFormat{
			TextFormat: &sheets.TextFormat{
				ForegroundColorStyle: &sheets.ColorStyle{
					RgbColor: checkboxForegroundRgba.ToGoogleSheetsColor(),
				},
			},
			BackgroundColorStyle: &sheets.ColorStyle{
				RgbColor: checkboxBackgroundRgba.ToGoogleSheetsColor(),
			},
		}

		oldCdataMap := columnDataMaps[expansionIndex]
		cdata := &ColumnStyleData{
			Name:         ColumnName(bossName),
			HeaderFormat: headerCellFormat,
			ColumnFormat: columnFormat,
		}
		cindex := ColumnIndex(bossExpansionIndex + 2)
		if oldCdataMap == nil {
			columnDataMaps[expansionIndex] = make(map[ColumnIndex]*ColumnStyleData)
			columnDataMaps[expansionIndex][cindex] = cdata
		} else {
			oldCdataMap[ColumnIndex(cindex)] = cdata
			columnDataMaps[expansionIndex] = oldCdataMap
		}

		innerMap[SheetMetadata{
			ID:    SheetID(sheetId),
			Index: SheetIndex(sheetIndex),
		}] = columnDataMaps[expansionIndex]
	}

	mapping := &ColumnMap{Mapping: innerMap}

	headerBackgroundHex := "#d9d9d9"
	headerBackgroundRgba, err := hex2rgba(headerBackgroundHex)
	if err != nil {
		return nil, fmt.Errorf("hex2rgba() 5 error: [%w]", err)
	}
	headerForegroundHex := "#4d4d4d"
	headerForegroundRgba, err := hex2rgba(headerForegroundHex)
	if err != nil {
		return nil, fmt.Errorf("hex2rgba() 6 error: [%w]", err)
	}
	columnBackgroundHex := "#e6e6e6"
	columnBackgroundRgba, err := hex2rgba(columnBackgroundHex)
	if err != nil {
		return nil, fmt.Errorf("hex2rgba() 7 error: [%w]", err)
	}
	columnForegroundHex := "#595959"
	columnForegroundRgba, err := hex2rgba(columnForegroundHex)
	if err != nil {
		return nil, fmt.Errorf("hex2rgba() 8 error: [%w]", err)
	}
	headerCellFormat := &sheets.CellFormat{
		TextFormat: &sheets.TextFormat{
			Bold: true,
			ForegroundColorStyle: &sheets.ColorStyle{
				RgbColor: headerForegroundRgba.ToGoogleSheetsColor(),
			},
		},
		BackgroundColorStyle: &sheets.ColorStyle{
			RgbColor: headerBackgroundRgba.ToGoogleSheetsColor(),
		},
	}
	columnFormat := &sheets.CellFormat{
		TextFormat: &sheets.TextFormat{
			ForegroundColorStyle: &sheets.ColorStyle{
				RgbColor: columnForegroundRgba.ToGoogleSheetsColor(),
			},
		},
		BackgroundColorStyle: &sheets.ColorStyle{
			RgbColor: columnBackgroundRgba.ToGoogleSheetsColor(),
		},
	}
	// add common column headings to every sheet
	for sheet := range mapping.Mapping {
		mapping.Mapping[sheet][0] = &ColumnStyleData{
			Name:         ColumnName("Row ID"),
			HeaderFormat: headerCellFormat,
			ColumnFormat: columnFormat,
		}
		mapping.Mapping[sheet][1] = &ColumnStyleData{
			Name:         ColumnName("Name"),
			HeaderFormat: headerCellFormat,
			ColumnFormat: columnFormat,
		}
	}
	return mapping, nil
}
