package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"google.golang.org/api/drive/v3"
)

func isGmailEmailAddress(email string) bool {
	rexp := regexp.MustCompile(`^.+@gmail.com$`)
	return rexp.Match([]byte(email))
}

func GetPermissions(mountSpreadsheetPermissionsFilepath string) ([]*drive.Permission, error) {
	rawJson, err := os.ReadFile(mountSpreadsheetPermissionsFilepath)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile() error: [%w]", err)
	}
	var perms []*drive.Permission
	err = json.Unmarshal(rawJson, &perms)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal() error: [%w]", err)
	}
	return perms, nil
}
