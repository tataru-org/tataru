package main

import (
	"google.golang.org/api/drive/v3"
)

const fileMimeType = "application/vnd.google-apps.spreadsheet"

type FileID string
type SheetID int64
type PermissionID string

func fileExists(fileId FileID) (*bool, error) {
	f, err := gdriveSvc.Files.Get(string(fileId)).SupportsAllDrives(true).Do()
	if err != nil {
		return nil, err
	}
	exists := f != nil
	return &exists, nil
}

func createFile(title string) (*FileID, error) {
	file := &drive.File{
		MimeType:        fileMimeType,
		Name:            title,
		WritersCanShare: true,
	}
	f, err := gdriveSvc.Files.Create(file).SupportsAllDrives(true).Do()
	if err != nil {
		return nil, err
	}
	fid := FileID(f.Id)
	return &fid, err
}
