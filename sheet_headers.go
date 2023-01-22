package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	"google.golang.org/api/sheets/v4"
)

type Expansion int

const (
	ExpansionARR Expansion = iota
	ExpansionHW
	ExpansionSTB
	ExpansionSHB
	ExpansionEW
)

type ExpansionName string

const (
	ExpansionNameARR ExpansionName = "A Real Reborn"
	ExpansionNameHW  ExpansionName = "Heavensward"
	ExpansionNameSTB ExpansionName = "Stormblood"
	ExpansionNameSHB ExpansionName = "Shadowbringers"
	ExpansionNameEW  ExpansionName = "Endwalker"
)

func ExpansionToExpansionName(exp Expansion) (ExpansionName, error) {
	switch exp {
	case ExpansionARR:
		{
			return ExpansionNameARR, nil
		}
	case ExpansionHW:
		{
			return ExpansionNameHW, nil
		}
	case ExpansionSTB:
		{
			return ExpansionNameSTB, nil
		}
	case ExpansionSHB:
		{
			return ExpansionNameSHB, nil
		}
	case ExpansionEW:
		{
			return ExpansionNameEW, nil
		}
	default:
		{
			return "", fmt.Errorf("%d is not an implemented expansion index", exp)
		}
	}
}

func ExpansionNameToExpansion(expName ExpansionName) (Expansion, error) {
	switch expName {
	case ExpansionNameARR:
		{
			return ExpansionARR, nil
		}
	case ExpansionNameHW:
		{
			return ExpansionHW, nil
		}
	case ExpansionNameSTB:
		{
			return ExpansionSTB, nil
		}
	case ExpansionNameSHB:
		{
			return ExpansionSHB, nil
		}
	case ExpansionNameEW:
		{
			return ExpansionEW, nil
		}
	default:
		{
			return -1, fmt.Errorf("%s is not an implemented expansion name", expName)
		}
	}
}

func getExpansions() []Expansion {
	return []Expansion{
		ExpansionARR,
		ExpansionHW,
		ExpansionSTB,
		ExpansionSHB,
		ExpansionEW,
	}
}

func getExpansionNames() []ExpansionName {
	return []ExpansionName{
		ExpansionNameARR,
		ExpansionNameHW,
		ExpansionNameSTB,
		ExpansionNameSHB,
		ExpansionNameEW,
	}
}

const (
	NumberOfExpansions int = 5
)

type ColumnData struct {
	Name         ColumnName
	HeaderFormat *sheets.CellFormat
	ColumnFormat *sheets.CellFormat
}

type SheetIndex int
type ColumnIndex int
type ColumnName string

type ColumnMap struct {
	Mapping map[SheetIndex]map[ColumnIndex]*ColumnData
}

func NewColumnMap(headerDataFilepath string) (*ColumnMap, error) {
	file, err := os.Open(headerDataFilepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// skip header row
	_, err = reader.Read()
	if err != nil {
		return nil, err
	}

	innerMap := map[SheetIndex]map[ColumnIndex]*ColumnData{}
	columnDataMaps := make([]map[ColumnIndex]*ColumnData, NumberOfExpansions)
	// read everything else
	for {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		columnIndexAfterCommonIndices, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			return nil, err
		}
		bossName := row[1]
		expansion, err := strconv.ParseInt(row[2], 10, 64)
		if err != nil {
			return nil, err
		}
		headerBackgroundHex := row[3]
		if !isHex(headerBackgroundHex) {
			return nil, fmt.Errorf("%s is not a valid header background hex color for %s", headerBackgroundHex, bossName)
		}
		headerForegroundHex := row[4]
		if !isHex(headerForegroundHex) {
			return nil, fmt.Errorf("%s is not a valid header foreground hex color for %s", headerForegroundHex, bossName)
		}
		checkboxBackgroundHex := row[5]
		if !isHex(checkboxBackgroundHex) {
			return nil, fmt.Errorf("%s is not a valid checkbox background hex color for %s", checkboxBackgroundHex, bossName)
		}
		checkboxForegroundHex := row[6]
		if !isHex(checkboxForegroundHex) {
			return nil, fmt.Errorf("%s is not a valid checkbox background hex color for %s", checkboxForegroundHex, bossName)
		}
		headerBackgroundRgba, err := hex2rgba(headerBackgroundHex)
		if err != nil {
			return nil, err
		}
		headerForegroundRgba, err := hex2rgba(headerForegroundHex)
		if err != nil {
			return nil, err
		}
		checkboxBackgroundRgba, err := hex2rgba(checkboxBackgroundHex)
		if err != nil {
			return nil, err
		}
		checkboxForegroundRgba, err := hex2rgba(checkboxForegroundHex)
		if err != nil {
			return nil, err
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

		oldCdataMap := columnDataMaps[expansion]
		cdata := &ColumnData{
			Name:         ColumnName(bossName),
			HeaderFormat: headerCellFormat,
			ColumnFormat: columnFormat,
		}
		cindex := ColumnIndex(columnIndexAfterCommonIndices + 2)
		if oldCdataMap == nil {
			columnDataMaps[expansion] = make(map[ColumnIndex]*ColumnData)
			columnDataMaps[expansion][cindex] = cdata
		} else {
			oldCdataMap[ColumnIndex(cindex)] = cdata
			columnDataMaps[expansion] = oldCdataMap
		}

		innerMap[SheetIndex(expansion)] = columnDataMaps[expansion]
	}
	mapping := &ColumnMap{Mapping: innerMap}

	headerBackgroundHex := "#d9d9d9"
	headerBackgroundRgba, err := hex2rgba(headerBackgroundHex)
	if err != nil {
		return nil, err
	}
	headerForegroundHex := "#4d4d4d"
	headerForegroundRgba, err := hex2rgba(headerForegroundHex)
	if err != nil {
		return nil, err
	}
	columnBackgroundHex := "#e6e6e6"
	columnBackgroundRgba, err := hex2rgba(columnBackgroundHex)
	if err != nil {
		return nil, err
	}
	columnForegroundHex := "#595959"
	columnForegroundRgba, err := hex2rgba(columnForegroundHex)
	if err != nil {
		return nil, err
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
	for i := 0; i < NumberOfExpansions; i++ {
		mapping.Mapping[SheetIndex(i)][0] = &ColumnData{
			Name:         ColumnName("Row ID"),
			HeaderFormat: headerCellFormat,
			ColumnFormat: columnFormat,
		}
		mapping.Mapping[SheetIndex(i)][1] = &ColumnData{
			Name:         ColumnName("Name"),
			HeaderFormat: headerCellFormat,
			ColumnFormat: columnFormat,
		}
	}
	return mapping, nil
}

func (m *ColumnMap) GetSheetNames() ([]string, error) {
	numSheets := len(m.Mapping)
	sheetNames := make([]string, numSheets)
	for nameIndex, _ := range m.Mapping {
		name, err := ExpansionToExpansionName(Expansion(nameIndex))
		if err != nil {
			return nil, err
		}
		sheetNames[nameIndex] = string(name)
	}
	return sheetNames, nil
}
