package main

import (
	"fmt"
	"regexp"
	"strconv"

	"google.golang.org/api/sheets/v4"
)

func toRgbaRatio(num int64) float64 {
	return float64(num) / float64(255)
}

func isHex(hex string) bool {
	rexp := regexp.MustCompile(`^#[a-fA-F0-9]{6}$|^#[a-fA-F0-9]{8}$`)
	return rexp.Match([]byte(hex))
}

type RGBA struct {
	Red   int64
	Green int64
	Blue  int64
	Alpha float64
}

func (c *RGBA) ToGoogleSheetsColor() *sheets.Color {
	return &sheets.Color{
		Red:   toRgbaRatio(c.Red),
		Green: toRgbaRatio(c.Green),
		Blue:  toRgbaRatio(c.Blue),
		Alpha: c.Alpha,
	}
}

func hex2rgba(hex string) (*RGBA, error) {
	rHex := hex[1:3]
	gHex := hex[3:5]
	bHex := hex[5:7]
	var aHex string

	hexLen := len(hex)
	if hexLen-1 == 8 {
		aHex = hex[7:9]
	} else if hexLen-1 == 6 {
		aHex = "FF"
	} else {
		return nil, fmt.Errorf("%s is not an 8- or 6-hex number", hex)
	}

	rRgba, err := strconv.ParseInt(rHex, 16, 64)
	if err != nil {
		return nil, fmt.Errorf("strconv.ParseInt() 1 error: [%w]", err)
	}
	gRgba, err := strconv.ParseInt(gHex, 16, 64)
	if err != nil {
		return nil, fmt.Errorf("strconv.ParseInt() 2 error: [%w]", err)
	}
	bRgba, err := strconv.ParseInt(bHex, 16, 64)
	if err != nil {
		return nil, fmt.Errorf("strconv.ParseInt() 3 error: [%w]", err)
	}
	aRgbaInt, err := strconv.ParseInt(aHex, 16, 64)
	if err != nil {
		return nil, fmt.Errorf("strconv.ParseInt() 4 error: [%w]", err)
	}
	aRgba := float64(aRgbaInt) / float64(255)
	return &RGBA{
		Red:   rRgba,
		Green: gRgba,
		Blue:  bRgba,
		Alpha: aRgba,
	}, nil
}
