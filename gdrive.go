package main

import (
	"google.golang.org/api/drive/v3"
)

const fileMimeType = "application/vnd.google-apps.spreadsheet"

type FileID string
type SheetID int64
type PermissionID string

func fileExists(fileId FileID) (*bool, error) {
	f, err := gdriveSvc.Files.Get(string(fileId)).Do()
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
	f, err := gdriveSvc.Files.Create(file).Do()
	if err != nil {
		return nil, err
	}
	fid := FileID(f.Id)
	return &fid, err
}

// Type, Role, and EmailAddress are required properties for perm
func addFilePermmission(fileId FileID, perm *drive.Permission) (*PermissionID, error) {
	p, err := gdriveSvc.Permissions.Create(string(fileId), perm).Do()
	if err != nil {
		return nil, err
	}
	pid := PermissionID(p.Id)
	return &pid, err
}

func removeFilePermission(fileId FileID, permId PermissionID) error {
	err := gdriveSvc.Permissions.Delete(string(fileId), string(permId)).Do()
	if err != nil {
		return err
	}
	return nil
}
