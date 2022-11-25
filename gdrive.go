package main

import (
	"github.com/disgoorg/log"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func fileExists(fileid string) bool {
	svc, err := drive.NewService(ctx, option.WithHTTPClient(gclient))
	if err != nil {
		log.Error(err)
	}

	f, err := svc.Files.Get(fileid).Do()
	return f != nil && err != nil
}

func makeFile(title string) string {
	svc, err := drive.NewService(ctx, option.WithHTTPClient(gclient))
	if err != nil {
		log.Error(err)
	}

	file := &drive.File{
		MimeType:        fileMimeType,
		Name:            title,
		WritersCanShare: true,
	}
	f, err := svc.Files.Create(file).Do()
	if err != nil {
		log.Error(err)
	}
	return f.Id
}

// Type, Role, and EmailAddress are required properties for perm
func addFilePermmission(fileid string, perm *drive.Permission) string {
	svc, err := drive.NewService(ctx, option.WithHTTPClient(gclient))
	if err != nil {
		log.Error(err)
	}

	p, err := svc.Permissions.Create(fileid, perm).Do()
	if err != nil {
		log.Error(err)
	}
	return p.Id
}
