package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"

	"google.golang.org/api/drive/v3"
)

func isGmailEmailAddress(email string) bool {
	rexp := regexp.MustCompile(`^.+@gmail.com$`)
	return rexp.Match([]byte(email))
}

func GetPermissions(mountSpreadsheetPermissionsFilepath string) ([]*drive.Permission, error) {
	file, err := os.Open(mountSpreadsheetPermissionsFilepath)
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

	perms := []*drive.Permission{}
	for {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		email := row[0]
		if !isGmailEmailAddress(email) {
			return nil, fmt.Errorf("%s is not a valid gmail address", email)
		}
		permType := row[1]
		role := row[2]
		perms = append(perms, &drive.Permission{
			Type:         permType,
			Role:         role,
			EmailAddress: email,
		})
	}
	return perms, nil
}
