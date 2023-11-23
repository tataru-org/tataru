package main

import (
	"fmt"

	"google.golang.org/api/drive/v3"
)

const fileMimeType = "application/vnd.google-apps.spreadsheet"

type FileID string
type PermissionID string

func fileExists(fileId FileID) (*bool, error) {
	f, err := gdriveSvc.Files.Get(string(fileId)).SupportsAllDrives(true).Do()
	if err != nil {
		return nil, fmt.Errorf("gdriveSvc.Files.Get() error: [%w]", err)
	}
	exists := f != nil
	return &exists, nil
}

func createFile(title string) (*FileID, error) {
	file := &drive.File{
		MimeType:        fileMimeType,
		Name:            title,
		WritersCanShare: true,
		Parents:         []string{botConfig.GoogleDriveDestinationFolderId},
	}
	f, err := gdriveSvc.Files.Create(file).SupportsAllDrives(true).Do()
	if err != nil {
		return nil, fmt.Errorf("gdriveSvc.Files.Create() error: [%w]", err)
	}
	fid := FileID(f.Id)
	return &fid, err
}
